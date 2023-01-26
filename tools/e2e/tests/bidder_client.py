"""Bidder API client."""

from dataclasses import dataclass
from typing import Any
from typing import Optional
from urllib.parse import urljoin

import requests


@dataclass
class Bidder:
    """Bidder API client."""

    external_url: str
    internal_url: str

    def request_external(
        self, method: str, path: str, status: int, **kwargs: Any
    ) -> requests.Response:
        """Run a request on the given external *path*, expecting *status*."""
        response = requests.request(method, urljoin(self.external_url, path), **kwargs)
        assert response.status_code == status, (  # nosec
            response.status_code,
            response.content,
        )
        return response

    def request_internal(
        self, method: str, path: str, status: int, **kwargs: Any
    ) -> requests.Response:
        """Run a request on the given internal *path*, expecting *status*."""
        response = requests.request(method, urljoin(self.internal_url, path), **kwargs)
        assert response.status_code == status, (  # nosec
            response.status_code,
            response.content,
        )
        return response

    def bid(self, body: dict, **kwargs: Any) -> dict:
        """Send a bid request and return a bid response."""
        # In future we might retry for the request to succeed.
        response = self.request_external(
            "POST", "/bidrequest", status=200, json=body, **kwargs
        )
        return response.json()

    def nobid(self, body: dict, status: int = 204, **kwargs: Any) -> None:
        """Send a bid request and verify it gets a no bid."""
        response = self.request_external(
            "POST", "/bidrequest", status=status, json=body, **kwargs
        )
        assert response.content == b""  # nosec

    def maybe_bid(self, body: dict, **kwargs: Any) -> Optional[dict]:
        """
        Send a bid request and return the bid response or None in case of no bid.

        Use this only in tests inspecting the response and not knowing if the bidder bids, e.g. due
        to nondeterministic budget behaviour.
        """
        response = requests.request(
            "POST", urljoin(self.external_url, "/bidrequest"), json=body, **kwargs
        )
        assert response.status_code in (200, 204), (  # nosec
            response.status_code,
            response.content,
        )

        # This branch isn't covered by every test run.
        if response.status_code == 200:  # pragma: no cover
            return response.json()
        assert response.content == b""  # nosec
        return None
