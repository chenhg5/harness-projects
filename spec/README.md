# Specifications

This directory contains the versioned Harness Projects standard. Specifications
describe the portable project contract; harness-specific behavior belongs under
`../adapters/`. Proposed changes live under `../rfcs/`.

Normative requirements use the RFC 2119 terms MUST, MUST NOT, SHOULD, SHOULD
NOT, and MAY. The project uses RFC-style review and versioning without requiring
the full IETF RFC publication format.

Planned areas:

- `core/` — project concepts and lifecycle;
- `project-manifest/` — machine-readable project metadata;
- `handoff/` — durable agent-to-agent handoffs;
- `skill-mounts/` — shared and harness-specific skill discovery.

Harness integration requirements live under `../adapters/`.
