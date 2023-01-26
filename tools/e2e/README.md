# End to end tests

This package provides bidder end to end tests and code for running them against a Kubernetes cluster and deploying a
test instance of the bidder.

## Dependencies

These tools are needed:

* Python 3.9 (we use [pyenv](https://github.com/pyenv/pyenv) to install a specific version)
* Poetry (install via `pip install poetry`)
* pre-commit (install via `pip install pre-commit`)
* kubectl, helm: as needed for bidder deployments

You also need configured cluster access and AWS credentials for DynamoDB access.

To workaround Aerospike Python client setup script attempting to download the C client in a build specific to the used
distribution version, on Debian Sid pretend that you run the supported version:

    export VERSION_ID=10

Install test dependencies via Poetry, running in the `tools/e2e` directory:

    poetry install

## Invoking tests

Against the bidder stack:

    poetry run pytest -x -m 'not dax and not aerospike'

Against a minikube cluster (with all the needed images loaded into cache):

    poetry run pytest --bidder-values-yaml=../../deployment/infrastructure/deployment/bidder/overlay-minikube.yaml --bidder-values-yaml=../../../aws-config.yaml -m 'not dax'

where your `../../../aws-config.yaml` specifies the `AWS_SECRET_ACCESS_KEY` and `AWS_ACCESS_KEY_ID` `config` keys.

(Tests using DAX can be invoked only within the VPC: the easiest way to do it is to let the CI run them.)

## Developing tests

Before running any test make sure that you selected cluster which the tests should be run on.

    make eks@use

We use many linters. Have pre-commit configured to invoke them automatically:

    pre-commit install

Otherwise run them on all files manually:

    pre-commit run -a

Coverage of the test code is measured when running on CI. Run locally with a coverage report:

    poetry run pytest --cov

If tests succeed, no branch should be left uncovered. Use the `# pragma: no cover` comments for intentionally uncovered
branches: these are normally error handling code, or for special cases of test runs not on CI environment, or for
handling nondeterministic application behaviour. If not clear why it's used, add a separate explanation comment.

When writing a new test, check the existing fixtures. Remember that pytest runs fixture setup left to right: fixtures
with mutable state changed before other fixtures set up are needed to configure the bidder for the specific test. Use
test parametrization to check the same behaviour across different setups: e.g. for different database backends.
