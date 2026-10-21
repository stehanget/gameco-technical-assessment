package checkout

import (
	"context"
	"errors"
	"strings"

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
