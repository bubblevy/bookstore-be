package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/pagination"
)

type DiscountTierInput struct {
	MinSpend    float64
	RewardType  string
	RewardValue float64
	MaxReward   *float64
}

type CreateDiscountInput struct {
	Code              string
	Name              string
	Description       *string
	DiscountType      string
	DiscountValue     float64
	MinPurchase       float64
	MaxDiscountAmount *float64
	Quota             int
	IsAllBooks        bool
	IsActive          *bool
	StartDate         time.Time
	EndDate           time.Time
	BookIDs           []uint
	Tiers             []DiscountTierInput
}

type UpdateDiscountInput struct {
	Code              string
	Name              string
	Description       *string
	DiscountType      string
	DiscountValue     float64
	MinPurchase       float64
	MaxDiscountAmount *float64
	Quota             int
	IsAllBooks        bool
	IsActive          *bool
	StartDate         time.Time
	EndDate           time.Time
	BookIDs           []uint
	Tiers             []DiscountTierInput
}

type DiscountTierResult struct {
	ID          uint     `json:"id"`
	MinSpend    float64  `json:"min_spend"`
	RewardType  string   `json:"reward_type"`
	RewardValue float64  `json:"reward_value"`
	MaxReward   *float64 `json:"max_reward,omitempty"`
}

type DiscountBookItem struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

type DiscountListItem struct {
	ID            uint      `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	DiscountType  string    `json:"discount_type"`
	DiscountValue float64   `json:"discount_value"`
	MinPurchase   float64   `json:"min_purchase"`
	Quota         int       `json:"quota"`
	UsedCount     int       `json:"used_count"`
	IsAllBooks    bool      `json:"is_all_books"`
	IsActive      bool      `json:"is_active"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
}

