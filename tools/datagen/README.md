# datagen

The `datagen` provides tools for data generation and simple management of
DynamoDB tables.

It generates three types of data:

* `campaigns` (uses campaign id as key and includes bid price and total budget),
* `audiences` (uses audience id as key and includes list of campaign ids),
* `devices` (uses device id as key and includes list of audiences).

The audience generator produces record per campaign id, the records contains
a list of audience ids. Then, we construct a reverse index where the
audience id is the key and campaigns ids are collected as values.

The only required option is `-type` to specify the data type to be generated.

There are following options to configure the generation process:

* `-output=stdout|dynamodb|aerospike` to specify where to write the data
  (default: `stdout`)

* `-low=N` `-high=M` generator will create items with consecutive identifiers
  from `N` to `M`, each identifier is converted to byte slice and encrypted
  with AES.

* `-min-audiences` `-max-audiences` `-max-audience-id` options specify
  that every record will have between `-min-audiences` and `-max-audiences`
  audiences, and there will be `-max-audience-id` unique audiences.

* `-table` `-concurrency` options are dedicated for the `dynamodb` output,
  `-table` specifies the table name, and `-concurrency` defines the number
  of writers used to write the data in parallel.

The helper command `dynamo_table` creates, deletes, describes and lists a
table. For details use `-h` option.

There is no equivalent helper for Aerospike: namespaces are set in the server
configuration, while sets are created when inserting the data. Use Aerospike
tools, <https://www.aerospike.com/docs/deploy_guides/docker/tools/index.html>,
to inspect the generated data.

## Manage AWS SDK

`datagen` uses default mechanism for credential resolution
(see: https://docs.aws.amazon.com/sdk-for-go/api/aws/session/).

In addition,
* use `AWS_REGION` environment variable to set the region,
* use `AWS_DYNAMODB_ENDPOINT_URL` environment variable to use non-standard
  endpoint URL to communicate with the DynamoDB service, e.g. in the case of
  using a local instance.

## Examples

Create a table, generate data, check number of records and list one of them,
all in local environment.

```shell
# Start a container with the DynamoDB
make deploy-local

# the output of the command `dynamodb_table` is in JSON format,
# you can use `jq` tool for pretty-printing.
./bin/dynamo_table -table=devices -cmd=create | jq

./bin/datagen -type=devices -output=dynamodb -local -table=devices -concurrency=4 -max-audiences=10 -max-audience-id=20 -low 250 -high 300

./bin/dynamo_table -table=devices -cmd=describe | jq .ItemCount

./bin/dynamo_table -table=devices -cmd=list -limit=1 | jq
```

### Generate a reverse-index of campaigns

_Example uses the table and attribute names defined in the infrastructure
code._

```shell
# create table
./bin/dynamo_table -table=audience_campaigns -cmd=create -pk=audience_id | jq
# or describe existing
./bin/dynamo_table -table=audience_campaigns -cmd=describe | jq

# generate 1M campaigns
./bin/datagen -type=audiences -max-audience-id=100000 -min-audiences=10 -max-audiences=20 -output=dynamodb -concurrency=16 -table=audience_campaigns -low=1 -high=1000000
```

### Generate campaigns

```shell
# create table
./bin/dynamo_table -table=campaign_budget -cmd=create -pk=campaign_id | jq
# or describe existing
./bin/dynamo_table -table=campaign_budget -cmd=describe | jq

# generate 1M budgets
./bin/datagen -type=campaigns -output=dynamodb -concurrency=16 -table=campaign_budget -low=1 -high=1000000
```

### Generate devices

```shell
# create table
./bin/dynamo_table -table=dev -cmd=create -pk=d | jq
# or describe existing
./bin/dynamo_table -table=dev -cmd=describe | jq

# generate 100M devices
./bin/datagen -type=devices -max-audience-id=100000 -min-audiences=10 -max-audiences=20 -output=dynamodb -concurrency=64 -table=dev -low=1 -high=100000000
```
