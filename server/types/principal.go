package types

// Principal is the authenticated caller resolved against the database: an
// Identity matched to its users row. Authorization decisions use this.
//
// It holds only account-level facts. Roles are NOT here because they are
// per-group (see MembershipRole) and are resolved with a group_id, not globally.
type Principal struct {
	UserID             string             `json:"id"` // users.id — the app's id, not the token 'sub'
	Email              string             `json:"email"`
	IsGlobalAdmin      bool               `json:"is_global_admin"`
	VerificationStatus VerificationStatus `json:"verification_status"`
}

// IsVerified reports whether the account may participate (join groups, record
// expenses, be settled against). Only verified accounts may.
func (p Principal) IsVerified() bool {
	return p.VerificationStatus == VerificationVerified
}
