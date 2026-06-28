package types

type Principal struct {
	UserID             string             `json:"id"`
	Email              string             `json:"email"`
	IsGlobalAdmin      bool               `json:"is_global_admin"`
	VerificationStatus VerificationStatus `json:"verification_status"`
}

func (p Principal) IsVerified() bool {
	return p.VerificationStatus == VerificationVerified
}
