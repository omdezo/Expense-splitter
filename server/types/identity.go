package types

// Identity is the authenticated caller as described by a validated access token.
// It is pure authentication data with no database lookup, so it is available
// even for a caller who holds a valid token but has no users row yet (e.g. while
// registering).
//
// Authorization facts deliberately live elsewhere:
//   - account-level facts (the internal user id, global-admin flag, verification
//     status) are loaded from the users table into a Principal.
//   - roles are PER-GROUP (memberships.role), so they are not a property of the
//     caller at all — they are resolved for a specific group_id per request.
type Identity struct {
	Subject           string // Keycloak 'sub' — NOT the application's users.id
	Email             string
	EmailVerified     bool
	Name              string
	PreferredUsername string
}
