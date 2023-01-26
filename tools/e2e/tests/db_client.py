"""Database client."""

import struct
from typing import Any
from typing import Optional
from typing import Union

import _pytest.fixtures
import aerospike


class DatabaseClient:
    """Base class of a database client inserting data and cleaning up."""

    def close(self) -> None:  # pragma: no cover
        """Delete temporary keys."""
        raise NotImplementedError

    def put_campaign(
        self, campaign_id: bytes, bid_price: int
    ) -> None:  # pragma: no cover
        """Put the given campaign."""
        raise NotImplementedError

    def put_audience(
        self, audience_id: bytes, campaign_ids: set[bytes]
    ) -> None:  # pragma: no cover
        """Put the given audience."""
        raise NotImplementedError

    def put_device(
        self, device_id: bytes, audience_ids: set[bytes]
    ) -> None:  # pragma: no cover
        """Put the given device."""
        raise NotImplementedError

    def put_budget(
        self, budget_id: int, campaign_id: bytes, campaign_budget: int
    ) -> None:  # pragma: no cover
        """
        Put the given budget.

        For simplicity, we handle only one campaign in the budget.
        """
        raise NotImplementedError


class BaseDynamoDBClient(DatabaseClient):
    """Base class for DynamoDB and DAX clients."""

    def __init__(self, request: _pytest.fixtures.SubRequest) -> None:
        """Initialize."""
        #: List of items (table name, key dict tuples) that were added by the test and must be
        #: deleted afterwards.
        self._items_to_delete: list[tuple[str, dict]] = []
        #: DynamoDB (or DAX) resource.
        self._dynamodb: Any = None

    def put(self, table: str, key: dict, value: dict) -> None:
        """Put the given key and value into the table."""
        self._dynamodb.Table(table).put_item(Item=value | key)
        self._items_to_delete.append((table, key))

    def close(self) -> None:
        """Delete temporary keys."""
        for table, key in self._items_to_delete:
            self._dynamodb.Table(table).delete_item(Key=key)

    def put_campaign(self, campaign_id: bytes, bid_price: int) -> None:
        """Put the given campaign."""
        self.put(
            "e2e-campaign",
            {"campaign_id": campaign_id},
            {"bid_price": bid_price},
        )

    def put_audience(self, audience_id: bytes, campaign_ids: set[bytes]) -> None:
        """Put the given audience."""
        self.put(
            "e2e-audience", {"audience_id": audience_id}, {"campaign_ids": campaign_ids}
        )

    def put_device(self, device_id: bytes, audience_ids: set[bytes]) -> None:
        """Put the given device."""
        self.put("e2e-device", {"d": device_id}, {"a": b"".join(audience_ids)})

    def put_budget(
        self, budget_id: int, campaign_id: bytes, campaign_budget: int
    ) -> None:
        """
        Put the given budget.

        For simplicity, we handle only one campaign in the budget.
        """
        batch = bytearray(24)
        struct.pack_into("<16sQ", batch, 0, campaign_id, campaign_budget)
        self.put("e2e-budget", {"i": budget_id}, {"b": batch, "s": 1})


class DynamoDBClient(BaseDynamoDBClient):
    """Database client for DynamoDB."""

    def __init__(self, request: _pytest.fixtures.SubRequest) -> None:
        """Initialize."""
        super().__init__(request)
        self._dynamodb = request.getfixturevalue("dynamodb")


class DAXClient(BaseDynamoDBClient):
    """Database client for DAX."""

    def __init__(self, request: _pytest.fixtures.SubRequest) -> None:
        """Initialize."""
        super().__init__(request)
        self._dynamodb = request.getfixturevalue("dax")


class AerospikeClient(DatabaseClient):
    """Database client for Aerospike."""

    _budget_batches_set = "budget_batches"

    def __init__(self, request: _pytest.fixtures.SubRequest) -> None:
        """Initialize."""
        #: List of items (table name, key dict tuples) that were added by the test and must be
        #: deleted afterwards.
        self._items_to_delete: list[tuple[str, Union[int, bytearray]]] = []
        self._aerospike = request.getfixturevalue("aerospike")
        self._namespace = request.config.getoption("aerospike_namespace", "test")
        self._budget_batches_key = request.getfixturevalue("budget_batches_key")

        self.initialize_budgets()

    def put(
        self,
        set_name: str,
        key: Union[int, bytearray],
        value: dict,
        policy: Optional[dict] = None,
    ) -> None:
        """
        Put the given key and value into the table.

        Since Aerospike client documentation looks unaware of Python 3 and the `bytes` type, we use
        `bytearray` for binary data.
        """
        self._aerospike.put(
            (self._namespace, set_name, key),
            value,
            policy=policy,
        )
        self._items_to_delete.append((set_name, key))

    def initialize_budgets(self) -> None:
        """
        Create an empty list for budget_batches.

        Aerospike does not have an API for inserting an empty list.
        So we need to hack it: use list_append (that will create a list)
        and then clear it.
        """
        self._aerospike.list_append(
            key=(self._namespace, self._budget_batches_set, self._budget_batches_key),
            bin="keys",
            val=None,
        )
        self._aerospike.list_clear(
            key=(self._namespace, self._budget_batches_set, self._budget_batches_key),
            bin="keys",
        )

    def budget_batch_key_append(self, value: int) -> None:
        """
        Append to the list of budget batches keys a new value.

        Because some tests update the same budget (using put budget)
        we should clear the list first not to have duplicates in the
        list.
        """
        self._aerospike.list_clear(
            key=(
                self._namespace,
                self._budget_batches_set,
                self._budget_batches_key,
            ),
            bin="keys",
        )
        self._aerospike.list_append(
            key=(self._namespace, self._budget_batches_set, self._budget_batches_key),
            bin="keys",
            val=value,
        )
        self._items_to_delete.append(
            (self._budget_batches_set, self._budget_batches_key)
        )

    def close(self) -> None:
        """Delete temporary keys."""
        for set_name, key in self._items_to_delete:
            try:
                self._aerospike.remove((self._namespace, set_name, key))
            except aerospike.exception.RecordNotFound:  # pylint:disable=no-member
                # Duplicate key.
                pass

    def put_campaign(self, campaign_id: bytes, bid_price: int) -> None:
        """Put the given campaign."""
        self.put(
            "campaign",
            bytearray(campaign_id),
            {"bid_price": bid_price},
            policy={"key": aerospike.POLICY_KEY_SEND},
        )

    def put_audience(self, audience_id: bytes, campaign_ids: set[bytes]) -> None:
        """Put the given audience."""
        self.put(
            "audience_campaigns",
            bytearray(audience_id),
            {"campaign_ids": [bytearray(campaign_id) for campaign_id in campaign_ids]},
            policy={"key": aerospike.POLICY_KEY_SEND},
        )

    def put_device(self, device_id: bytes, audience_ids: set[bytes]) -> None:
        """Put the given device."""
        self.put(
            "device",
            bytearray(device_id),
            {"audience_id": bytearray(b"".join(audience_ids))},
        )

    def put_budget(
        self, budget_id: int, campaign_id: bytes, campaign_budget: int
    ) -> None:
        """
        Put the given budget.

        For simplicity, we handle only one campaign in the budget.
        """
        self.budget_batch_key_append(budget_id)

        batch = bytearray(24)
        struct.pack_into("<16sQ", batch, 0, campaign_id, campaign_budget)
        self.put("budget", budget_id, {"batch": batch})
