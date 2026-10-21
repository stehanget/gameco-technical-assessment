package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	catalogconstants "gameco-technical-assessment/internal/constants/catalog"
	errorconst "gameco-technical-assessment/internal/constants/error"
	platformauth "gameco-technical-assessment/internal/platform/auth"
	"gameco-technical-assessment/internal/platform/faults"
	"gameco-technical-assessment/internal/platform/identifier"
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

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service { return &Service{repository: repository} }

type cursorPosition struct {
	Version   int    `json:"v"`
	SortBy    string `json:"sort_by"`
	SortOrder string `json:"sort_order"`
	Filter    string `json:"filter"`
	Value     string `json:"value"`
	Tie       string `json:"tie"`
}

type listResult struct {
	Items      []ProductResponse
	HasMore    bool
	NextCursor *string
	PerPage    int
}

type sellerListResult struct {
	Items      []SellerProductResponse
	HasMore    bool
	NextCursor *string
	PerPage    int
}

func (s *Service) List(ctx context.Context, request ListQuery) (listResult, error) {
	params, cursor, err := normalizeListQuery(request)
	if err != nil {
		return listResult{}, err
	}
	params.Cursor = cursor

	rows, hasMore, err := s.repository.List(ctx, params)
	if err != nil {
		return listResult{}, err
	}

	items := make([]ProductResponse, 0, len(rows))
	for _, row := range rows {
		item, err := mapProductView(row)
		if err != nil {
			return listResult{}, err
		}
		items = append(items, item)
	}

	var nextCursor *string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		value, err := cursorValue(params.SortBy, last)
		if err != nil {
			return listResult{}, err
		}
		encoded, err := encodeCursor(cursorPosition{
			Version:   1,
			SortBy:    params.SortBy,
			SortOrder: params.SortOrder,
			Filter:    filterFingerprint(params),
			Value:     value,
			Tie:       last.PublicID,
		})
		if err != nil {
			return listResult{}, err
		}
		nextCursor = &encoded
	}

	return listResult{Items: items, HasMore: hasMore, NextCursor: nextCursor, PerPage: params.PerPage}, nil
}

func (s *Service) FindByPublicID(ctx context.Context, publicID string) (ProductResponse, error) {
	parsed, err := uuid.Parse(publicID)
	if err != nil {
		return ProductResponse{}, faults.New(faults.KindNotFound, errorconst.CatalogProductNotFound, err)
	}
	row, err := s.repository.FindByPublicID(ctx, parsed.String())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ProductResponse{}, faults.New(faults.KindNotFound, errorconst.CatalogProductNotFound, err)
		}
		return ProductResponse{}, err
	}
	return mapProductView(row)
}

func (s *Service) ListBySeller(ctx context.Context, actor platformauth.Actor, request ListQuery) (sellerListResult, error) {
	sellerID := actor.UserID
	params, cursor, err := normalizeListQuery(request)
	if err != nil {
		return sellerListResult{}, err
	}
	params.Cursor = cursor

	rows, hasMore, err := s.repository.ListBySeller(ctx, sellerID, params)
	if err != nil {
		return sellerListResult{}, err
	}

	items := make([]SellerProductResponse, 0, len(rows))
	for _, row := range rows {
		item, err := mapSellerProductView(row)
		if err != nil {
			return sellerListResult{}, err
		}
		items = append(items, item)
	}

	var nextCursor *string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		value, err := cursorValue(params.SortBy, last)
		if err != nil {
			return sellerListResult{}, err
		}
		encoded, err := encodeCursor(cursorPosition{Version: 1, SortBy: params.SortBy, SortOrder: params.SortOrder, Filter: filterFingerprint(params), Value: value, Tie: last.PublicID})
		if err != nil {
			return sellerListResult{}, err
		}
		nextCursor = &encoded
	}

	return sellerListResult{Items: items, HasMore: hasMore, NextCursor: nextCursor, PerPage: params.PerPage}, nil
}

func (s *Service) FindSellerProductByPublicID(ctx context.Context, actor platformauth.Actor, publicID string) (SellerProductResponse, error) {
	sellerID := actor.UserID
	parsed, err := uuid.Parse(publicID)
	if err != nil {
		return SellerProductResponse{}, faults.New(faults.KindNotFound, errorconst.CatalogProductNotFound, err)
	}
	row, err := s.repository.FindByPublicIDForSeller(ctx, sellerID, parsed.String())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SellerProductResponse{}, faults.New(faults.KindNotFound, errorconst.CatalogProductNotFound, err)
		}
		return SellerProductResponse{}, err
	}
	return mapSellerProductView(row)
}

