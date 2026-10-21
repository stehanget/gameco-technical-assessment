package checkout

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	errorconst "gameco-technical-assessment/internal/constants/error"
	"gameco-technical-assessment/internal/platform/auth"
	"gameco-technical-assessment/internal/platform/faults"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct{ repository *Repository }

func NewService(repository *Repository) *Service { return &Service{repository: repository} }

func (s *Service) Create(ctx context.Context, actor auth.Actor, key string) (OrderResponse, error) {
	if strings.TrimSpace(key) == "" || len(key) > 255 {
		return OrderResponse{}, faults.New(errorconst.CheckoutIdempotencyKeyRequired, nil)
	}
	response, err := s.repository.Checkout(ctx, actor.UserID, key, RequestHash())
	if errors.Is(err, errIdempotencyConflict) {
		return OrderResponse{}, faults.New(errorconst.CheckoutIdempotencyConflict, err)
	}
	if errors.Is(err, errEmptyCart) {
		return OrderResponse{}, faults.New(errorconst.CheckoutEmptyCart, err)
	}
	if errors.Is(err, errUnavailableCart) {
		return OrderResponse{}, faults.New(errorconst.CheckoutCartUnavailable, err)
	}
	if errors.Is(err, errInsufficientStock) {
		return OrderResponse{}, faults.New(errorconst.CheckoutInsufficientStock, err)
	}
	if errors.Is(err, errSelfPurchase) {
		return OrderResponse{}, faults.New(errorconst.SelfPurchaseNotAllowed, err)
	}
	if err != nil {
		return OrderResponse{}, err
	}
	return response, nil
}

func (s *Service) Detail(ctx context.Context, actor auth.Actor, publicID string) (OrderResponse, error) {
	parsed, err := uuid.Parse(publicID)
	if err != nil {
		return OrderResponse{}, faults.New(errorconst.OrderNotFound, err)
	}
	response, err := s.repository.FindOrderByPublicID(ctx, actor.UserID, parsed)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return OrderResponse{}, faults.New(errorconst.OrderNotFound, err)
	}
	if err != nil {
		return OrderResponse{}, err
	}
	return response, nil
}

func (s *Service) ProcessWebhook(ctx context.Context, request PaymentWebhookRequest, body []byte, now time.Time) error {
	if err := request.Validate(); errors.Is(err, errWebhookMalformedPayload) {
		return faults.New(errorconst.PaymentWebhookMalformedPayload, err)
	} else if err != nil {
		return faults.New(errorconst.PaymentWebhookInvalidEvent, err)
	}
	sum := sha256.Sum256(body)
	err := s.repository.ProcessWebhook(ctx, request, hex.EncodeToString(sum[:]), now)
	switch {
	case errors.Is(err, errWebhookEventConflict):
		return faults.New(errorconst.PaymentWebhookEventConflict, err)
	case errors.Is(err, errWebhookInvalidEvent):
		return faults.New(errorconst.PaymentWebhookInvalidEvent, err)
	case errors.Is(err, errWebhookPaymentNotFound):
		return faults.New(errorconst.PaymentWebhookPaymentNotFound, err)
	case errors.Is(err, errWebhookAmountMismatch):
		return faults.New(errorconst.PaymentWebhookAmountMismatch, err)
	case err != nil:
		return err
	default:
		return nil
	}
}
