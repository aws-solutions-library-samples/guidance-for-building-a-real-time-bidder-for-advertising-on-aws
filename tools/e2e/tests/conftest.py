"""Fixtures."""

import contextlib
import logging
import os
import random
import signal
import subprocess  # nosec
from pathlib import Path
from typing import Any
from typing import Iterator
from uuid import UUID
from uuid import uuid4

import _pytest.config
import _pytest.config.argparsing
import _pytest.fixtures
import boto3
import pytest
import waiting
import yaml
from aerospike import client as aerospike_client
from amazondax import AmazonDaxClient
from pexpect import popen_spawn

from tests.bidder_client import Bidder
from tests.db_client import AerospikeClient
from tests.db_client import DatabaseClient
from tests.db_client import DAXClient
from tests.db_client import DynamoDBClient

TIMEOUT_SECONDS = 60


def pytest_addoption(
    parser: _pytest.config.argparsing.Parser,
    pluginmanager: _pytest.config.PytestPluginManager,
) -> None:
    """Add custom options to pytest."""
    parser.addoption(
        "--stack-name",
        help="Name of the CloudFormation stack we are testing on",
    )
    parser.addoption(
        "--bidder-values-yaml",
        action="append",
        help="Path to an additional values.yaml file for the bidder to use "
        "before the test-specific overrides.",
    )
    parser.addoption(
        "--bidder-image-tag",
        default="latest",
        help="Tag of the bidder image to test (must be compatible with the current bidder chart)",
    )
    parser.addoption(
        "--aerospike-host",
        default="aerospike-e2e",
        help="Aerospike server host for use in tests",
    )
    parser.addoption(
        "--aerospike-namespace",
        default="bidder",
        help="Name of the Aerospike namespace for use in tests",
    )


@pytest.fixture(scope="session")
def dynamodb() -> Iterator[Any]:
    """
    Return the DynamoDB service resource.

    This needs AWS credentials.
    """
    # Do not log many lines about request signing.
    logging.getLogger("botocore").setLevel(logging.INFO)
    resource = boto3.resource("dynamodb")
    yield resource
    # Workaround <https://github.com/boto/boto3/issues/454>.
    resource.meta.client._endpoint.http_session._manager.clear()  # pylint:disable=protected-access


@pytest.fixture(scope="session")
def dax(stack_name: str, aws_region: str) -> Any:
    """
    Return the DAX service resource.

    This needs AWS credentials and works only on a cluster in the VPC with DAX.
    """
    return AmazonDaxClient.resource(
        endpoint_url=f"dax.{stack_name}.{aws_region}.ab.clearcode.cc:8111"
    )


@pytest.fixture(scope="session")
def aerospike(pytestconfig: _pytest.config.Config) -> Any:
    """Return the Aerospike client instance."""
    return aerospike_client(
        {"hosts": [(pytestconfig.getoption("aerospike_host"), 3000)]}
    ).connect()


@pytest.fixture(scope="session")
def aws_region(dynamodb: Any) -> str:
    """Return the AWS region name used for our DynamoDB connection."""
    return dynamodb.meta.client.meta.config.region_name


@pytest.fixture(scope="session")
def stack_name(pytestconfig: _pytest.config.Config) -> str:
    """Return the cluster stack name."""
    return pytestconfig.getoption("stack_name", "") or "bidder"


@pytest.fixture(scope="session")
def ecr_registry(aws_region: str) -> str:
    """
    Return the ECR registry domain.

    We obtain it either via the ``ECR_REGISTRY`` variable (easy to pass in chart
    deployments) or ``AWS_ACCOUNT`` (set for local development).
    """
    ecr_registry = os.getenv("ECR_REGISTRY", "")
    if not ecr_registry:  # pragma: no cover
        aws_account = os.getenv("AWS_ACCOUNT", "")
        assert (  # nosec
            aws_account
        ), "AWS_ACCOUNT environment variable must be set to your AWS account ID"
        ecr_registry = f"{aws_account}.dkr.ecr.{aws_region}.amazonaws.com"
    return ecr_registry