func (s *Service) CategoryOptions(ctx context.Context) ([]CategoryOptionResponse, error) {
	rows, err := s.repository.ListCategoryOptions(ctx)
	if err != nil {
		return nil, err
	}
	options := make([]CategoryOptionResponse, 0, len(rows))
	for _, row := range rows {
		options = append(options, CategoryOptionResponse{PublicID: row.PublicID, Slug: row.Slug, Name: row.Name})
	}
	return options, nil
}

func (s *Service) CreateProduct(ctx context.Context, actor platformauth.Actor, request CreateProductRequest) (SellerProductResponse, error) {
	sellerID := actor.UserID
	category, err := s.repository.FindCategoryByPublicID(ctx, request.CategoryPublicID.String())
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return SellerProductResponse{}, err
		}
		return SellerProductResponse{}, faults.New(faults.KindInvalidInput, errorconst.InvalidRequest, err)
	}
	publicID, err := identifier.NewPublicID()
	if err != nil {
		return SellerProductResponse{}, err
	}
	product := &Product{PublicID: publicID, SellerID: sellerID, CategoryID: category.ID, Name: strings.TrimSpace(request.Name), Description: strings.TrimSpace(request.Description), PriceAmount: request.PriceAmount, StockQuantity: request.StockQuantity, Status: catalogconstants.ProductStatusActive}
	if product.Name == "" || strings.TrimSpace(product.Description) == "" {
		return SellerProductResponse{}, faults.New(faults.KindInvalidInput, errorconst.InvalidRequest, nil)
	}
	if product.PriceAmount < 0 || product.StockQuantity < 0 {
		return SellerProductResponse{}, faults.New(faults.KindInvalidInput, errorconst.InvalidRequest, nil)
	}
	if err := s.repository.CreateProduct(ctx, product); err != nil {
		return SellerProductResponse{}, err
	}
	return s.FindSellerProductByPublicID(ctx, actor, publicID.String())
}

func (s *Service) UpdateProduct(ctx context.Context, actor platformauth.Actor, publicID string, request UpdateProductRequest) (SellerProductResponse, error) {
	sellerID := actor.UserID
	parsed, err := uuid.Parse(publicID)
	if err != nil {
		return SellerProductResponse{}, faults.New(faults.KindNotFound, errorconst.CatalogProductNotFound, err)
	}
	existing, err := s.repository.FindByPublicIDForSeller(ctx, sellerID, parsed.String())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SellerProductResponse{}, faults.New(faults.KindNotFound, errorconst.CatalogProductNotFound, err)
		}
		return SellerProductResponse{}, err
	}
	name := existing.Name
	description := existing.Description
	priceAmount := existing.PriceAmount
	stockQuantity := existing.StockQuantity
	values := map[string]any{}
	if request.Name != nil {
		name = strings.TrimSpace(*request.Name)
		values["name"] = name
	}
	if request.Description != nil {
		description = strings.TrimSpace(*request.Description)
		values["description"] = description
	}
	if request.PriceAmount != nil {
		priceAmount = *request.PriceAmount
		values["price_amount"] = priceAmount
	}
	if request.StockQuantity != nil {
		stockQuantity = *request.StockQuantity
		values["stock_quantity"] = stockQuantity
	}
	if request.CategoryPublicID != nil {
		category, categoryErr := s.repository.FindCategoryByPublicID(ctx, request.CategoryPublicID.String())
		if categoryErr != nil {
			if !errors.Is(categoryErr, gorm.ErrRecordNotFound) {
				return SellerProductResponse{}, categoryErr
			}
			return SellerProductResponse{}, faults.New(faults.KindInvalidInput, errorconst.InvalidRequest, categoryErr)
		}
		values["category_id"] = category.ID
	}
	if len(values) == 0 {
		return SellerProductResponse{}, faults.New(faults.KindInvalidInput, errorconst.InvalidRequest, nil)
	}
	if name == "" || description == "" || priceAmount < 0 || stockQuantity < 0 {
		return SellerProductResponse{}, faults.New(faults.KindInvalidInput, errorconst.InvalidRequest, nil)
	}
	if err := s.repository.UpdateProduct(ctx, sellerID, parsed.String(), values); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SellerProductResponse{}, faults.New(faults.KindNotFound, errorconst.CatalogProductNotFound, err)
		}
		return SellerProductResponse{}, err
	}
	return s.FindSellerProductByPublicID(ctx, actor, parsed.String())
}

