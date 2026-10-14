# Work log specification

Work logs are append-only Markdown records for meaningful agent sessions. Every
record MUST state the session objective, completed work, remaining work, and
next action. It SHOULD identify the harness, task, run, changed files, and
verification evidence.

Work logs are distinct from handoffs. A handoff is required when responsibility
moves between agents or harnesses; a work log is expected after ordinary
progress even when the same agent continues.