@pytest.fixture
def values_yaml(
    pytestconfig: _pytest.config.Config,
    stack_name: str,
    ecr_registry: str,
    aws_region: str,
) -> dict:
    """
    Return a bidder chart ``values.yaml`` override.

    Fixtures can mutate it (independently for each test) before installing the bidder.
    """
    aws_config = {
        key: value for (key, value) in os.environ.items() if key.startswith("AWS_")
    }
    return {
        "stackName": stack_name,
        "awsRegion": aws_region,
        "image": {
            "registry": ecr_registry,
            "tag": pytestconfig.getoption("bidder_image_tag"),
        },
        # Make the external service instead work as an internal one on port 8090: tests running
        # outside the cluster wouldn't be able to access an internal AWS load balancer.
        "service": {
            "type": "ClusterIP",
            "port": "8090",
            "targetPort": "8090",
            "annotations": {
                # Delete default annotations for the AWS load balancer.
                "service.beta.kubernetes.io/aws-load-balancer-ssl-ports": None,
                "service.beta.kubernetes.io/aws-load-balancer-backend-protocol": None,
                "service.beta.kubernetes.io/aws-load-balancer-type": None,
                "service.beta.kubernetes.io/aws-load-balancer-internal": None,
            },
        },
        # Keep default internal service on port 8091.
        "serviceInternal": {
            "type": "ClusterIP",
            "port": "8091",
            "targetPort": "8091",
        },
        "config": {
            # We need to know if e.g. a campaign is out of budget or the request has timed out.
            "LOG_LEVEL": "trace",
            # Use DynamoDB tables for e2e tests.
            "DYNAMODB_DEVICE_TABLE": "e2e-device",
            "DYNAMODB_AUDIENCE_TABLE": "e2e-audience",
            "DYNAMODB_CAMPAIGN_TABLE": "e2e-campaign",
            "DYNAMODB_BUDGET_TABLE": "e2e-budget",
            # And Aerospike cluster for e2e tests.
            "AEROSPIKE_HOST": pytestconfig.getoption("aerospike_host"),
            "AEROSPIKE_NAMESPACE": pytestconfig.getoption("aerospike_namespace"),
            "AEROSPIKE_BUDGET_GET_TIMEOUT": "20s",
            # Should be high enough to certainly bid if allowed to, unless the test overrides it to
            # timeout.
            "BIDREQUEST_TIMEOUT": "10s",
            # Only specific tests enable it.
            "KINESIS_DISABLE": "false",
            # 3.0 except for tests checking switching the default to 2.5.
            "BIDREQUEST_OPEN_RTB_VERSION": "3.0",
        }
        | aws_config,
        # Disable to not need a Prometheus installation.
        "serviceMonitor": {"enabled": False},
    }


@pytest.fixture
def values_yaml_path(tmp_path: Path, values_yaml: dict) -> Path:
    """Return a path to a temporary file with rendered *values_yaml*."""
    path = tmp_path / "values.yaml"
    with path.open("w") as file_object:
        yaml.dump(values_yaml, file_object)
    return path


@contextlib.contextmanager
def port_forward(resource: str, pod_port: int) -> Iterator[int]:
    """Run port forwarding, yielding host port."""
    child = popen_spawn.PopenSpawn(
        ["kubectl", "port-forward", resource, f":{pod_port}"]
    )
    try:
        assert (  # nosec
            child.expect(
                f"Forwarding from 127\\.0\\.0\\.1:(\\d+) -> {pod_port}",
                timeout=TIMEOUT_SECONDS,
            )
            == 0
        )
        yield int(child.match.group(1))
    finally:
        child.kill(signal.SIGTERM)
        try:
            waiting.wait(
                lambda: child.proc.poll() is not None, timeout_seconds=TIMEOUT_SECONDS
            )
        except waiting.TimeoutExpired:  # pragma: no cover
            child.kill(signal.SIGKILL)
        child.wait()
        # Do not leak file descriptors.
        for file_object in child.proc.stdin, child.proc.stdout, child.proc.stderr:
            if file_object is not None:
                file_object.close()


@pytest.fixture(scope="session")
def bidder_name() -> str:
    """Return a random name for the bidder chart installation."""
    return f"bidder-e2e-{uuid4()}"


@pytest.fixture(scope="session")
def bidder_cleanup(bidder_name: str) -> Iterator[list[bool]]:
    """
    Uninstall the bidder after tests if the first item of the returned list is set to true.

    This allows uninstalling the bidder only if it was installed and only once per multiple tests.
    """
    flag = [False]
    yield flag
    if not flag[0]:  # pragma: no cover
        # The bidder wasn't installed (maybe something failed earlier), skip the uninstallation.
        return
    subprocess.check_call(["helm", "uninstall", bidder_name])  # nosec


