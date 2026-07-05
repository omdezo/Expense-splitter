package types

import (
	"strings"
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

func (r *SubmitProofRequest) Validate() APIError {
	if strings.TrimSpace(r.Note) == "" {
		return NewBadRequestError("note is required (image proofs are not supported yet)")
	}
	if utf8.RuneCountInString(r.Note) > proofNoteMaxLen {
		return NewBadRequestError("note must be at most 2000 characters")
	}
	return nil
}
