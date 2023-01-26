"""Test miscellaneous bidder endpoints."""

import pytest

from tests.bidder_client import Bidder


@pytest.mark.parametrize("method", ("GET", "HEAD"))
def test_healthz_ok(bidder: Bidder, method: str) -> None:
    """Test that the /healthz endpoint returns an empty 200 response on a GET or HEAD request."""
    response = bidder.request_external(method, "/healthz", status=200)
    assert response.content == b""  # nosec


def test_healthz_method_not_supported(bidder: Bidder) -> None:
    """Test that the /healthz endpoint does not support POST requests."""
    bidder.request_external("POST", "/healthz", status=405)


def test_healthz_not_internal(bidder: Bidder) -> None:
    """Test that the /healthz endpoint is not available on the internal endpoint."""
    bidder.request_internal("GET", "/healthz", status=404)


def test_profiler_endpoints(bidder: Bidder) -> None:
    """Test that profiler endpoints return non-empty responses."""
    for endpoint in "profile", "allocs", "heap":
        response = bidder.request_internal(
            "GET", f"/debug/pprof/{endpoint}?seconds=1", status=200
        )
        assert len(response.content) > 0  # nosec
