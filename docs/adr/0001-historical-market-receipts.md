# ADR-0001: Explicit historical routes with immutable per-call receipts

- Date: 2026-10-07
- Status: accepted

## Context

v1.1.0 at `83df6a08abfa7c4a4616dbdafbe662ae6ecb4307` implements recent candles
and funding history. Consumers need the official separate history-candles route
and exact response evidence without intercepting their injected HTTP transports.
The source audit is recorded in `testdata/market-history/README.md`.

## Alternatives

- Substitute recent candles for history: rejected because their routes and page
  limits differ and the result would misrepresent coverage.
- Re-encode typed data or intercept transport in consumers: rejected because
  neither gives the SDK a consistent bounded immutable evidence contract.
- Replace existing signatures with receipt-returning methods: rejected because
  it breaks callers using the v1.1.0 admitted limited-history methods.
- Add per-call companion methods and reuse the existing bounded HTTP transport:
  selected; local strict parsing is confined to the new endpoints.

## Decision

Expose `GetHistoryCandles` and `GetLiquidations` plus `WithReceipt` companions.
Receipts are immutable snapshots with copying accessors, HTTP status, local time
after body reading and before decoding, and allowlisted selector metadata without
headers/host/credentials. They retain at most 10 MiB. Incomplete reads/oversize
return an error and partial bounded receipt; errors before a response return nil.
Fully captured bytes do not imply valid JSON or complete market coverage.

NewClient's injected function signature remains compatible; a separate
NewClientWithReceipts wires optional evidence transport. No raw receipt is
fabricated when the legacy injected transport only provides typed results.

Keep request bounds unchanged and preserve all returned candles, including the
documented extra rounded interval. No automatic pagination, filtering or coverage
claim is added. Public UTA historical fills have no admitted documented route.
Liquidation pages remain explicitly partial, with unmodified opaque cursor,
original amount/time strings, no synthetic ID and no inferred unit conversion.

## Consequences and verification

Consumers must persist receipts, manage progress/overlapping pages and define
their own coverage admission. Retaining a receipt costs a bounded extra copy;
callers mutating accessor results cannot change saved evidence. Only new routes
use strict candle/page decoding; legacy route behaviour remains unchanged.
Fixtures test both possible equality behaviours rather than claiming a live
boundary guarantee. HTTP size/read/error/cancellation tests, immutable concurrent
access tests and legacy-route regression tests verify the decision.
