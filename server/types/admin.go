package types

import "time"

type AdminUserView struct {
	ID                 string             `json:"id"`
	Email              string             `json:"email"`
	IsGlobalAdmin      bool               `json:"is_global_admin"`
	VerificationStatus VerificationStatus `json:"verification_status"`
	Linked             bool               `json:"linked"`
	CreatedAt          time.Time          `json:"created_at"`
}

type UserMembershipView struct {
	GroupID   string           `json:"group_id"`
	GroupName string           `json:"group_name"`
	Role      MembershipRole   `json:"role"`
	Status    MembershipStatus `json:"status"`
	CreatedAt time.Time        `json:"created_at"`
}

type AdminUserDetail struct {
	AdminUserView
	Memberships []UserMembershipView `json:"memberships"`
}

type AdminGroupView struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	StartDate   time.Time   `json:"start_date"`
	EndDate     time.Time   `json:"end_date"`
	Status      GroupStatus `json:"status"`
	CreatedBy   string      `json:"created_by"`
	MemberCount int64       `json:"member_count"`
	TotalSpent  int64       `json:"total_spent"`
	CreatedAt   time.Time   `json:"created_at"`
}
