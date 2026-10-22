# Decision: one payment call

Decided by checkout-team.

Authorize and capture happen in one `POST /payments` call when the provider supports immediate capture. Orders that need manual review keep the two-step flow. Reason: the second round trip costs a median 9 seconds in the timing baseline.
