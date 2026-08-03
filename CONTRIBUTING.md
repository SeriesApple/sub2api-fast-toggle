# Contributing

Contributions are welcome when they keep the proxy small, auditable, and
provider-neutral.

1. Do not add API keys, personal paths, database dumps, request logs, or account
   identifiers to commits, tests, or issue reports.
2. Run `go test -race ./...` and `go vet ./...` from `proxy/`.
3. Preserve streaming behavior and never log authorization headers or bodies.
4. Document new environment variables in both README files and `.env.example`.
5. Keep Sub2API and OpenAI attribution in `NOTICE.md` and
   `THIRD_PARTY_NOTICES.md`.

By submitting a contribution, you agree that it may be distributed under the
MIT License used by this repository.
