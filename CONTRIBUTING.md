# Contributing

1. Fork the repository and create a branch off `main`.
2. Add or update tests for any behavior change (`go test ./...` must pass).
3. Run `make check` (fmt + lint + test) before opening a PR.
4. Every exported method must include a `Docs:` line in its docblock linking
   to the exact Bitget API documentation page it implements.
5. Open a pull request describing the change and, for new endpoints, the
   doc URL it covers. Update [docs/endpoints.md](docs/endpoints.md)
   accordingly.

Found a security issue? Email sovletig@gmail.com directly instead of
opening a public issue.
