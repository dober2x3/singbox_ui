# Agent Rules

## Lint Before Commit

Always run linters and compilation before committing and fix all reported issues:

- **Go backend**:
  - Lint: `golangci-lint run ./...` in `server/`
  - Build: `go build ./...` in `server/`
- **Frontend**:
  - Lint: `npm run lint` in `frontend/`
  - Build: `npm run build` in `frontend/`

Do not commit if linters produce warnings/errors or compilation fails. Fix all findings first.

If `golangci-lint` is not installed, install it rather than skipping the check:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```
