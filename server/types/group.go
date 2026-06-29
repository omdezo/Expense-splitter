package types

import (
	"strings"
	"time"
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
	ExpectedMemberCount *int        `json:"expected_member_count,omitempty"`
	CreatedBy           string      `json:"created_by"`
	CreatedAt           time.Time   `json:"created_at"`
}

type CreateGroupRequest struct {
	Name                string    `json:"name"`
	StartDate           time.Time `json:"start_date"`
	EndDate             time.Time `json:"end_date"`
	ExpectedMemberCount *int      `json:"expected_member_count"`
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
	return nil
}
