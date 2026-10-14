# Handoff Specification

Handoffs are append-only records of work transferred between agents or coding
harnesses. A project may contain any number of handoff records.

The reference CLI creates handoffs with `hp handoff create`, lists them with `hp handoff list`, and reads the newest record with `hp handoff latest`. Handoff Markdown remains the portable source of truth.
