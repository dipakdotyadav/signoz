# SigNoz Community Edition multi-tenancy MVP

This folder contains local bootstrap and validation helpers for the Community
Edition multi-tenancy branch. The MVP uses shared ClickHouse storage with
server-derived organization/project identity and project-scoped ingestion keys.

## Scope

- Two projects in the authenticated organization.
- A distinct ingestion key per project.
- OTLP authentication at the collector.
- Trusted tenant identity injected by the collector, not accepted from client
  resource attributes.
- Structured Query Builder v5 queries scoped by the authenticated project.
- Legacy query endpoints reject `X-SigNoz-Project-ID`; raw SQL and PromQL are
  not project-scoped in this MVP and must fail closed.

This helper does not create organizations or users. Sign in to the target
organization with an organization administrator account first and supply that
account's bearer token.

## Bootstrap

From the SigNoz repository root:

```bash
export SIGNOZ_API_URL=http://localhost:8080
export SIGNOZ_ADMIN_TOKEN='<admin bearer token>'
export SIGNOZ_MVP_ENV_FILE="$PWD/signoz-mvp-projects.env"
bash scripts/multitenancy/mvp-bootstrap.sh
```

The script calls:

- `POST /api/v2/projects`
- `POST /api/v2/projects/{projectId}/ingestion-keys`

The returned plaintext ingestion secrets are written once to the output file
with mode `0600`. Keep that file out of source control.

## Collector

Run the forked Collector with both its default configuration and
`config/multitenant-local.yaml`. For local loopback-only testing, the overlay
must point its resolver to:

```yaml
resolver_endpoint: http://localhost:8080/api/v2/internal/ingestion-keys/resolve
allow_insecure: true
```

Set the same high-entropy `SIGNOZ_INTERNAL_INGESTION_RESOLVER_TOKEN` in the
SigNoz Community server and Collector processes. This HTTP exception is only
for a local loopback development environment; use HTTPS and
`allow_insecure: false` outside it.

## Ingestion endpoints

- OTLP/gRPC: `localhost:4317`
- OTLP/HTTP: `http://localhost:4318`

Send the key in HTTP as `signoz-ingestion-key`, or in gRPC metadata with the
same name. Requests without a key, with an invalid/revoked/expired key, or
with a key for an archived project must be rejected.

## MVP acceptance tests

1. Start ClickHouse, the Community server, and the forked Collector.
2. Confirm the internal resolver rejects a missing/wrong internal token.
3. Confirm OTLP requests without an ingestion key are rejected.
4. Ingest traces, logs, and metrics using Project A and Project B keys.
5. Send deliberately false `signoz.org.id` and `signoz.project.id` resource
   attributes with Project A's key; verify the Collector overwrites them with
   Project A's server-resolved identity.
6. Query through `POST /api/v5/query_range` with
   `X-SigNoz-Project-ID: <projectId>`. Verify Project A cannot retrieve
   Project B's signals and vice versa.
7. Verify project-scoped requests to legacy v1-v4 query endpoints are rejected.
8. Revoke each key and verify the next OTLP request is rejected.
9. Verify the pre-existing organization-scoped single-tenant flow still works
   when the multi-tenant Collector overlay is not enabled.

A successful HTTP ingest response is not proof of storage/query isolation.
Do not mark the MVP production-ready until the two-project negative query tests
and the legacy compatibility test pass against a running local stack.
