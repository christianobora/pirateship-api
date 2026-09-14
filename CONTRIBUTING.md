# Contributing

Thanks for contributing.

1. Open an issue before making a large API change.
2. Keep the public surface backwards-compatible whenever possible.
3. Add unit tests using `httptest`; tests must never purchase or refund a live
   label.
4. Run `make check` before opening a pull request.
5. Use `gofmt`/`goimports`, wrap errors with context, and keep mutations free
   from automatic retries.

Never commit captured production responses, session cookies, addresses,
tracking numbers, payment-source details, or label URLs. Sanitize fixtures.
