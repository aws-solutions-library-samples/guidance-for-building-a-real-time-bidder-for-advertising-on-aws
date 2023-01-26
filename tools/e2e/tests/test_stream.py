"""Tests checking if bid requests and responses are logged."""

import json
import threading
from base64 import b64decode
from typing import Iterator

import pytest
import waiting
from aws_kinesis_agg.deaggregator import iter_deaggregate_records
from kinesis.consumer import KinesisConsumer
from zstandard import ZstdDecompressor

from tests.bidder_client import Bidder

# We parametrize the tests with ``database_type`` to run them only with DynamoDB: streaming is the
# same regardless of the used database (if the bidder bids as tested elsewhere), so choose the
# simplest one.
pytestmark = pytest.mark.parametrize("database_type", ("dynamodb",))


@pytest.fixture
def enable_kinesis(values_yaml: dict) -> None:
    """Enable Kinesis integration."""
    values_yaml["config"]["KINESIS_DISABLE"] = "false"


@pytest.fixture
def kinesis_user_records(stack_name: str, bid_request_id: str) -> Iterator[list[dict]]:
    """
    Run a Kinesis consumer and return a list of received messages.

    Only messages relating to our bid request or its bid responses are returned: otherwise we would
    have no way of handling concurrent test runs on the shared Kinesis stream.

    The list is asynchronously updated when new messages are obtained.
    """
    consumer = KinesisConsumer(f"{stack_name}-bids")

    consumed = []

    def consume() -> None:
        """Consume messages until the consumer is stopped."""
        decompressor = ZstdDecompressor()
        for record in iter_deaggregate_records(consumer, data_format="Boto3"):
            record_data = json.loads(
                decompressor.decompress(
                    b64decode(record["kinesis"]["data"].decode("utf-8"))
                )
            )
            if (
                record_data.get("openrtb", {}).get("request", {}).get("id", "")
                == bid_request_id
                or record_data.get("openrtb", {}).get("response", {}).get("id", "")
                == bid_request_id
                or record_data.get("id", "") == bid_request_id
            ):
                consumed.append(record_data)

    thread = threading.Thread(target=consume)
    thread.start()

    yield consumed

    consumer.run = False
    thread.join()
    # <https://github.com/boto/boto3/issues/454>
    client = consumer.kinesis_client
    client._endpoint.http_session._manager.clear()  # pylint:disable=protected-access


def test_stream_kinesis_enabled_no_bid(
    database_type: str,
    enable_kinesis: None,
    kinesis_user_records: list[dict],
    bid_request30: dict,
    bidder: Bidder,
) -> None:
    """Test that bid request is streamed if the bidder responds with a no bid."""
    # Opt-out device ID, never bids, but has the request streamed.
    bid_request30["openrtb"]["request"]["context"]["device"][
        "ifa"
    ] = "00000000-0000-0000-0000-000000000000"

    bidder.nobid(bid_request30)

    expected_messages = [bid_request30]

    # What can fail here: 1) the thread can get an unhandled exception (pytest reports it) 2) we can
    # log unexpected things (so we need the assert below) 3) we can run this statement before the
    # message is received (so we need the wait).
    try:
        waiting.wait(
            lambda: expected_messages == kinesis_user_records, timeout_seconds=15.0
        )
    except waiting.TimeoutExpired:  # pragma: no cover
        # Get a nice error message from pytest.
        assert expected_messages == kinesis_user_records  # nosec


def test_stream_kinesis_enabled_bid_timeout(
    database_type: str,
    enable_kinesis: None,
    short_timeout: None,
    bid_request30: dict,
    kinesis_user_records: list[dict],
    present_ifa: str,
    bidder: Bidder,
) -> None:
    """Test that bid request is streamed if the bidder times out querying the database."""
    bid_request30["openrtb"]["request"]["context"]["device"]["ifa"] = present_ifa

    bidder.nobid(bid_request30, status=504)

    expected_messages = [bid_request30]

    # What can fail here: 1) the thread can get an unhandled exception (pytest reports it) 2) we can
    # log unexpected things (so we need the assert below) 3) we can run this statement before the
    # message is received (so we need the wait).
    try:
        waiting.wait(
            lambda: expected_messages == kinesis_user_records, timeout_seconds=15.0
        )
    except waiting.TimeoutExpired:  # pragma: no cover
        # Get a nice error message from pytest.
        assert expected_messages == kinesis_user_records  # nosec


def test_stream_kinesis_enabled_invalid_bid_request(
    database_type: str,
    enable_kinesis: None,
    kinesis_user_records: list[dict],
    bid_request30: dict,
    bidder: Bidder,
) -> None:
    """Test that an invalid bid request is not streamed."""
    del bid_request30["openrtb"]["request"]["id"]

    bidder.request_external(
        "POST", "/bidrequest", status=400, data=json.dumps(bid_request30)
    )
    assert kinesis_user_records == []  # nosec


def test_stream_kinesis_enabled_bid_30(
    database_type: str,
    enable_kinesis: None,
    kinesis_user_records: list[dict],
    bid_request30: dict,
    present_ifa: str,
    bidder: Bidder,
) -> None:
    """Test that bid request and bid response are streamed if the bidder bids."""
    bid_request30["openrtb"]["request"]["context"]["device"]["ifa"] = present_ifa
    bid_response = bidder.bid(bid_request30)

    expected_messages = [bid_request30, bid_response]

    # What can fail here: 1) the thread can get an unhandled exception (pytest reports it) 2) we can
    # log unexpected things (so we need the assert below) 3) we can run this statement before the
    # message is received (so we need the wait).
    try:
        waiting.wait(
            lambda: expected_messages == kinesis_user_records, timeout_seconds=15.0
        )
    except waiting.TimeoutExpired:  # pragma: no cover
        # Get a nice error message from pytest.
        assert expected_messages == kinesis_user_records  # nosec


def test_stream_kinesis_enabled_bid_25(
    database_type: str,
    enable_kinesis: None,
    kinesis_user_records: list[dict],
    bid_request25: dict,
    present_ifa: str,
    bidder: Bidder,
) -> None:
    """Test that bid request and bid response are streamed if the bidder bids."""
    bid_request25["device"]["ifa"] = present_ifa
    bid_response = bidder.bid(bid_request25, headers={"x-openrtb-version": "2.5"})

    expected_messages = [bid_request25, bid_response]

    # What can fail here: 1) the thread can get an unhandled exception (pytest reports it) 2) we can
    # log unexpected things (so we need the assert below) 3) we can run this statement before the
    # message is received (so we need the wait).
    try:
        waiting.wait(
            lambda: expected_messages == kinesis_user_records, timeout_seconds=15.0
        )
    except waiting.TimeoutExpired:  # pragma: no cover
        # Get a nice error message from pytest.
        assert expected_messages == kinesis_user_records  # nosec


def test_stream_kinesis_disabled(
    database_type: str,
    bid_request30: dict,
    present_ifa: str,
    kinesis_user_records: list[dict],
    bidder: Bidder,
) -> None:
    """Test that bid request and bid response are not streamed if Kinesis is disabled."""
    bid_request30["openrtb"]["request"]["context"]["device"]["ifa"] = present_ifa

    bidder.bid(bid_request30)

    assert kinesis_user_records == []  # nosec
