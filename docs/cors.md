# CORS

The gateway can optionally add permissive CORS headers to all responses, allowing browser-based clients to call your Lambda functions from any origin.

> This behaviour is disabled by default.

## Enabling permissive CORS

Set the `CORS_PERMISSIVE` environment variable to `true`:

    CORS_PERMISSIVE=true

## Behaviour

When enabled, the gateway:

- Reflects the requesting origin back as the allowed origin, so any origin is permitted.
- Allows common request headers: `Accept`, `Authorization`, `Content-Type`, `X-Requested-With`, `X-Request-ID`, and `X-Correlation-ID`.
- Allows all standard HTTP methods: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS`, and `HEAD`.
- Permits credentialed requests (cookies, authorization headers).
- Responds to `OPTIONS` preflight requests automatically, so your Lambda functions do not need to handle them.
- Caches preflight responses for 24 hours to reduce the number of preflight requests made by browsers.

## Example

    CORS_PERMISSIVE=true ./lambdahttpgw

A browser-based application on `https://myapp.example.com` can then call:

```javascript
fetch("http://localhost:8090/MyFunction/path", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "Authorization": "Bearer my-token"
  },
  body: JSON.stringify({ key: "value" })
})
```

The gateway will include the appropriate CORS headers in the response, allowing the browser to complete the request.
