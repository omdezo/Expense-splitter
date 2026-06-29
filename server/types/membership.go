package types

import (
	"time"

	"github.com/google/uuid"
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

type Membership struct {
	Role   MembershipRole
	Status MembershipStatus
}

func (m Membership) Active() bool { return m.Status == MembershipApproved }

func (m Membership) Satisfies(need MembershipRole) bool {
	if need == RoleGroupAdmin {
		return m.Role == RoleGroupAdmin
	}
	return true
}

type JoinGroupRequest struct {
	InviteToken string `json:"invite_token"`
}

func (r *JoinGroupRequest) Validate() APIError {
	if _, err := uuid.Parse(r.InviteToken); err != nil {
		return NewBadRequestError("invite_token must be a valid uuid")
	}
	return nil
}

type MembershipView struct {
	GroupID   string           `json:"group_id"`
	UserID    string           `json:"user_id"`
	Email     string           `json:"email,omitempty"`
	Role      MembershipRole   `json:"role"`
	Status    MembershipStatus `json:"status"`
	CreatedAt time.Time        `json:"created_at"`
}
