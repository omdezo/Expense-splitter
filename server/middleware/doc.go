// Package middleware holds the server's Echo middleware.
//
//   - jwt.go         validates Keycloak access tokens against the realm JWKS
//                    (signature, issuer, audience, expiry).
//   - auth.go        authentication middleware: requires a valid bearer token
//                    and puts the resolved identity in the request context.
//   - permissions.go authorization middleware (per-group roles, global admin) —
//                    not yet implemented.
package middleware
