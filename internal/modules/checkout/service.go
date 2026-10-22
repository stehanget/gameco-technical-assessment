package checkout

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	errorconst "gameco-technical-assessment/internal/constants/error"
	"gameco-technical-assessment/internal/platform/auth"
	"gameco-technical-assessment/internal/platform/faults"
	"gameco-technical-assessment/internal/platform/pagination"
	"gameco-technical-assessment/internal/platform/timestamp"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	defaultSortBy    = "created_at"
	defaultSortOrder = "desc"
	defaultPerPage   = 20
	maxPerPage       = 100
)

type Service struct{ repository *Repository }

func NewService(repository *Repository) *Service { return &Service{repository: repository} }

type cursorPosition struct {
	Version   int    `json:"v"`
	SortBy    string `json:"sort_by"`
	SortOrder string `json:"sort_order"`
	Filter    string `json:"filter"`
	Value     string `json:"value"`
	Tie       string `json:"tie"`
}

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

func (s *Service) ListSellerOrders(ctx context.Context, actor auth.Actor, request SellerOrderListQuery) (sellerOrderListResult, error) {
	params, cursor, err := normalizeSellerOrderQuery(request)
	if err != nil {
		return sellerOrderListResult{}, err
	}
	params.Cursor = cursor

	orders, items, hasMore, err := s.repository.ListOrdersBySeller(ctx, actor.UserID, params)
	if err != nil {
		return sellerOrderListResult{}, err
	}

	itemsByOrder := make(map[uint][]SellerOrderItemResponse)
	for _, item := range items {
		itemsByOrder[item.OrderID] = append(itemsByOrder[item.OrderID], SellerOrderItemResponse{
			ProductPublicID: item.ProductPublicID,
			ProductName:     item.ProductName,
			UnitPriceAmount: item.UnitPriceAmount,
			Quantity:        item.Quantity,
			SubtotalAmount:  item.SubtotalAmount,
		})
	}

	result := make([]SellerOrderResponse, 0, len(orders))
	for _, order := range orders {
		orderItems := itemsByOrder[order.OrderID]
		if orderItems == nil {
			orderItems = []SellerOrderItemResponse{}
		}
		result = append(result, SellerOrderResponse{
			PublicID:          order.PublicID,
			Status:            order.Status,
			SellerTotalAmount: order.SellerTotalAmount,
			PaymentStatus:     order.PaymentStatus,
			Items:             orderItems,
			CreatedAt:         timestamp.New(order.CreatedAt),
		})
	}

	var nextCursor *string
	if hasMore && len(orders) > 0 {
		last := orders[len(orders)-1]
		value := timestamp.New(last.CreatedAt).String()
		if params.SortBy == "seller_total" {
			value = strconv.FormatInt(last.SellerTotalAmount, 10)
		}
		encoded, err := encodeCursor(cursorPosition{
			Version:   1,
			SortBy:    params.SortBy,
			SortOrder: params.SortOrder,
			Filter:    filterFingerprint(params),
			Value:     value,
			Tie:       last.PublicID.String(),
		})
		if err != nil {
			return sellerOrderListResult{}, err
		}
		nextCursor = &encoded
	}
	return sellerOrderListResult{Items: result, PerPage: params.PerPage, HasMore: hasMore, NextCursor: nextCursor}, nil
}

func normalizeSellerOrderQuery(request SellerOrderListQuery) (sellerOrderListParams, *cursorPosition, error) {
	sortBy := request.SortBy
	if sortBy == "" {
		sortBy = defaultSortBy
	}
	if sortBy != "created_at" && sortBy != "seller_total" {
		return sellerOrderListParams{}, nil, faults.New(errorconst.InvalidRequest, nil)
	}

	sortOrder := request.SortOrder
	if sortOrder == "" {
		sortOrder = defaultSortOrder
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		return sellerOrderListParams{}, nil, faults.New(errorconst.InvalidRequest, nil)
	}

	perPage := request.PerPage
	if perPage == 0 {
		perPage = defaultPerPage
	}
	if perPage < 1 || perPage > maxPerPage || len(request.Query) > 255 || len(request.Status) > 64 || len(request.PaymentStatus) > 64 {
		return sellerOrderListParams{}, nil, faults.New(errorconst.InvalidRequest, nil)
	}

	params := sellerOrderListParams{
		Query:         strings.TrimSpace(request.Query),
		Status:        strings.TrimSpace(request.Status),
		PaymentStatus: strings.TrimSpace(request.PaymentStatus),
		SortBy:        sortBy,
		SortOrder:     sortOrder,
		PerPage:       perPage,
	}
	if request.Cursor == "" {
		return params, nil, nil
	}

	cursor, err := decodeCursor(request.Cursor)
	if err != nil || cursor.SortBy != sortBy || cursor.SortOrder != sortOrder || cursor.Filter != filterFingerprint(params) {
		return sellerOrderListParams{}, nil, faults.New(errorconst.InvalidRequest, err)
	}
	return params, &cursor, nil
}

func filterFingerprint(params sellerOrderListParams) string {
	return pagination.FilterFingerprint(map[string]string{
		"q":              params.Query,
		"status":         params.Status,
		"payment_status": params.PaymentStatus,
		"sort_by":        params.SortBy,
		"sort_order":     params.SortOrder,
	})
}

func encodeCursor(cursor cursorPosition) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeCursor(value string) (cursorPosition, error) {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return cursorPosition{}, err
	}
	var cursor cursorPosition
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return cursorPosition{}, err
	}
	if cursor.Version != 1 || cursor.SortBy == "" || cursor.SortOrder == "" || cursor.Filter == "" || cursor.Value == "" || cursor.Tie == "" {
		return cursorPosition{}, errors.New("invalid cursor payload")
	}
	if _, err := uuid.Parse(cursor.Tie); err != nil {
		return cursorPosition{}, err
	}
	return cursor, nil
}

type sellerOrderListResult struct {
	Items      []SellerOrderResponse
	PerPage    int
	HasMore    bool
	NextCursor *string
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
