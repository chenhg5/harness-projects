# Lifecycle and cleanup specification

Project state is divided into durable history and bounded transient material.
Goals, specifications, tasks, runs, verification, memory, and handoffs are
durable and MUST NOT be selected by routine cleanup. Work logs, generated
artifacts, and imported raw context MAY be archived after a retention period.

`hp cleanup plan` MUST preview candidates. `hp cleanup apply` MUST archive by
default. Permanent deletion requires an explicit delete action and force flag.
