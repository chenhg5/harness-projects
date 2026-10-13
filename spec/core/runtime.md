# Project Runtime

The project runtime extends the file contract with goals, tasks, runs, Agents,
context, events, artifacts, verification, and human control. The provider
adapters under `adapters/` map these portable objects to Claude Code Projects,
Cursor Projects, Codex sessions, or other execution systems.

The runtime MUST preserve the portable project files even when an adapter also
stores provider-specific state. Provider state is an optimization or integration
detail, not the project’s only durable record.
