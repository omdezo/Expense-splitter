package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"

	"expense-splitter/database/repo"
	"expense-splitter/storage"
	"expense-splitter/types"
)

// sniffImage validates that data really is an image by its magic bytes
// (bonus #2) — a renamed binary fails no matter its extension or the
// Content-Type header the client claims.
func sniffImage(data []byte) (string, bool) {
	ct := http.DetectContentType(data)
	switch ct {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return ct, true
	}
	return ct, false
}

// SubmitImageProof is the image variant of SubmitProof (req #15 + bonus #2):
// validate magic bytes, hash, store the bytes in object storage, and record
// metadata — all behind the same state-machine transition as text proofs.
func (s *Services) SubmitImageProof(ctx context.Context, id types.Identity, paymentID string, data []byte) (*types.PaymentView, types.APIError) {
	contentType, ok := sniffImage(data)
	if !ok {
		return nil, types.NewBadRequestError(fmt.Sprintf("file is not a supported image (detected %s; jpeg/png/gif/webp only)", contentType))
	}

	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	size := int64(len(data))
	key := fmt.Sprintf("proofs/%s/%s", paymentID, hash)

	return s.transitionPayment(ctx, id, paymentID, actionSubmitProof, func(qtx *repo.Queries, p repo.GetPaymentRow) error {
		if err := s.store.Put(ctx, key, data, contentType); err != nil {
			if errors.Is(err, storage.ErrNotConfigured) {
				return err
			}
			return fmt.Errorf("upload proof image: %w", err)
		}
		if err := qtx.UnsetCurrentProof(ctx, paymentID); err != nil {
			return err
		}
		_, err := qtx.CreateProof(ctx, repo.CreateProofParams{
			PaymentID:  paymentID,
			ProofType:  types.ProofImage,
			Sha256:     &hash,
			ByteSize:   &size,
			StorageKey: &key,
		})
		return err
	})
}

// GetProof returns the current proof's metadata (req #18) — including the
// image size when the proof is an image.
func (s *Services) GetProof(ctx context.Context, id types.Identity, paymentID string) (*types.ProofView, types.APIError) {
	if apiErr := s.requireProofViewer(ctx, id, paymentID); apiErr != nil {
		return nil, apiErr
	}

	p, err := s.q.GetCurrentProof(ctx, paymentID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("no proof has been submitted for this payment")
	case err != nil:
		s.logger.Errorw("get proof: query", "error", err)
		return nil, types.NewServerError()
	}

	view := &types.ProofView{
		PaymentID: paymentID,
		ProofType: p.ProofType,
		CreatedAt: p.CreatedAt,
	}
	if p.Sha256 != nil {
		view.Sha256 = *p.Sha256
	}
	if p.ByteSize != nil {
		view.ByteSize = *p.ByteSize
	}
	if p.Note != nil {
		view.Note = *p.Note
	}
	return view, nil
}

// GetProofImage streams back the stored image bytes (req #18). A text-note
// proof has no bytes — that case is a 409, not an empty file.
func (s *Services) GetProofImage(ctx context.Context, id types.Identity, paymentID string) ([]byte, string, types.APIError) {
	if apiErr := s.requireProofViewer(ctx, id, paymentID); apiErr != nil {
		return nil, "", apiErr
	}

	p, err := s.q.GetCurrentProof(ctx, paymentID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, "", types.NewNotFoundError("no proof has been submitted for this payment")
	case err != nil:
		s.logger.Errorw("get proof image: query", "error", err)
		return nil, "", types.NewServerError()
	}
	if p.ProofType != types.ProofImage || p.StorageKey == nil {
		return nil, "", types.NewConflictError("the proof is a text note, not an image")
	}

	data, err := s.store.Get(ctx, *p.StorageKey)
	if err != nil {
		s.logger.Errorw("get proof image: fetch object", "error", err)
		return nil, "", types.NewServerError()
	}
	contentType, _ := sniffImage(data)
	return data, contentType, nil
}

// requireProofViewer allows the payment's parties and admins to see the proof:
// debtor, creditor, the group's admin, or the global admin.
func (s *Services) requireProofViewer(ctx context.Context, id types.Identity, paymentID string) types.APIError {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("proof viewer: resolve caller", "error", err)
		return types.NewServerError()
	}

	p, err := s.q.GetPayment(ctx, paymentID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return types.NewNotFoundError("payment not found")
	case err != nil:
		s.logger.Errorw("proof viewer: load payment", "error", err)
		return types.NewServerError()
	}

	if caller.IsGlobalAdmin || p.FromUserID == caller.UserID || p.ToUserID == caller.UserID {
		return nil
	}
	m, err := s.q.GetMembership(ctx, repo.GetMembershipParams{GroupID: p.GroupID, UserID: caller.UserID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return types.NewForbiddenError("you are not a participant in this payment")
	case err != nil:
		s.logger.Errorw("proof viewer: load membership", "error", err)
		return types.NewServerError()
	}
	if m.Role == types.RoleGroupAdmin && m.Status == types.MembershipApproved {
		return nil
	}
	return types.NewForbiddenError("you are not a participant in this payment")
}
