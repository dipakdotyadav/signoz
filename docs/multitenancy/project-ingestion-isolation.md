# Project-level ingestion isolation

## Status

Accepted design for the multi-tenancy implementation. This document records the ingestion boundary before introducing project credentials and changing telemetry storage.

## Scope: self-hosted and cloud-compatible deployments

The product target includes both self-hosted/on-premises and cloud deployments, with hybrid isolation: organization boundaries are mandatory and project boundaries apply within an organization. This implementation establishes the self-hosted API/collector contract first. Cloud gateway deployments must integrate with the same credential-resolution and trusted-identity contract; existing Cloud gateway credentials and hosted control-plane behavior must not be silently replaced or bypassed.

The current Community Edition API keys authenticate service accounts. The service-account identity resolves to an organization and does not identify a project. Attaching a client-supplied `project_id` attribute is not a trusted isolation boundary.

## Decision

Introduce a first-party project ingestion credential subsystem for Community Edition.

A credential is bound at creation time to exactly one organization and one project. The project must belong to that organization and be active. The credential resolver derives both IDs from the credential record; it must never trust organization or project IDs supplied by the telemetry client.

### Credential record

The persistent record will contain:

- credential ID
- organization ID
- project ID
- display name
- public key identifier / lookup prefix
- cryptographic secret hash
- optional expiry timestamp
- revocation timestamp
- last-used timestamp
- created and updated timestamps

Only the raw secret is returned once at creation. Store a one-way hash, compare hashes in constant time where applicable, and never log the secret. Use a cryptographically secure random secret with sufficient entropy. Key rotation creates a new credential; revocation invalidates the old credential immediately.

### Ingestion authentication

Add a dedicated ingestion authentication resolver/header that is distinct from interactive service-account API keys. On successful resolution it creates trusted request identity containing both `org_id` and `project_id`.

The resolver must reject expired, revoked, unknown, and inactive-project credentials. Existing organization-level authentication remains supported during migration; it must not be silently reinterpreted as a project credential.

### Telemetry propagation and storage

The trusted project identity must be propagated server-side into the ingestion pipeline and persisted as a first-class tenant dimension. A client-provided resource attribute named `project_id` is untrusted and must be overwritten or rejected.

All read paths must apply organization and project predicates derived from authenticated claims. Project filtering must be enforced in the query layer, not only in UI filters. ClickHouse table partitioning/order keys are a performance decision; they are not a substitute for mandatory query predicates.

### Backward compatibility

- Existing organization-level keys continue to work according to the deployment's compatibility policy.
- Existing telemetry remains associated with the organization and is not automatically assigned to an arbitrary project.
- A default project can be introduced for legacy traffic only through an explicit migration/configuration decision.
- Mixed-mode rollout must not allow a project-scoped identity to query another project by changing a request parameter or telemetry attribute.

## Implementation sequence

1. Add the project ingestion credential model, SQL migration, store, and unit tests.
2. Add create/list/revoke/rotate APIs with project-level authorization.
3. Add the dedicated credential resolver and trusted project claims.
4. Propagate project identity through OTLP ingestion and telemetry storage schemas.
5. Enforce project scope in logs, traces, metrics, dashboards, alerts, and other query paths.
6. Add integration tests proving cross-project reads and writes are denied.
7. Roll out compatibility behavior for existing organization-level ingestion.

## Security acceptance criteria

- A credential for project A cannot write data into project B.
- A credential for project A cannot read project B data.
- Spoofed `org_id` or `project_id` resource attributes cannot override authenticated identity.
- Revoked and expired credentials fail closed.
- A project ID from another organization cannot be bound to a credential.
- Organization-level compatibility does not bypass project authorization for project-scoped users.


## Community Edition credential-management API

The project ingestion credential module is registered in the Community Edition dependency graph and exposed under the existing project API namespace:

| Method | Endpoint | Project permission | Behavior |
|---|---|---|---|
| POST | `/api/v2/projects/{id}/ingestion-keys` | attach | Create credential; return secret once |
| GET | `/api/v2/projects/{id}/ingestion-keys` | read | List credential metadata |
| GET | `/api/v2/projects/{id}/ingestion-keys/{credentialId}` | read | Read credential metadata |
| DELETE | `/api/v2/projects/{id}/ingestion-keys/{credentialId}` | delete | Revoke credential |

The routes reuse the existing project authorization middleware. Project ID and organization ID are resolved from the authenticated request and route; clients cannot choose the organization binding in the request body.

Credential rotation is intentionally not exposed as a non-atomic revoke-then-create operation. It should be added with a transactional store operation so a failed rotation cannot unexpectedly leave the project without a valid credential or two active credentials.

## Collector authentication contract