@pytest.fixture
def bidder(
    pytestconfig: _pytest.config.Config,
    values_yaml_path: Path,
    bidder_name: str,
    bidder_cleanup: list[bool],
    request: _pytest.fixtures.FixtureRequest,
    db_client: DatabaseClient,
) -> Iterator[Bidder]:
    """Run bidder and return its client."""
    values_yamls = (pytestconfig.getoption("bidder_values_yaml") or []) + [
        values_yaml_path
    ]
    helm_args: list[str] = [
        "helm",
        "upgrade",
        "--install",
        "--description",
        request.node.name,
        bidder_name,
        str(
            Path(__file__).parent.parent.parent.parent
            / "infrastructure"
            / "charts"
            / "bidder"
        ),
    ] + [arg for path in values_yamls for arg in ["-f", path]]
    # Install bidder.
    subprocess.check_call(helm_args)  # nosec
    bidder_cleanup[0] = True
    try:
        # Ensure all bidder pods are replaced.
        subprocess.check_call(  # nosec
            [
                "kubectl",
                "scale",
                "deployment",
                bidder_name,
                "--replicas=0",
                f"--timeout={TIMEOUT_SECONDS}s",
            ]
        )
        subprocess.check_call(  # nosec
            [
                "kubectl",
                "scale",
                "deployment",
                bidder_name,
                "--replicas=1",
                f"--timeout={TIMEOUT_SECONDS}s",
            ]
        )
        subprocess.check_call(  # nosec
            [
                "kubectl",
                "wait",
                "--for=condition=Available",
                f"--timeout={TIMEOUT_SECONDS}s",
                f"deployment/{bidder_name}",
            ]
        )
        # Forward ports.
        with port_forward(f"svc/{bidder_name}", 8090) as external_port, port_forward(
            f"svc/{bidder_name}-internal", 8091
        ) as internal_port:
            yield Bidder(
                external_url=f"http://127.0.0.1:{external_port}/",
                internal_url=f"http://127.0.0.1:{internal_port}/",
            )

    finally:
        # Write bidder logs.
        subprocess.check_call(  # nosec
            [
                "kubectl",
                "logs",
                "--tail=-1",
                "-l",
                f"app.kubernetes.io/instance={bidder_name}",
            ]
        )
        # Describe pods.
        subprocess.check_call(  # nosec
            [
                "kubectl",
                "describe",
                "pod",
                "-l",
                f"app.kubernetes.io/instance={bidder_name}",
            ]
        )


@pytest.fixture
def bid_request_id() -> str:
    """
    Return the bid request ID used for the test.

    It can be used to distinguish e.g. stream records referring to our bid request from ones
    produced in concurrent tests.
    """
    return str(uuid4())


@pytest.fixture
def bid_request30(bid_request_id: str) -> dict:
    """
    Return a bid request as a dict.

    It's a valid, complete with the usual fields, bid request that should bid on sample data.
    """
    return {
        "openrtb": {
            "ver": "3.0",
            "domainspec": "adcom",
            "domainver": "1.0",
            "request": {
                "id": bid_request_id,
                "tmax": 51,
                "at": 1,
                "cur": ["USD"],
                "source": {
                    "tid": "1lmo0hJYZX5eH3BPmqHSVzYfSGa",
                    "ts": 1611320889,
                    "pchain": "1lmo0cdhb6woJTWl0Bouj5dXR5b",
                },
                "item": [
                    {
                        "id": "1nQSb4hpjPgxAINsLWdBhJZmkUu",
                        "qty": 2,
                        "flr": 3.58,
                        "flrcur": "USD",
                        "exp": 4,
                        "spec": {
                            "placement": {
                                "tagid": "1nQSb6LpN743FfEwedJnWrv0zen",
                                "ssai": 1,
                                "display": {
                                    "mime": "text/html",
                                    "w": 1767,
                                    "h": 1092,
                                    "unit": 1,
                                },
                            }
                        },
                    }
                ],
                "context": {
                    "site": {
                        "domain": "example.com",
                        "page": "http://easy.example.com/easy?cu=13824;cre=mu;target=_blank",
                        "ref": "http://tpc.googlesyndication.com/pagead/js/loader12.html"
                        "?http://sdk.streamrail.com/vpaid/js/668/sam.js",
                        "cat": [
                            "1",
                            "33",
                            "544",
                            "765",
                            "1222",
                            "1124",
                            "789",
                            "995",
                            "133",
                            "45",
                            "76",
                            "91",
                        ],
                        "pub": {
                            "id": "qqwer1234xgfd",
                            "name": "site_name",
                            "domain": "my.site.com",
                        },
                    },
                    "user": {
                        "id": "1nQSb4rMriy3eo43fZIrR74qGjs",
                        "buyeruid": "1nQSb9fP14KbRLJiOv7K01Fg6rx",
                        "yob": 1961,
                        "gender": "F",
                        "data": [
                            {
                                "id": "pub-demographics",
                                "name": "data_name",
                                "segment": [
                                    {
                                        "id": "345qw245wfrtgwertrt56765wert",
                                        "name": "segment_name",
                                        "value": "segment_value",
                                    }
                                ],
                            }
                        ],
                    },
                    "device": {
                        "type": 2,
                        "ifa": "a64140cbff371cdcd47d70e103995b46",
                        "os": 2,
                        "lang": "en",
                        "osv": "10",
                        "model": "browser",
                        "make": "desktop",
                        "ip": "104.4.9.67",
                        "ua": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
                        "(KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36",
                        "geo": {
                            "lat": 37.789,
                            "lon": -122.394,
                            "country": "USA",
                            "city": "San Francisco",
                            "region": "CA",
                            "zip": "94105",
                            "type": 2,
                            "ext": {
                                "dma": 650,
                                "state": "oklahoma",
                                "continent": "north america",
                            },
                        },
                    },
                    "restrictions": {
                        "cattax": 2,
                        "bcat": [
                            "12",
                            "143",
                            "34",
                            "887",
                            "122",
                            "999",
                            "1023",
                            "13",
                            "4",
                            "565",
                            "920",
                            "224",
                            "857",
                            "1320",
                        ],
                        "badv": [
                            "facebook.com",
                            "twitter.com",
                            "google.com",
                            "amazon.com",
                            "youtube.com",
                        ],
                    },
                    "regs": {"coppa": 0, "gdpr": 0, "ext": {"sb568": 0}},
                },
            },
        }
    }


