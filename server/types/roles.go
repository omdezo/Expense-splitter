package types

// These mirror the Postgres enums from the initial migration — the typed
// vocabulary for identity and authorization.

// VerificationStatus is an account's verification state (users.verification_status).
type VerificationStatus string

const (
	VerificationRegistered VerificationStatus = "registered"
	VerificationPending    VerificationStatus = "pending_verification"
	VerificationVerified   VerificationStatus = "verified"
	VerificationRejected   VerificationStatus = "rejected"
)

// MembershipRole is a caller's role WITHIN one group (memberships.role). Roles
// are per-(user, group): the same user can be group_admin of one trip and a
// member of another. The single global admin holds group-admin powers in every
// group implicitly, with no membership row.
type MembershipRole string

const (
	RoleGroupAdmin MembershipRole = "group_admin"
	RoleMember     MembershipRole = "member"
)

// MembershipStatus is the join state of a (user, group) pair (memberships.status).
// Only approved members participate in a group.
type MembershipStatus string

const (
	MembershipRequested MembershipStatus = "requested"
	MembershipApproved  MembershipStatus = "approved"
	MembershipRejected  MembershipStatus = "rejected"
)
