package types

type VerificationStatus string

const (
	VerificationRegistered VerificationStatus = "registered"
	VerificationPending    VerificationStatus = "pending_verification"
	VerificationVerified   VerificationStatus = "verified"
	VerificationRejected   VerificationStatus = "rejected"
)

type MembershipRole string

const (
	RoleGroupAdmin MembershipRole = "group_admin"
	RoleMember     MembershipRole = "member"
)

type MembershipStatus string

const (
	MembershipRequested MembershipStatus = "requested"
	MembershipApproved  MembershipStatus = "approved"
	MembershipRejected  MembershipStatus = "rejected"
)
