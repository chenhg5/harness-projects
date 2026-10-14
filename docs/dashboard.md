# Local project dashboard

Start a dashboard from a source checkout or installed `hp` binary:

```bash
hp project serve --path /path/to/harness-project --open
```

The command prints a local URL. Add `--port 8765` for a fixed port or omit
`--open` to open the URL manually. Press Ctrl-C to stop the server.

The page shows the project's current phase, health, progress summary, goals,
completion conditions, optional key results, linked task counts, all task
statuses, registered repositories, next actions, risks, and the latest work log
and handoff. Goal Markdown is available as an expandable document. A sticky
section navigation bar jumps to each area, and its Refresh view link loads
changes without interrupting a reading session. Each page request reads the
project files again. The header switches between English and Simplified Chinese;
the selection stays in the URL (`?lang=en` or `?lang=zh-CN`) and survives a
refresh. Without a selection, the dashboard follows the browser's preferred
language when it is English or Chinese, then falls back to English. Project
content keeps the language in which it was written. `GET /api/project` provides
the unchanged project snapshot as JSON for local integrations.

The dashboard is read-only and binds to `127.0.0.1` only. It does not expose
arbitrary project paths or modify state. Task progress is calculated from task
records, while goal completion remains the explicit goal status and evidence
recorded by the project. The dashboard does not infer completion from task or
key-result counts.
