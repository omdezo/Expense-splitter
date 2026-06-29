# Tests

Go convention: **unit tests live next to the code** they test, as
`<file>_test.go` in the same package — that's also the only way to reach a
package's *unexported* functions. So the fast unit tests (e.g. the authorization
matrix in `services/authz_test.go`) are co-located, not here.

This `test/` tree is for tests that genuinely belong in a separate place because
they're black-box and need external resources:

- **`integration/`** — tests that hit a real Postgres (e.g. the `memberships`
  query behind `RequireGroupRole`, registration linking). Need the DB up.
- **`e2e/`** — full HTTP + Keycloak flows against the running stack.

Run everything from `server/` via the pinned toolchain (the local Go is too old):

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.25 go test ./...
```
