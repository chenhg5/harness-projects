# Project Manifest

The initial manifest is `.harness-project/project.json`.

## Required fields

```json
{
  "schemaVersion": "1",
  "name": "Example Project",
  "createdAt": "2026-10-14T00:00:00Z"
}
```

An implementation MUST preserve unknown fields when updating the manifest. A
consumer MUST reject an unsupported `schemaVersion` rather than silently
interpreting it as the current version.
