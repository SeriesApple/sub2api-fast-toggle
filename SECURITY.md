# Security

## Intended Deployment

The proxy binds to `127.0.0.1` by default. Keep it local unless you provide a
separate authenticated and encrypted ingress. The proxy forwards the incoming
`Authorization` header to Sub2API and must be treated as part of the trusted API
path.

The proxy never logs request bodies, response bodies, API keys, authorization
headers, or prompt content. Its event logs contain only an action, HTTP method,
request path, and model ID.

Do not commit `.env`, API keys, Sub2API database files, logs, or exported usage
data. This repository's `.gitignore` excludes common local files, but secret
scanning before publication is still required.

## Request Inspection

For configured Responses endpoints, the proxy inspects JSON bodies so it can
add or remove Fast service tiers. Compressed, malformed, or oversized matching
requests are rejected instead of being forwarded without inspection. This
prevents Fast OFF from being bypassed by an uninspectable request.

## Reporting

Use GitHub private vulnerability reporting for vulnerabilities in this
project. Report Sub2API vulnerabilities to the upstream Sub2API maintainers.
Do not include live credentials, private prompts, or personal data in reports.
