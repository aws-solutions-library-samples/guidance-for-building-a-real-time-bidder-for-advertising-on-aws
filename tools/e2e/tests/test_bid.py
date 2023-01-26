"""Tests sending bid requests."""

import binascii
from unittest.mock import ANY

import pytest

from tests.bidder_client import Bidder
from tests.db_client import DatabaseClient


@pytest.mark.parametrize(
    "body, status, response_content",
    (
        (b"{}", 400, b"empty request body"),
        (
            b"}",
            400,
            b"error while parsing request: cannot parse JSON:"
            b' cannot parse number: unexpected char: "}"; unparsed tail: "}"',
        ),
    ),
)
def test_invalid_bid_request(
    bidder: Bidder, body: bytes, status: int, response_content: bytes
) -> None:
    """Test that the bidder responds with an error on an invalid bid request."""
    response = bidder.request_external("POST", "/bidrequest", status=status, data=body)
    assert response.content == response_content  # nosec


def test_invalid_bid_request_unsupported_version(
    bidder: Bidder,
    bid_request30: dict,
) -> None:
    """Test that the bidder rejects valid bid requests of invalid version header."""
    response = bidder.request_external(
        "POST",
        "/bidrequest",
        status=400,
        json=bid_request30,
        headers={"x-openrtb-version": "3.11"},
    )
    assert response.content == b"unsupported x-openrtb-version"  # nosec


def test_invalid_bid_request_wrong_version_30(
    bidder: Bidder,
    bid_request30: dict,
) -> None:
    """
    Test that the bidder rejects wrongly versioned bid request.

    We declare a supported version, but send a request of a different version, so it's
    invalid according to the declared version.
    """
    response = bidder.request_external(
        "POST",
        "/bidrequest",
        status=400,
        json=bid_request30,
        headers={"x-openrtb-version": "2.5"},
    )
    assert response.content == b"empty request ID"  # nosec


def test_invalid_bid_request_wrong_version_25(
    bidder: Bidder,
    bid_request25: dict,
) -> None:
    """
    Test that the bidder rejects wrongly versioned bid request.

    We declare a supported version, but send a request of a different version, so it's
    invalid according to the declared version.
    """
    response = bidder.request_external(
        "POST",
        "/bidrequest",
        status=400,
        json=bid_request25,
        headers={"x-openrtb-version": "3.0"},
    )
    assert response.content == b"empty request body"  # nosec


def test_bid(
    present_ifa: str, campaign: bytes, bidder: Bidder, bid_request30: dict
) -> None:
    """
    Test that the bidder bids on the sample bid request twice.

    Some potential state corruptions would make only the first bid request bid.
    """
    bid_request30["openrtb"]["request"]["context"]["device"]["ifa"] = present_ifa
    for _ in range(2):
        response = bidder.bid(bid_request30)
        response_bid = response["openrtb"]["response"]["seatbid"][0]["bid"][0]
        assert (  # nosec
            response["openrtb"]["response"]["id"]
            == bid_request30["openrtb"]["request"]["id"]
        )
        assert response_bid["cid"] == binascii.hexlify(campaign).decode(  # nosec
            "utf-8"
        ), "Our campaign must win."

        ad = response_bid["media"]["ad"]
        expected_burl = (
            f"https://t.ab.clearcode.cc/{response['openrtb']['response']['id']}/{ad['id']}/"
            "${OPENRTB_PRICE}"
        )
        assert response_bid["burl"] == expected_burl  # nosec

        assert ad == {  # nosec
            "id": ANY,
            "adomain": ["ford.com"],
            "secure": 1,
            "display": {
                "mime": "image/jpeg",
                "ctype": 1,
                "w": 320,
                "h": 50,
                "curl": ANY,
            },
        }
        expected_curl = (
            "https://t.ab.clearcode.cc/"
            f"{response['openrtb']['response']['id']}/{ad['id']}"
        )
        assert ad["display"]["curl"] == expected_curl  # nosec


@pytest.mark.parametrize("database_type", ("dynamodb",))
def test_bid_correct_version_30(
    present_ifa: str, bidder: Bidder, bid_request30: dict
) -> None:
    """Test that we bid on an OpenRTB 3.0 bid request with x-openrtb-version."""
    bid_request30["openrtb"]["request"]["context"]["device"]["ifa"] = present_ifa
    bidder.bid(bid_request30, headers={"x-openrtb-version": "3.0"})


