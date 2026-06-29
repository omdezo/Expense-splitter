package types

type VerificationStatus string

const (
	VerificationRegistered VerificationStatus = "registered"
	VerificationPending    VerificationStatus = "pending_verification"
	VerificationVerified   VerificationStatus = "verified"
	VerificationRejected   VerificationStatus = "rejected"
)
