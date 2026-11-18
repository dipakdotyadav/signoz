# Project-level ingestion isolation

## Status

Accepted design for the multi-tenancy implementation. This document records the ingestion boundary before introducing project credentials and changing telemetry storage.

## Existing SigNoz behavior

The current code has two different credential paths:

- Cloud ingestion keys are proxied through the external gateway API. They are workspace-scoped and are not a suitable source of project identity in a self-hosted deployment.
- Self-hosted API keys authenticate service accounts. The service-account identity currently resolves to an organization and does not identify a project.

Therefore, attaching a client-supplied `project_id` attribute or adding a project field to the Cloud gateway response would not establish a trusted isolation boundary.

## Decision

Introduce a first-party project ingestion credential subsystem, independent of the Cloud gateway.

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
