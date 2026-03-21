# Routing

The gateway supports two routing modes for resolving incoming HTTP requests to Lambda function names: **path-based** (default) and **subdomain-based**.

## Path-based routing (default)

In path-based routing, the Lambda function name is extracted from the first segment of the URL path. The remaining path is forwarded to the Lambda function.

    ROUTING_MODE=path   # or simply leave unset

**Example:**

    curl http://localhost:8090/MyFunction/some/path

- Function name: `MyFunction`
- Forwarded path: `/some/path`

If only a function name is provided (e.g. `/MyFunction`), the forwarded path defaults to `/`.

## Subdomain-based routing

In subdomain-based routing, the Lambda function name is extracted from the subdomain of the `Host` header. The entire URL path is forwarded to the Lambda function unchanged.

    ROUTING_MODE=subdomain
    BASE_DOMAIN=live.mocks.cloud

**Example:**

    curl https://my-function.live.mocks.cloud/some/path

- Function name: `my-function`
- Forwarded path: `/some/path`

The `BASE_DOMAIN` environment variable must be set when using subdomain routing. The gateway strips `.<BASE_DOMAIN>` from the `Host` header to determine the function name. Ports in the `Host` header are handled automatically.

Requests whose `Host` header does not match the base domain will receive a `400 Bad Request` response.

## Function prefix

In both routing modes, you can configure an optional prefix that is prepended to the resolved function name before invoking Lambda:

    FUNCTION_PREFIX=imposter-

**Example with path-based routing:**

    curl http://localhost:8090/abc123/pets
    # Invokes Lambda function: imposter-abc123

**Example with subdomain-based routing:**

    BASE_DOMAIN=live.mocks.cloud
    FUNCTION_PREFIX=imposter-

    curl https://abc123.live.mocks.cloud/pets
    # Invokes Lambda function: imposter-abc123

This is useful when the URL-facing identifier differs from the actual Lambda function name by a known prefix.
