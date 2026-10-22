#!/bin/sh
# Rebuild examples/checkout-revamp with the hp CLI. Run from the repository root:
#   sh examples/build-checkout-revamp.sh
set -eu
HP="${HP:-hp}"
P=examples/checkout-revamp
rm -rf "$P"
"$HP" project init --path "$P" --name "Checkout Revamp" --no-git >/dev/null

cat > "$P/PROJECT.md" <<'EOF'
# Checkout Revamp

Example Harness Project. A small team and several coding agents (Codex, Claude Code, Cursor) cut checkout time in half across two repositories: `web` (storefront) and `api` (payments service). Nothing here depends on chat history: open this folder with any agent and run `hp brief`.

## Repositories

- `web` — storefront and checkout pages; checkout at `repos/web`.
- `api` — payments and order service; checkout at `repos/api`.

The source code is not part of this example; `repos/` holds only its README.
EOF

"$HP" repository add --project "$P" --id web --url https://github.com/example/web --default-branch main >/dev/null
"$HP" repository add --project "$P" --id api --url https://github.com/example/api --default-branch main >/dev/null

"$HP" goal create --project "$P" --id faster-checkout --title "Cut median checkout time from 62s to under 30s" \
  --description "Checkout abandonment rises with every extra step. Remove the address re-entry and the second payment round trip." \
  --outcome "Median time from cart to order confirmation stays under 30 seconds for two consecutive weeks." \
  --done-when "Median checkout time under 30s for two weeks,Abandonment rate does not increase,Both repositories released behind one feature flag" \
  --evidence-required "timing dashboard export,release links" --owner checkout-team --status active >/dev/null

task() { "$HP" task create --project "$P" --goal-id faster-checkout "$@" >/dev/null; }
task --id timing-baseline --title "Instrument checkout timing" --priority high --status ready
task --id saved-address --title "Reuse the saved shipping address" --priority high --depends-on timing-baseline
task --id single-payment-call --title "Collapse authorize and capture into one API call" --priority critical --depends-on timing-baseline
task --id provider-sandbox --title "Get payment provider sandbox keys for combined capture" --priority high \
  --status blocked --blocked-reason "waiting on finance to request keys"
task --id web-one-page --title "Merge shipping and payment into one page" --priority high --depends-on saved-address
task --id flag-rollout --title "Roll out behind the fast-checkout flag" --priority medium --depends-on web-one-page,single-payment-call
"$HP" task update --project "$P" --id timing-baseline --status completed --result "Median 62s; payment round trips 9s, address re-entry 14s" >/dev/null
"$HP" task update --project "$P" --id saved-address --status completed --result "Median 48s on staging" >/dev/null
"$HP" task update --project "$P" --id single-payment-call --status in_progress >/dev/null

cat > "$P/spec/payment-flow.md" <<'EOF'
# Decision: one payment call

Decided by checkout-team.

Authorize and capture happen in one `POST /payments` call when the provider supports immediate capture. Orders that need manual review keep the two-step flow. Reason: the second round trip costs a median 9 seconds in the timing baseline.
EOF

cat > "$P/state/current-plan.md" <<'EOF'
# Current plan

1. Done: timing baseline (median 62s; payment round trips 9s, address re-entry 14s).
2. Done: saved address reuse in `web` (median now 48s).
3. Now: single payment call in `api` (see `spec/payment-flow.md`).
4. Next: one-page checkout in `web`, then the flag rollout.
EOF

"$HP" worklog create --project "$P" --id saved-address-shipped --agent codex --harness codex --task-id saved-address \
  --objective "Reuse the saved shipping address in checkout" \
  --completed "Prefilled the address from the profile in web; median checkout 62s -> 48s on staging" \
  --remaining "Nothing for this task" \
  --verification "web e2e checkout suite passed" \
  --next "Start single-payment-call in api" >/dev/null
sleep 1
"$HP" handoff create --project "$P" --id payment-call-to-claude --from codex --to claude-code \
  --objective "Collapse authorize and capture into one API call" \
  --completed "New POST /payments handler and tests on api branch fast-capture; works against the fake provider" \
  --next "Add an idempotency key so client retries never double-charge, then run the combined capture test once sandbox keys arrive" \
  --repositories "api: branch fast-capture" \
  --verification "go test ./payments/... passes" \
  --blockers "provider-sandbox: sandbox keys not issued yet" >/dev/null

"$HP" state update --project "$P" --phase build --health at_risk \
  --progress "2 of 6 tasks done; median checkout 48s (target 30s)" \
  --current-goal-id faster-checkout --active-tasks single-payment-call --blocked-tasks provider-sandbox \
  --next-actions "Add an idempotency key to POST /payments,Ask finance for provider sandbox keys" \
  --risks "Combined capture is untested against the real provider sandbox" \
  --source-refs state/current-plan.md,spec/payment-flow.md >/dev/null

printf '# Status\n\nBuild phase, at risk: 2 of 6 tasks done, median checkout 48s against a 30s target. See `hp brief`.\n' > "$P/state/status.md"
"$HP" project validate --path "$P" --format text