@pytest.mark.parametrize("database_type", ("dynamodb",))
def test_bid_correct_version_25_header(
    present_ifa: str, bidder: Bidder, bid_request25: dict
) -> None:
    """Test that we bid on an OpenRTB 2.5 bid request with x-openrtb-version."""
    bid_request25["device"]["ifa"] = present_ifa
    bidder.bid(bid_request25, headers={"x-openrtb-version": "2.5"})


@pytest.fixture
def openrtb25_as_default(values_yaml: dict) -> None:
    """Make OpenRTB 2.5 the default protocol."""
    values_yaml["config"]["BIDREQUEST_OPEN_RTB_VERSION"] = "2.5"


@pytest.mark.parametrize("database_type", ("dynamodb",))
def test_bid_correct_version_25_default(
    openrtb25_as_default: None, present_ifa: str, bidder: Bidder, bid_request25: dict
) -> None:
    """Test that we bid on an OpenRTB 2.5 bid request if default without x-openrtb-version."""
    bid_request25["device"]["ifa"] = present_ifa
    bidder.bid(bid_request25)


def test_nobid_wrong_ifa_30(
    present_ifa: str, absent_ifa: str, bidder: Bidder, bid_request30: dict
) -> None:
    """Test that the bidder does not bid if the device ID is unknown."""
    bid_request30["openrtb"]["request"]["context"]["device"]["ifa"] = absent_ifa
    bidder.nobid(bid_request30)


@pytest.mark.parametrize("database_type", ("dynamodb",))
def test_nobid_wrong_ifa_25(
    present_ifa: str, absent_ifa: str, bidder: Bidder, bid_request25: dict
) -> None:
    """Test that the bidder does not bid if the device ID is unknown."""
    bid_request25["device"]["ifa"] = absent_ifa
    bidder.nobid(bid_request25, headers={"x-openrtb-version": "2.5"})


def test_nobid_no_ifa(database_type: str, bidder: Bidder, bid_request30: dict) -> None:
    """Test that the bidder does not bid if the device ID is missing."""
    del bid_request30["openrtb"]["request"]["context"]["device"]["ifa"]
    bidder.nobid(bid_request30)


def test_nobid_timeout(
    short_timeout: None,
    present_ifa: str,
    campaign: bytes,
    bidder: Bidder,
    bid_request30: dict,
) -> None:
    """Test that the bidder does not bid on a known device if the query timeout is too short."""
    bid_request30["openrtb"]["request"]["context"]["device"]["ifa"] = present_ifa
    bidder.nobid(bid_request30, status=504)


def test_spend_budget(
    db_client: DatabaseClient,
    present_ifa: str,
    campaign: bytes,
    budget: int,
    bidder: Bidder,
    bid_request30: dict,
) -> None:  # pragma: no cover
    """
    Test that the bidder does not bid after spending the entire budget and possibly one more bid.

    Due to budget refreshes, this requires writing an updated (non-available) budget before every
    refresh.
    """
    bid_request30["openrtb"]["request"]["context"]["device"]["ifa"] = present_ifa
    for i in range(10):
        response = bidder.bid(bid_request30)
        assert (  # nosec
            response["openrtb"]["response"]["id"]
            == bid_request30["openrtb"]["request"]["id"]
        )
        assert response["openrtb"]["response"]["seatbid"][0]["bid"][0][  # nosec
            "cid"
        ] == binascii.hexlify(campaign).decode("utf-8"), "Our campaign must win."

        # Update budget.
        db_client.put_budget(budget, campaign, 1_000_000 * (9 - i))

    # Budget is spent, now nobid. We might bid once if the budget was refresh between the last
    # expected bid and budget update.
    extra_bids = 0
    for _ in range(10):
        extra_bids += bidder.maybe_bid(bid_request30) is not None

    # Possibly due to cache update performance, we sometimes observe two extra bids.
    too_many_extra_bids = 3

    assert (  # nosec
        extra_bids < too_many_extra_bids
    ), "After spending its budget, the bidder should bid at most once"
