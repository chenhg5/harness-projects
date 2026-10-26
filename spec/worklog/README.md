# Work log specification

Work logs are append-only Markdown records for meaningful agent sessions. Every
record MUST make the session objective, completed work, what remains (including
"nothing" when complete), and the next action understandable to another agent.
Remaining work and next action may share a section. A specific heading language
or Markdown template is not required. A log SHOULD include verification evidence
and identify the harness, task, run, and changed files. When the actual provider
session ID is available, a new log SHOULD record it as `sessionId`; the log's
`id` is a separate, human-readable record identifier. Never invent a session
ID. `hp finish` and `hp worklog create` capture a detected session ID or accept
`--session-id` explicitly.

The CLI writes a consistent English Markdown template for new logs. The doctor
checks explicit template fields and empty placeholders. It does not reject
historical or translated prose solely because its headings differ. A mechanical
check cannot establish whether a summary is accurate or useful, so agents must
still write concrete results, remaining work, and a next action someone can take.

Work logs are distinct from handoffs. A handoff is required when responsibility
moves between agents or harnesses; a work log is expected after ordinary
progress even when the same agent continues.