@pytest.fixture
def bid_request25(bid_request_id: str) -> dict:
    """
    Return an OpenRTB 2.5 bid request as a dict.

    It's a valid, complete with the usual fields, bid request that should bid on sample data.
    """
    return {
        "at": 1,
        "badv": [
            "facebook.com",
            "twitter.com",
            "google.com",
            "amazon.com",
            "youtube.com",
        ],
        "bcat": [
            "12",
            "143",
            "34",
            "887",
            "122",
            "999",
            "1023",
            "13",
            "4",
            "565",
            "920",
            "224",
            "857",
            "1320",
        ],
        "cur": ["USD"],
        "device": {
            "devicetype": 2,
            "ifa": "a64140cbff371cdcd47d70e103995b46",
            "ip": "104.4.9.67",
            "language": "en",
            "make": "desktop",
            "model": "browser",
            "os": "iOS",
            "osv": "10",
            "ua": (
                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
                " (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36"
            ),
        },
        "ext": {"cattax": 2},
        "id": bid_request_id,
        "imp": [
            {
                "banner": {
                    "ext": {"qty": 2, "unit": 1},
                    "h": 1092,
                    "mimes": "text/html",
                    "w": 1767,
                },
                "bidfloor": 3.58,
                "bidfloorcur": "USD",
                "exp": 4,
                "ext": {"ssai": 1},
                "id": "1nQSb4hpjPgxAINsLWdBhJZmkUu",
                "tagid": "1nQSb6LpN743FfEwedJnWrv0zen",
            }
        ],
        "regs": {"coppa": 0, "ext": {"gdpr": 0, "sb568": 0}},
        "site": {
            "cat": [
                "1",
                "33",
                "544",
                "765",
                "1222",
                "1124",
                "789",
                "995",
                "133",
                "45",
                "76",
                "91",
            ],
            "domain": "example.com",
            "page": "http://easy.example.com/easy?cu=13824;cre=mu;target=_blank",
            "publisher": {
                "domain": "my.site.com",
                "id": "qqwer1234xgfd",
                "name": "site_name",
            },
            "ref": (
                "http://tpc.googlesyndication.com/pagead/js/loader12.html"
                "?http://sdk.streamrail.com/vpaid/js/668/sam.js"
            ),
        },
        "source": {
            "ext": {"ts": 1611320889},
            "pchain": "1lmo0cdhb6woJTWl0Bouj5dXR5b",
            "tid": "1lmo0hJYZX5eH3BPmqHSVzYfSGa",
        },
        "tmax": 51,
        "user": {
            "buyeruid": "1nQSb9fP14KbRLJiOv7K01Fg6rx",
            "data": [
                {
                    "id": "pub-demographics",
                    "name": "data_name",
                    "segment": [
                        {
                            "id": "345qw245wfrtgwertrt56765wert",
                            "name": "segment_name",
                            "value": "segment_value",
                        }
                    ],
                }
            ],
            "gender": "F",
            "geo": {
                "city": "San Francisco",
                "country": "USA",
                "ext": {"continent": "north america", "dma": 650, "state": "oklahoma"},
                "lat": 37.789,
                "lon": -122.394,
                "region": "CA",
                "type": 2,
                "zip": "94105",
            },
            "id": "1nQSb4rMriy3eo43fZIrR74qGjs",
            "yob": 1961,
        },
    }


