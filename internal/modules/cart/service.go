package cart

import (
	"context"
	"errors"

	cartconstants "gameco-technical-assessment/internal/constants/cart"
	catalogconstants "gameco-technical-assessment/internal/constants/catalog"
	errorconst "gameco-technical-assessment/internal/constants/error"
	"gameco-technical-assessment/internal/platform/auth"
	"gameco-technical-assessment/internal/platform/faults"
	"gameco-technical-assessment/internal/platform/identifier"
	"gameco-technical-assessment/internal/platform/timestamp"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service { return &Service{repository: repository} }

func (s *Service) Get(ctx context.Context, actor auth.Actor) (CartResponse, error) {
	view, err := s.repository.ReadActiveCart(ctx, actor.UserID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return CartResponse{}, err
		}
		return s.createEmptyCart(ctx, actor.UserID)
	}
	if view.Cart.ID == 0 {
		return s.createEmptyCart(ctx, actor.UserID)
	}
	return toResponse(view), nil
}

func (s *Service) SetItem(ctx context.Context, actor auth.Actor, productPublicID string, request SetItemRequest) (CartResponse, error) {
	productID, err := uuid.Parse(productPublicID)
	if err != nil {
		return CartResponse{}, faults.New(faults.KindInvalidInput, errorconst.InvalidRequest, err)
	}
	cartID, err := identifier.NewPublicID()
	if err != nil {
		return CartResponse{}, err
	}
	view, err := s.repository.SetItem(ctx, actor.UserID, productID.String(), request.Quantity, cartID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return CartResponse{}, faults.New(faults.KindNotFound, errorconst.CatalogProductNotFound, err)
	}
	if errors.Is(err, errInsufficientStock) {
		return CartResponse{}, faults.New(faults.KindConflict, errorconst.Conflict, err)
	}
	if err != nil {
		return CartResponse{}, err
	}
	return toResponse(view), nil
}

func (s *Service) DeleteItem(ctx context.Context, actor auth.Actor, productPublicID string) (CartResponse, error) {
	if _, err := uuid.Parse(productPublicID); err != nil {
		return CartResponse{}, faults.New(faults.KindInvalidInput, errorconst.InvalidRequest, err)
	}
	view, err := s.repository.DeleteItem(ctx, actor.UserID, productPublicID)
	if err != nil {
		return CartResponse{}, err
	}
	if view.Cart.ID == 0 {
		return s.createEmptyCart(ctx, actor.UserID)
	}
	return toResponse(view), nil
}

func (s *Service) createEmptyCart(ctx context.Context, userID uint) (CartResponse, error) {
	publicID, err := identifier.NewPublicID()
	if err != nil {
		return CartResponse{}, err
	}
	cart, err := s.repository.FindOrCreateActiveCart(ctx, userID, publicID)
	if err != nil {
		return CartResponse{}, err
	}
	return toResponse(cartView{Cart: cart}), nil
}

func toResponse(view cartView) CartResponse {
	items := make([]ItemResponse, 0, len(view.Rows))
	unavailableItems := make([]ItemResponse, 0)
	var total int64
	for _, row := range view.Rows {
		subtotal := row.PriceAmount * int64(row.Quantity)
		item := ItemResponse{
			ProductPublicID: uuid.MustParse(row.ProductPublicID),
			Name:            row.Name,
			PriceAmount:     row.PriceAmount,
			Quantity:        row.Quantity,
			SubtotalAmount:  subtotal,
			StockQuantity:   row.StockQuantity,
			Availability:    cartconstants.AvailabilityAvailable,
		}
		var unavailableReason string
		switch {
		case row.DeletedAt != nil:
			unavailableReason = cartconstants.UnavailableReasonProductDeleted
		case row.Status != catalogconstants.ProductStatusActive:
			unavailableReason = cartconstants.UnavailableReasonProductInactive
		case row.Quantity > row.StockQuantity:
			unavailableReason = cartconstants.UnavailableReasonInsufficientStock
		}
		if unavailableReason != "" {
			item.Availability = cartconstants.AvailabilityUnavailable
			item.UnavailableReason = &unavailableReason
			unavailableItems = append(unavailableItems, item)
			continue
		}
		total += subtotal
		items = append(items, item)
	}
	return CartResponse{
		PublicID:         view.Cart.PublicID,
		Status:           view.Cart.Status,
		Items:            items,
		UnavailableItems: unavailableItems,
		TotalAmount:      total,
		CreatedAt:        timestamp.New(view.Cart.CreatedAt),
		UpdatedAt:        timestamp.New(view.Cart.UpdatedAt),
	}
}
