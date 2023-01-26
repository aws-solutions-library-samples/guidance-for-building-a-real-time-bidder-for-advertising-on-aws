# Load generator

Load generator is a tool for benchmarking bidder. It generates HTTP POST requests with OpenRTB 3.0 bid requests in its body.

# Building docker image
From the main catalog, run
```makefile
 make load-generator@build
```

This will build a docker image load generation app installed.

## Load generator parameters

To start the load test run:
```shell
docker run -e LOAD_GENERATOR_TARGET=https://your-target-url.com/ ${AWS_ACCOUNT}.dkr.ecr.us-east-1.amazonaws.com/load-generator
```

To start the load test with profiler run:
```shell
kubectl port-forward svc/bidder-internal 8091:8091

docker run --network host -e \
  LOAD_GENERATOR_TARGET=https://your-target-url.com/ \
  LOAD_GENERATOR_PROFILER_URL=http://localhost:8091/debug/pprof/ \
  LOAD_GENERATOR_PROFILER_OUTPUT=/tmp/pprof.out \
  ${AWS_ACCOUNT}.dkr.ecr.us-east-1.amazonaws.com/load-generator
```

Number of device IDs generated during load test:

Load generator uses two flags to describe the number and value of device IDs that are to be 
generated during load test.

`LOAD_GENERATOR_DEVICES_USED` - Number of unique valid device IDs used in the load test. It can 
be manipulated to test various scenarios e.g. behavior of caches.

`LOAD_GENERATOR_NOBID_FRACTION` - Fraction of bidrequests that should provoke nobid response by 
using non-existing device ID. Example: for `LOAD_GENERATOR_DEVICES_USED=1000` and 
`LOAD_GENERATOR_NOBID_FRACTION=0.1` load generator will use 1000 device IDs from the database and
111 IDs outside database range.

By default a constant number of requests per second is generated: set `LOAD_GENERATOR_RATE` to their number per
second.

List of available environment variables:

| Env                            | Default Value                                 | Description                                                      |
| -------------------------------|-----------------------------------------------| -----------------------------------------------------------------|
| LOAD_GENERATOR_RATE            | 100                                           | number of requests per second                                    |
| LOAD_GENERATOR_DURATION        | 6s                                            | duration of load test in seconds                                 |
| LOAD_GENERATOR_TIMEOUT         | 100ms                                         | request timeout                                                  |
| LOAD_GENERATOR_TARGET          | None                                          | URL to the bidder Open RTB endpoint                              |
| LOAD_GENERATOR_DEVICES_USED    | 10                                            | number of device ids that can be generated                       |
| LOAD_GENERATOR_NOBID_FRACTION  | 0.1                                           | fraction of bidrequests that should provoke nobid response       |
| LOAD_GENERATOR_WORKERS         | 10                                            | the number of workers used in the attack                         |
| LOAD_GENERATOR_PROFILER_URL    |                                               | URL of the profiler endpoints                                    |
| LOAD_GENERATOR_PROFILER_OUTPUT |                                               | File path to save pprof output to                                |
| LOAD_GENERATOR_HISTOGRAM       | 0ms:29ms:1ms,30ms:100ms:2ms,110ms:1000ms:10ms | Histogram buckets spec. in format 'FROM:TO:SIZE,FROM:TO:SIZE...' |
| LOAD_GENERATOR_TRACK_ERRORS    | false                                         | Enable request error tracking                                    |
