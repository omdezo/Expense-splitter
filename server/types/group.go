package types

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type GroupStatus string

const (
	GroupOpen    GroupStatus = "open"
	GroupClosed  GroupStatus = "closed"
	GroupSettled GroupStatus = "settled"
)

type Group struct {
	ID                  string      `json:"id"`
	Name                string      `json:"name"`
	StartDate           time.Time   `json:"start_date"`
	EndDate             time.Time   `json:"end_date"`
	Status              GroupStatus `json:"status"`
	InviteToken         string      `json:"invite_token"`
	StatusToken         string      `json:"status_token,omitempty"`
	ExpectedMemberCount *int        `json:"expected_member_count,omitempty"`
	CreatedBy           string      `json:"created_by"`
	CreatedAt           time.Time   `json:"created_at"`
}

type CreateGroupRequest struct {
	Name                string    `json:"name"`
	StartDate           time.Time `json:"start_date"`
	EndDate             time.Time `json:"end_date"`
	ExpectedMemberCount *int      `json:"expected_member_count"`
	AdminUserID         *string   `json:"admin_user_id"`
}

func (r *CreateGroupRequest) Validate() APIError {
	if strings.TrimSpace(r.Name) == "" {
		return NewBadRequestError("name is required")
	}
	if r.StartDate.IsZero() {
		return NewBadRequestError("start_date is required")
	}
	if r.EndDate.IsZero() {
		return NewBadRequestError("end_date is required")
	}
	if r.EndDate.Before(r.StartDate) {
		return NewBadRequestError("end_date must be on or after start_date")
	}
	if r.ExpectedMemberCount != nil && *r.ExpectedMemberCount < 0 {
		return NewBadRequestError("expected_member_count must be >= 0")
	}
	if r.AdminUserID != nil {
		if _, err := uuid.Parse(*r.AdminUserID); err != nil {
			return NewBadRequestError("admin_user_id must be a valid uuid")
		}
	}
	return nil
}

// UpdateGroupRequest replaces the editable metadata of an open group. The
// invite token, status and creator are not editable here.
type UpdateGroupRequest struct {
	Name                string    `json:"name"`
	StartDate           time.Time `json:"start_date"`
	EndDate             time.Time `json:"end_date"`
	ExpectedMemberCount *int      `json:"expected_member_count"`
}

func (r *UpdateGroupRequest) Validate() APIError {
	if strings.TrimSpace(r.Name) == "" {
		return NewBadRequestError("name is required")
	}
	if r.StartDate.IsZero() {
		return NewBadRequestError("start_date is required")
	}
	if r.EndDate.IsZero() {
		return NewBadRequestError("end_date is required")
	}
	if r.EndDate.Before(r.StartDate) {
		return NewBadRequestError("end_date must be on or after start_date")
	}
	if r.ExpectedMemberCount != nil && *r.ExpectedMemberCount < 0 {
		return NewBadRequestError("expected_member_count must be >= 0")
	}
	return nil
}

// GroupListItem is a group the caller belongs to, plus their own role/status.
type GroupListItem struct {
	Group
	Role             MembershipRole   `json:"role"`
	MembershipStatus MembershipStatus `json:"membership_status"`
}

// GroupDetail is a group's metadata plus its member list (req #6, distinct from
// the financial summary in #12).
type GroupDetail struct {
	Group
	Members []MembershipView `json:"members"`
}
