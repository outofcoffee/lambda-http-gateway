# Lambda HTTP Gateway

Simple HTTP gateway for AWS Lambda. Acts as a lightweight API gateway for AWS Lambda functions.

## Run

You can download the binary for your platform from the [releases](https://github.com/outofcoffee/lambda-http-gateway/releases) page, or use the [Docker image](#docker-image).

> **Important:** Ensure the relevant AWS credentials are configured before run. The gateway uses the standard AWS mechanisms to authenticate/authorise with the AWS Lambda API, so the usual approaches of profiles/credentials apply.

### Binary

    ./lambdahttpgw

### Docker

    docker run -it -p 8090:8090 outofcoffee/lambdahttpgw

> Note: the home directory for the `gateway` user that runs the binary is `/opt/gateway`

## Call Lambda function

Call the Lambda function via the gateway:

    curl http://localhost:8090/MyLambdaName/some/path
    ...
    <Lambda HTTP response>

> Note the prefix of the Lambda function name (`MyLambdaName` above), before the path. The function receives the portion of the path without the function name, i.e. `/some/path` in this example.

The Lambda function receives events in the standard AWS API Gateway JSON format, and is expected to respond in kind.

The gateway also supports [subdomain-based routing](./docs/routing.md), where the function name is extracted from the `Host` header instead of the path.

## Configuration

Environment variables:

| Variable              | Meaning                                                                                         | Default     | Example                            |
|-----------------------|-------------------------------------------------------------------------------------------------|-------------|------------------------------------|
| AWS_REGION            | AWS region in which to connect to Lambda functions.                                             | `eu-west-1` | `us-east-1`                        |
| BASE_DOMAIN           | Base domain for subdomain routing. Required when `ROUTING_MODE=subdomain`.                      | Empty       | `live.mocks.cloud`                 |
| FUNCTION_PREFIX       | Optional prefix prepended to resolved function names before invoking Lambda.                    | Empty       | `imposter-`                        |
| LOG_LEVEL             | Log level (trace, debug, info, warn, error).                                                    | `debug`     | `warn`                             |
| PORT                  | Port on which to listen.                                                                        | `8090`      | `8080`                             |
| REQUEST_ID_HEADER     | Name of request header to use as request ID for logging. If absent, a UUID will be used.        | Empty       | `x-correlation-id`                 |
| ROUTING_MODE          | Routing mode: `path` extracts function name from URL path, `subdomain` from the Host header.    | `path`      | `subdomain`                        |
| STATS_RECORDER        | Whether to record number of hits for each function.                                             | `false`     | `true`                             |
| STATS_REPORT_INTERVAL | The frequency with which stats should be reported, if enabled.                                  | `5s`        | `2m`                               |
| STATS_REPORT_URL      | URL to which stats should be reported. If not empty, hits are recorded for each function name.  | Empty       | `https://example.com`              |

See [Routing](./docs/routing.md) for details on path-based vs subdomain-based routing.

## Build

Prerequisites:

- Go 1.17+

Steps:

    go build

## Docker image

Image on [Docker Hub](https://hub.docker.com/r/outofcoffee/lambdahttpgw):

    outofcoffee/lambdahttpgw

## Stats recording and reporting

The Gateway can optionally record the number of hits per function and report it to an external hit counter server.

> This behaviour is disabled by default.

See [Stats recording and reporting](./docs) for details.
