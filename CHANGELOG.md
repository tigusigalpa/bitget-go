# Changelog

## Unreleased

- Added lossless synchronous WebSocket raw-frame observation with receipt time,
  connection generation, observable subscription ACK/provider-error lifecycle,
  and terminal failure signalling for provenance-sensitive consumers.
- Added typed public market-data methods for candles, recent fills, and
  realized funding-rate history, preserving Bitget decimal values as strings.
- Added loopback tests for the new WebSocket contract and public REST routes.
