# Changelog

## Unreleased

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
