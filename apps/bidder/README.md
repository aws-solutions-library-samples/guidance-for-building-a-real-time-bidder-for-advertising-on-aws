# Bidder application

## Run bidder locally

```
set -a ; . apps/bidder/env/local.env ; set +a
cd apps/bidder/cmd
go run .
curl http://localhost:8090/metrics
```

### Run dependencies

To run LocalStack and Aerospike:

```
make bidder@dev-up
```

To run LocalStack only:

```
make bidder@dev-up-ls
```

To run Aerospike only:

```
make bidder@dev-up-as
```

#### Accessing Aerospike locally

Use following parameters:

Host: `localhost`
Port: `3000`
Namespace: `test`

Authentication is disabled.

### Stop local dependencies

```
make bidder@dev-down
```
