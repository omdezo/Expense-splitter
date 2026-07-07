package types

import (
	"strings"
	"time"
	"unicode/utf8"
)

type ProofType string

const (
	ProofImage ProofType = "image"
	ProofText  ProofType = "text"
)

const proofNoteMaxLen = 2000

type SubmitProofRequest struct {
	Note string `json:"note"`
}

// ProofView is the metadata half of proof retrieval (req #18); the raw image
// bytes have their own endpoint. The storage key stays internal.
type ProofView struct {
	PaymentID string    `json:"payment_id"`
	ProofType ProofType `json:"proof_type"`
	Sha256    string    `json:"sha256,omitempty"`
	ByteSize  int64     `json:"byte_size,omitempty"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (r *SubmitProofRequest) Validate() APIError {
	if strings.TrimSpace(r.Note) == "" {
		return NewBadRequestError("note is required (image proofs are not supported yet)")
	}
	if utf8.RuneCountInString(r.Note) > proofNoteMaxLen {
		return NewBadRequestError("note must be at most 2000 characters")
	}
	return nil
}
