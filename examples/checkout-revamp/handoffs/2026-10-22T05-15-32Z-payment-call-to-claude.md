# Handoff: payment-call-to-claude

- From: codex
- To: claude-code
- Created: 2026-10-22T05:15:32Z

## Objective

Collapse authorize and capture into one API call

## Completed

New POST /payments handler and tests on api branch fast-capture; works against the fake provider

## Next actions

Add an idempotency key so client retries never double-charge, then run the combined capture test once sandbox keys arrive

## Repositories and commits

api: branch fast-capture

## Verification

go test ./payments/... passes

## Blockers and risks

provider-sandbox: sandbox keys not issued yet
