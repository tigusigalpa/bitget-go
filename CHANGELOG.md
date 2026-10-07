# Changelog

## Unreleased

## v1.2.0 — 2026-10-07

- Added the separate UTA deep-history `Market.GetHistoryCandles` endpoint with
  all seven selectors and a request maximum of 100; preserved exact strings,
  inclusive/exclusive responses, provider order and documented extra rounding
  candles without changing recent candles or funding-history methods.
- Added bounded immutable `models.RESTReceipt` evidence and per-call
  `GetHistoryCandlesWithReceipt`/`GetLiquidationsWithReceipt` variants, including
  exact bodies on decoding/API/read errors, local receipt time, safe request
  metadata, and explicit incomplete flags for oversized/interrupted responses.
- Added explicitly partial three-day liquidation pages with opaque cursor and
  original amount/event-time strings; no stable identity, units or exhaustive
  coverage is inferred. Audited public historical fills and did not substitute
  private account history or classic v2 routes for an undocumented public UTA API.
- Added source-linked synthetic fixtures, regression/boundary/pagination/error/
  cancellation tests, historical-data documentation and a public runnable example.

## v1.1.0

- Added lossless synchronous WebSocket raw-frame observation with receipt time,
  connection generation, observable subscription ACK/provider-error lifecycle,
  and terminal failure signalling for provenance-sensitive consumers.
- Added typed public market-data methods for candles, recent fills, and
  realized funding-rate history, preserving Bitget decimal values as strings.
- Added loopback tests for the new WebSocket contract and public REST routes.
- Expanded the README with a first-request walkthrough, public market-data,
  ACK/readiness, and exact raw-frame usage examples.
- Checked response and WebSocket close errors so the full lint configuration
  accepts production code, examples, and WebSocket tests.
