package types

import (
	"encoding/json"
	"time"
)

type AuditEntryView struct {
	ID          int64           `json:"id"`
	ActorUserID *string         `json:"actor_user_id"`
	Action      string          `json:"action"`
	Before      json.RawMessage `json:"before"`
	After       json.RawMessage `json:"after"`
	CreatedAt   time.Time       `json:"created_at"`
}