type DiscountResult struct {
	ID                uint                 `json:"id"`
	Code              string               `json:"code"`
	Name              string               `json:"name"`
	Description       *string              `json:"description,omitempty"`
	DiscountType      string               `json:"discount_type"`
	DiscountValue     float64              `json:"discount_value"`
	MinPurchase       float64              `json:"min_purchase"`
	MaxDiscountAmount *float64             `json:"max_discount_amount,omitempty"`
	Quota             int                  `json:"quota"`
	UsedCount         int                  `json:"used_count"`
	IsAllBooks        bool                 `json:"is_all_books"`
	IsActive          bool                 `json:"is_active"`
	StartDate         time.Time            `json:"start_date"`
	EndDate           time.Time            `json:"end_date"`
	Books             []DiscountBookItem   `json:"books,omitempty"`
	Tiers             []DiscountTierResult `json:"tiers,omitempty"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
}

type CheckDiscountResult struct {
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	DiscountAmount float64 `json:"discount_amount"`
	FinalAmount    float64 `json:"final_amount"`
}

type CartItemInput struct {
	BookID   uint
	Quantity int
}

type DiscountUsecase interface {
	Create(ctx context.Context, input CreateDiscountInput) (*DiscountResult, error)
	FindAll(ctx context.Context, filter repository.DiscountFilter, pq pagination.Query) ([]DiscountListItem, pagination.Meta, error)
	FindByID(ctx context.Context, id uint) (*DiscountResult, error)
	FindByCode(ctx context.Context, code string) (*DiscountResult, error)
	Update(ctx context.Context, id uint, input UpdateDiscountInput) (*DiscountResult, error)
	Delete(ctx context.Context, id uint) error
	ValidateAndCalculate(ctx context.Context, code string, items []CartItemInput) (*CheckDiscountResult, error)
}

type discountUsecase struct {
	discountRepo repository.DiscountRepository
	bookRepo     repository.BookRepository
}

func NewDiscountUsecase(discountRepo repository.DiscountRepository, bookRepo repository.BookRepository) DiscountUsecase {
	return &discountUsecase{
		discountRepo: discountRepo,
		bookRepo:     bookRepo,
	}
}

func (u *discountUsecase) toResponse(d *entity.Discount) *DiscountResult {
	books := make([]DiscountBookItem, 0, len(d.Books))
	for _, b := range d.Books {
		books = append(books, DiscountBookItem{
			ID:    b.ID,
			Title: b.Title,
			Slug:  b.Slug,
		})
	}

	tiers := make([]DiscountTierResult, 0, len(d.Tiers))
	for _, t := range d.Tiers {
		tiers = append(tiers, DiscountTierResult{
			ID:          t.ID,
			MinSpend:    t.MinSpend,
			RewardType:  string(t.RewardType),
			RewardValue: t.RewardValue,
			MaxReward:   t.MaxReward,
		})
	}

	return &DiscountResult{
		ID:                d.ID,
		Code:              d.Code,
		Name:              d.Name,
		Description:       d.Description,
		DiscountType:      string(d.DiscountType),
		DiscountValue:     d.DiscountValue,
		MinPurchase:       d.MinPurchase,
		MaxDiscountAmount: d.MaxDiscountAmount,
		Quota:             d.Quota,
		UsedCount:         d.UsedCount,
		IsAllBooks:        d.GetIsAllBooks(),
		IsActive:          d.IsActive,
		StartDate:         d.StartDate,
		EndDate:           d.EndDate,
		Books:             books,
		Tiers:             tiers,
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
	}
}

func (u *discountUsecase) mapTiers(inputs []DiscountTierInput) []entity.DiscountTier {
	tiers := make([]entity.DiscountTier, 0, len(inputs))
	for _, t := range inputs {
		tiers = append(tiers, entity.DiscountTier{
			MinSpend:    t.MinSpend,
			RewardType:  entity.DiscountType(t.RewardType),
			RewardValue: t.RewardValue,
			MaxReward:   t.MaxReward,
		})
	}
	return tiers
}

func (u *discountUsecase) Create(ctx context.Context, input CreateDiscountInput) (*DiscountResult, error) {
	codeUpper := strings.ToUpper(strings.TrimSpace(input.Code))
	exists, err := u.discountRepo.ExistsByCode(ctx, codeUpper)
	if err != nil {
		return nil, fmt.Errorf("failed checking code: %w", err)
	}
	if exists {
		return nil, entity.ErrDiscountCodeConflict
	}

	if input.DiscountType == string(entity.DiscountTypeTiered) && len(input.Tiers) == 0 {
		return nil, entity.ErrDiscountTierRequired
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	if !input.IsAllBooks && len(input.BookIDs) > 0 {
		count, err := u.bookRepo.CountExistingIDs(ctx, input.BookIDs)
		if err != nil {
			return nil, fmt.Errorf("failed validating book ids: %w", err)
		}
		if int(count) != len(input.BookIDs) {
			return nil, entity.ErrDiscountBookNotFound
		}
	}

	isAllBooks := input.IsAllBooks

	discount := &entity.Discount{
		Code:              codeUpper,
		Name:              input.Name,
		Description:       input.Description,
		DiscountType:      entity.DiscountType(input.DiscountType),
		DiscountValue:     input.DiscountValue,
		MinPurchase:       input.MinPurchase,
		MaxDiscountAmount: input.MaxDiscountAmount,
		Quota:             input.Quota,
		IsAllBooks:        &isAllBooks,
		IsActive:          isActive,
		StartDate:         input.StartDate,
		EndDate:           input.EndDate,
	}

	tiers := u.mapTiers(input.Tiers)

	if err := u.discountRepo.Create(ctx, discount, input.BookIDs, tiers); err != nil {
		return nil, fmt.Errorf("failed creating discount: %w", err)
	}

	res, err := u.discountRepo.FindByID(ctx, discount.ID)
	if err != nil {
		return nil, err
	}
	return u.toResponse(res), nil
}

func (u *discountUsecase) FindAll(ctx context.Context, filter repository.DiscountFilter, pq pagination.Query) ([]DiscountListItem, pagination.Meta, error) {
	filter.Limit = pq.PerPage
	filter.Offset = pq.Offset()

	discounts, total, err := u.discountRepo.FindAll(ctx, filter)
	if err != nil {
		return nil, pagination.Meta{}, fmt.Errorf("failed fetching discounts: %w", err)
	}

	items := make([]DiscountListItem, 0, len(discounts))
	for _, d := range discounts {
		items = append(items, DiscountListItem{
			ID:            d.ID,
			Code:          d.Code,
			Name:          d.Name,
			DiscountType:  string(d.DiscountType),
			DiscountValue: d.DiscountValue,
			MinPurchase:   d.MinPurchase,
			Quota:         d.Quota,
			UsedCount:     d.UsedCount,
			IsAllBooks:    d.GetIsAllBooks(),
			IsActive:      d.IsActive,
			StartDate:     d.StartDate,
			EndDate:       d.EndDate,
		})
	}

	meta := pagination.BuildMeta(total, pq.Page, pq.PerPage)
	return items, meta, nil
}

func (u *discountUsecase) FindByID(ctx context.Context, id uint) (*DiscountResult, error) {
	discount, err := u.discountRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed getting discount: %w", err)
	}
	if discount == nil {
		return nil, entity.ErrDiscountNotFound
	}
	return u.toResponse(discount), nil
}

func (u *discountUsecase) FindByCode(ctx context.Context, code string) (*DiscountResult, error) {
	codeUpper := strings.ToUpper(strings.TrimSpace(code))
	discount, err := u.discountRepo.FindByCode(ctx, codeUpper)
	if err != nil {
		return nil, fmt.Errorf("failed getting discount: %w", err)
	}
	if discount == nil {
		return nil, entity.ErrDiscountNotFound
	}
	return u.toResponse(discount), nil
}

func (u *discountUsecase) Update(ctx context.Context, id uint, input UpdateDiscountInput) (*DiscountResult, error) {
	discount, err := u.discountRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed getting discount: %w", err)
	}
	if discount == nil {
		return nil, entity.ErrDiscountNotFound
	}

	codeUpper := strings.ToUpper(strings.TrimSpace(input.Code))
	exists, err := u.discountRepo.ExistsByCode(ctx, codeUpper, id)
	if err != nil {
		return nil, fmt.Errorf("failed checking code: %w", err)
	}
	if exists {
		return nil, entity.ErrDiscountCodeConflict
	}

	if input.DiscountType == string(entity.DiscountTypeTiered) && len(input.Tiers) == 0 {
		return nil, entity.ErrDiscountTierRequired
	}

	if !input.IsAllBooks && len(input.BookIDs) > 0 {
		count, err := u.bookRepo.CountExistingIDs(ctx, input.BookIDs)
		if err != nil {
			return nil, fmt.Errorf("failed validating book ids: %w", err)
		}
		if int(count) != len(input.BookIDs) {
			return nil, entity.ErrDiscountBookNotFound
		}
	}

	isAllBooks := input.IsAllBooks

	discount.Code = codeUpper
	discount.Name = input.Name
	discount.Description = input.Description
	discount.DiscountType = entity.DiscountType(input.DiscountType)
	discount.DiscountValue = input.DiscountValue
	discount.MinPurchase = input.MinPurchase
	discount.MaxDiscountAmount = input.MaxDiscountAmount
	discount.Quota = input.Quota
	discount.IsAllBooks = &isAllBooks
	if input.IsActive != nil {
		discount.IsActive = *input.IsActive
	}
	discount.StartDate = input.StartDate
	discount.EndDate = input.EndDate

	tiers := u.mapTiers(input.Tiers)

	if err := u.discountRepo.Update(ctx, discount, input.BookIDs, tiers); err != nil {
		return nil, fmt.Errorf("failed updating discount: %w", err)
	}

	res, err := u.discountRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return u.toResponse(res), nil
}

func (u *discountUsecase) Delete(ctx context.Context, id uint) error {
	discount, err := u.discountRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed getting discount: %w", err)
	}
	if discount == nil {
		return entity.ErrDiscountNotFound
	}

	if err := u.discountRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed deleting discount: %w", err)
	}
	return nil
}

func (u *discountUsecase) ValidateAndCalculate(ctx context.Context, code string, items []CartItemInput) (*CheckDiscountResult, error) {
	codeUpper := strings.ToUpper(strings.TrimSpace(code))
	discount, err := u.discountRepo.FindByCode(ctx, codeUpper)
	if err != nil {
		return nil, fmt.Errorf("failed finding discount: %w", err)
	}
	if discount == nil {
		return nil, entity.ErrDiscountNotFound
	}

	if err := discount.IsValidNow(); err != nil {
		return nil, err
	}

	bookIDs := make([]uint, 0, len(items))
	qtyMap := make(map[uint]int)
	for _, it := range items {
		bookIDs = append(bookIDs, it.BookID)
		qtyMap[it.BookID] += it.Quantity
	}

	books, err := u.bookRepo.FindByIDs(ctx, bookIDs)
	if err != nil {
		return nil, fmt.Errorf("failed fetching book prices: %w", err)
	}

	if len(books) != len(qtyMap) {
		return nil, entity.ErrDiscountBookNotFound
	}

	var totalCartAmount float64
	var eligibleSubtotal float64

	eligibleBookMap := make(map[uint]bool)
	for _, b := range discount.Books {
		eligibleBookMap[b.ID] = true
	}

	for _, book := range books {
		qty := qtyMap[book.ID]
		itemTotal := book.Price * float64(qty)
		totalCartAmount += itemTotal

		if discount.GetIsAllBooks() || eligibleBookMap[book.ID] {
			eligibleSubtotal += itemTotal
		}
	}

	if eligibleSubtotal == 0 {
		return nil, entity.ErrDiscountNotApplicable
	}

	deduction, err := discount.CalculateDiscount(eligibleSubtotal)
	if err != nil {
		return nil, err
	}

	finalAmount := totalCartAmount - deduction
	if finalAmount < 0 {
		finalAmount = 0
	}

	return &CheckDiscountResult{
		Code:           discount.Code,
		Name:           discount.Name,
		DiscountAmount: deduction,
		FinalAmount:    finalAmount,
	}, nil
}