The companion collector implementation is being developed in [dipakdotyadav/signoz-otel-collector](https://github.com/dipakdotyadav/signoz-otel-collector/tree/feat/project-ingestion-key-auth).

- OTLP clients send `signoz-ingestion-key` as an HTTP header or gRPC metadata value.
- The collector calls `POST /api/v2/internal/ingestion-keys/resolve` with JSON `{"ingestionKey":"..."}` and the `X-SigNoz-Internal-Token` service header.
- The API validates the key hash, expiry, revocation state, and active project, then returns only canonical `orgId` and `projectId` in the standard SigNoz success envelope.
- Configure the same high-entropy `SIGNOZ_INTERNAL_INGESTION_RESOLVER_TOKEN` on the API and collector. The collector resolver endpoint must use HTTPS in production. The endpoint fails closed when the token is missing or invalid.
- The collector authenticator does not cache successful resolutions in this first version, so revocation is observed on the next request.
- The `signoztenantcontext` processor copies trusted identity to `signoz.org.id` and `signoz.project.id`, overwriting spoofed resource attributes. Multi-tenant pipelines must place it after OTLP authentication and before processors/connectors/exporters that derive or persist tenant data.

Do not enable the authenticator on existing OTLP listeners until the collector image has been built from the companion branch and both API and collector have the resolver token configured. This preserves the existing single-tenant/organization-key behavior during rollout.

## Implementation status

Implemented foundations:
- Project-scoped key model, hash verification, SQL migration, store, management APIs, and project authorization.
- Private API resolver endpoint protected by a constant-time checked service token.
- Collector server authenticator and trusted tenant-context processor for traces, metrics, and logs.
- Collector component registration and focused unit tests.

Still outstanding before production:
- Wire the authenticator and tenant-context processor into an explicit multi-tenant collector configuration and deployment image.
- Run Go formatting, unit tests, and collector/API builds in CI; repository writes through GitHub have not themselves verified compilation.
- Persist tenant identifiers as first-class ClickHouse dimensions and enforce organization/project predicates on every read, aggregation, alert, export, and derived-signal path.
- Complete cross-project negative integration tests, revocation tests, and compatibility tests for existing ingestion keys.
- Integrate the cloud gateway path against the same trusted-identity contract.

The current code is an ingestion-authentication foundation, not yet a complete telemetry-isolation implementation.


## Authenticated project query scope

The API now has a reusable `ctxtypes.TenantScope` context value and two authorization middleware entry points:

- `ProjectAccess` attaches the scope after authorizing the project identified by the route's `{id}`.
- `ProjectHeaderAccess` attaches the scope after authorizing the project selected through `X-SigNoz-Project-ID`, for routes such as query-range APIs that do not carry a project path parameter.

The header is a selector, not proof of identity. The middleware resolves the authenticated organization from claims and checks project access against the authorization service before creating the scope. Downstream code should use only `ctxtypes.TenantScopeFromContext`; it must not trust the raw header.

Implementation status on `feat/multi-tenancy-foundation`:

- `ProjectHeaderAccessIfPresent` is wired to `/api/v5/query_range` and `/api/v5/query_range/preview` using `X-SigNoz-Project-ID`. The header is only a selector; middleware authorizes the project against the authenticated organization before attaching `TenantScope`.
- Requests without the project header retain organization-wide behavior for backward compatibility with existing organization-level credentials.
- The query layer injects mandatory organization and project predicates into structured trace, log, metric, and AI builder queries. Trace-operator expressions are validated against named operands, which are independently scoped; formulas consume the scoped query results.
- Project-scoped requests using unsupported query languages fail closed. In particular, PromQL and arbitrary ClickHouse SQL are not yet project-scoped. Raw SQL requires a dedicated isolation policy or restricted execution role; string-appending a predicate is not a safe SQL isolation strategy.

Do not advertise complete project-scoped query isolation yet. Remaining work includes a secure PromQL strategy, a restricted raw-SQL policy, coverage of every query/read path, and cross-project integration tests proving that project A cannot read project B data.

## Query-scope regression coverage

The query-scope unit suite now covers:
- Mandatory organization and project predicates for traces, logs, metrics, and AI trace builder queries.
- Preservation of an existing user filter while adding the mandatory tenant predicates.
- Fail-closed behavior for PromQL and arbitrary ClickHouse SQL.
- Organization mismatch and malformed project UUID rejection.
- Legacy organization-wide requests without a project scope remaining unchanged.
- Nil query request rejection.
- Trace-operator queries whose named operands are independently tenant-scoped, and rejection of missing operands.

These are unit-level request-transformation tests. They do not replace end-to-end tests against ClickHouse proving that two projects cannot read one another's persisted telemetry. They also do not validate that every API route which reaches the querier has the intended authorization middleware.

### Collector spoofing regression coverage

The companion collector's `signoztenantcontextprocessor` test suite explicitly verifies that authenticated identity overwrites canonical tenant attributes and removes client-supplied aliases:
- `org_id`, `project_id`, and `organization_id`
- `signoz-ingestion-key` and `SIGNOZ-API-KEY`

The test change is committed in [signoz-otel-collector commit 936c1d6](https://github.com/dipakdotyadav/signoz-otel-collector/commit/936c1d6f8f48ece89cd71ea9edbb16e49715b384). This is regression-test coverage only; the collector/API build and test workflow has not yet reported a CI run for that commit.