@pytest.fixture(
    params=["dynamodb", "dynamodb-lowlevel", "dax", "aerospike-get", "aerospike-scan"]
)
def database_type(request: _pytest.fixtures.SubRequest, values_yaml: dict) -> str:
    """
    Configure what database the bidder uses and how.

    These chooses among alternative databases and bidder clients. Each test using this fixture runs
    separately for each configuration, ensuring equivalent bidder behaviour.
    """
    values_yaml["config"]["DYNAMODB_ENABLE_LOW_LEVEL"] = (
        "true" if request.param == "dynamodb-lowlevel" else "false"
    )
    values_yaml["config"]["DAX_ENABLE"] = "true" if request.param == "dax" else "false"
    values_yaml["config"]["DATABASE_CLIENT"] = (
        "aerospike"
        if request.param in ("aerospike-scan", "aerospike-get")
        else "dynamodb"
    )
    values_yaml["config"]["AEROSPIKE_DISABLE_SCAN"] = (
        "true" if request.param == "aerospike-get" else "false"
    )
    values_yaml["config"]["AEROSPIKE_BUDGET_BATCHES_KEY"] = request.getfixturevalue(
        "budget_batches_key"
    )
    return request.param


def pytest_collection_modifyitems(
    session: pytest.Session, config: _pytest.config.Config, items: list[pytest.Item]
) -> None:
    """Add markers for tests using DAX."""
    for item in items:
        database_type = ""
        for node in item.listchain():
            try:
                database_type = node.callspec.params["database_type"]  # type: ignore[attr-defined]
            except (AttributeError, KeyError):
                pass
        if database_type == "dax":
            item.add_marker(pytest.mark.dax)
        if database_type in ("aerospike-get", "aerospike-scan"):
            item.add_marker(pytest.mark.aerospike)


@pytest.fixture
def db_client(
    request: _pytest.fixtures.SubRequest, database_type: str
) -> Iterator[DatabaseClient]:
    """Yield a function that puts items in DynamoDB and deletes them after the test."""
    if database_type == "dax":
        client: DatabaseClient = DAXClient(request)
    elif database_type in ("aerospike-scan", "aerospike-get"):
        client = AerospikeClient(request)
    else:
        client = DynamoDBClient(request)

    with contextlib.closing(client):
        yield client


@pytest.fixture
def campaign(db_client: DatabaseClient) -> bytes:
    """Insert a sample campaign into DynamoDB returning its ID."""
    campaign_id = uuid4().bytes
    db_client.put_campaign(campaign_id, bid_price=1_000_000)
    return campaign_id


@pytest.fixture
def budget(db_client: DatabaseClient, campaign: bytes) -> int:
    """Insert a budget for the sample campaign into a database."""
    budget_id = random.getrandbits(31)
    db_client.put_budget(budget_id, campaign, 10_000_000)
    return budget_id


@pytest.fixture
def audience(db_client: DatabaseClient, campaign: bytes) -> bytes:
    """Insert a sample audience into DynamoDB returning its ID."""
    audience_id = uuid4().bytes
    db_client.put_audience(audience_id, {campaign})
    return audience_id


@pytest.fixture
def device(db_client: DatabaseClient, audience: bytes) -> bytes:
    """Insert a sample device into DynamoDB returning its ID."""
    device_id = uuid4().bytes
    db_client.put_device(device_id, {audience})
    return device_id


@pytest.fixture
def present_ifa(budget: bytes, device: bytes) -> str:
    """Return a formatted device ID present in DynamoDB."""
    return str(UUID(bytes=device))


@pytest.fixture
def absent_ifa() -> str:
    """Return a formatted device ID hopefully absent from DynamoDB."""
    return str(uuid4())


@pytest.fixture
def short_timeout(values_yaml: dict) -> None:
    """Set a short bid request timeout."""
    values_yaml["config"]["BIDREQUEST_TIMEOUT"] = "1ns"


@pytest.fixture
def budget_batches_key() -> str:
    """Return randomized key value for budget batches."""
    return f"budget_batches_key_{str(uuid4())}"
