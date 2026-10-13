# Core Project Model

**Status:** Draft normative specification

This document defines the portable project boundary. RFCs under `../../rfcs/`
propose changes; accepted behavior is recorded here.

## Normative language

The terms **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT**, and **MAY** are to
be interpreted as described by RFC 2119.

## Canonical areas

An initialized Harness Project MUST contain:

```text
PROJECT.md
spec/
goals/
tasks/
runs/
agents/
context/
handoffs/
events/
verification/
skills/
scripts/
repos/
state/
artifacts/
integrations/
.harness-project/
```

`PROJECT.md` is the human-readable entry point. `.harness-project/` contains
machine-readable metadata. Harness-specific files are generated views.

## Source of truth

The canonical files in the project root MUST remain useful without a specific
coding harness. Adapters MUST NOT store the only copy of project requirements,
plans, decisions, or handoffs.