func normalizeListQuery(request ListQuery) (listParams, *cursorPosition, error) {
	sortBy := request.SortBy
	if sortBy == "" {
		sortBy = defaultSortBy
	}
	if sortBy != "created_at" && sortBy != "name" && sortBy != "price" {
		return listParams{}, nil, faults.New(faults.KindInvalidInput, errorconst.CatalogInvalidSort, nil)
	}

	sortOrder := request.SortOrder
	if sortOrder == "" {
		sortOrder = defaultSortOrder
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		return listParams{}, nil, faults.New(faults.KindInvalidInput, errorconst.CatalogInvalidSort, nil)
	}

	perPage := request.PerPage
	if perPage == 0 {
		perPage = defaultPerPage
	}
	if perPage < 1 || perPage > maxPerPage {
		return listParams{}, nil, faults.New(faults.KindInvalidInput, errorconst.InvalidRequest, nil)
	}
	if request.MinPrice != nil && *request.MinPrice < 0 || request.MaxPrice != nil && *request.MaxPrice < 0 {
		return listParams{}, nil, faults.New(faults.KindInvalidInput, errorconst.InvalidRequest, nil)
	}
	if request.MinPrice != nil && request.MaxPrice != nil && *request.MinPrice > *request.MaxPrice {
		return listParams{}, nil, faults.New(faults.KindInvalidInput, errorconst.InvalidRequest, nil)
	}
	if len(request.Query) > 255 || len(request.Category) > 160 {
		return listParams{}, nil, faults.New(faults.KindInvalidInput, errorconst.InvalidRequest, nil)
	}

	params := listParams{
		Query:     strings.TrimSpace(request.Query),
		Category:  strings.TrimSpace(request.Category),
		MinPrice:  request.MinPrice,
		MaxPrice:  request.MaxPrice,
		SortBy:    sortBy,
		SortOrder: sortOrder,
		PerPage:   perPage,
	}
	if request.Cursor == "" {
		return params, nil, nil
	}

	cursor, err := decodeCursor(request.Cursor)
	if err != nil || cursor.SortBy != sortBy || cursor.SortOrder != sortOrder || cursor.Filter != filterFingerprint(params) {
		return listParams{}, nil, faults.New(faults.KindInvalidInput, errorconst.CatalogInvalidCursor, err)
	}
	return params, &cursor, nil
}

func filterFingerprint(params listParams) string {
	minPrice := ""
	if params.MinPrice != nil {
		minPrice = strconv.FormatInt(*params.MinPrice, 10)
	}
	maxPrice := ""
	if params.MaxPrice != nil {
		maxPrice = strconv.FormatInt(*params.MaxPrice, 10)
	}
	value := strings.Join([]string{params.Query, params.Category, minPrice, maxPrice, params.SortBy, params.SortOrder}, "\x00")
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
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
	return cursor, nil
}

func cursorValue(sortBy string, row productView) (string, error) {
	switch sortBy {
	case "name":
		return row.Name, nil
	case "price":
		return strconv.FormatInt(row.PriceAmount, 10), nil
	case "created_at":
		return timestamp.New(row.CreatedAt).String(), nil
	default:
		return "", fmt.Errorf("unsupported catalog sort: %s", sortBy)
	}
}

func mapProductView(row productView) (ProductResponse, error) {
	publicID, err := uuid.Parse(row.PublicID)
	if err != nil {
		return ProductResponse{}, err
	}
	return ProductResponse{
		PublicID:    publicID,
		Name:        row.Name,
		Description: row.Description,
		PriceAmount: row.PriceAmount,
		Stock:       row.StockQuantity,
		Category:    CategoryResponse{Slug: row.CategorySlug, Name: row.CategoryName},
		CreatedAt:   timestamp.New(row.CreatedAt),
		UpdatedAt:   timestamp.New(row.UpdatedAt),
	}, nil
}

func mapSellerProductView(row productView) (SellerProductResponse, error) {
	publicID, err := uuid.Parse(row.PublicID)
	if err != nil {
		return SellerProductResponse{}, err
	}
	categoryPublicID, err := uuid.Parse(row.CategoryPublicID)
	if err != nil {
		return SellerProductResponse{}, err
	}
	return SellerProductResponse{
		PublicID:    publicID,
		Name:        row.Name,
		Description: row.Description,
		PriceAmount: row.PriceAmount,
		Stock:       row.StockQuantity,
		Status:      row.Status,
		Category:    SellerCategory{PublicID: categoryPublicID, Slug: row.CategorySlug, Name: row.CategoryName},
		CreatedAt:   timestamp.New(row.CreatedAt),
		UpdatedAt:   timestamp.New(row.UpdatedAt),
	}, nil
}
