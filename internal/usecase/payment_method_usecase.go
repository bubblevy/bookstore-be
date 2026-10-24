package usecase

import (
	"context"
	"math"
	"strings"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/storage"
)

type PaymentMethodUsecase interface {
	GetAll(ctx context.Context, onlyActive bool, amount int) (*dto.PaymentMethodsResponse, error)
	GetByID(ctx context.Context, id uint) (*dto.PaymentMethodDetailResponse, error)
	Update(ctx context.Context, id uint, req dto.UpdatePaymentMethodRequest) (*dto.PaymentMethodDetailResponse, error)
}

type paymentMethodUsecase struct {
	repo       repository.PaymentMethodRepository
	storageSvc storage.StorageService
}

func NewPaymentMethodUsecase(repo repository.PaymentMethodRepository, storageSvc ...storage.StorageService) PaymentMethodUsecase {
	var s storage.StorageService
	if len(storageSvc) > 0 {
		s = storageSvc[0]
	}
	return &paymentMethodUsecase{repo: repo, storageSvc: s}
}

func (u *paymentMethodUsecase) formatFullImageURL(img string) string {
	clean := strings.TrimSpace(img)
	if clean == "" {
		return ""
	}
	if strings.HasPrefix(clean, "http://") || strings.HasPrefix(clean, "https://") {
		return clean
	}
	if u.storageSvc != nil {
		return u.storageSvc.GetURL(clean)
	}
	return "/api/v1/static/" + strings.TrimLeft(clean, "/")
}

func (u *paymentMethodUsecase) GetAll(ctx context.Context, onlyActive bool, amount int) (*dto.PaymentMethodsResponse, error) {
	if amount < 0 {
		amount = 0
	}

	methods, err := u.repo.FindAll(ctx, onlyActive)
	if err != nil {
		return nil, err
	}

	calcFee := func(feeType string, flatAmount int, percent float64) int {
		if feeType == "percentage" {
			if amount > 0 {
				return int(math.Round(float64(amount) * percent))
			}
			return 0
		}
		return flatAmount
	}

	categoryOrder := []string{"virtual_account", "ewallet", "convenience_store"}
	categoryLabels := map[string]string{
		"virtual_account":   "Virtual Account (Transfer Bank)",
		"ewallet":           "E-Wallet & QRIS",
		"convenience_store": "Gerai Retail (Convenience Store)",
	}

	groupsMap := make(map[string][]dto.PaymentMethodItemResponse)
	for _, m := range methods {
		item := dto.PaymentMethodItemResponse{
			ID:           m.ID,
			Code:         m.Code,
			Name:         m.Name,
			Category:     m.Category,
			FeeType:      m.FeeType,
			FeeAmount:    m.FeeAmount,
			FeePercent:   m.FeePercent,
			EstimatedFee: calcFee(m.FeeType, m.FeeAmount, m.FeePercent),
			Description:  m.Description,
			Image:        u.formatFullImageURL(m.Image),
			IsActive:     m.IsActive,
		}

		groupsMap[m.Category] = append(groupsMap[m.Category], item)
	}

	var groups []dto.PaymentMethodGroupResponse
	seenCategories := make(map[string]bool)

	// Add predefined order first
	for _, cat := range categoryOrder {
		items, exists := groupsMap[cat]
		if exists && len(items) > 0 {
			groups = append(groups, dto.PaymentMethodGroupResponse{
				Category: cat,
				Name:     categoryLabels[cat],
				Methods:  items,
			})
			seenCategories[cat] = true
		}
	}

	// Add any remaining dynamic categories
	for cat, items := range groupsMap {
		if !seenCategories[cat] && len(items) > 0 {
			label, exists := categoryLabels[cat]
			if !exists {
				label = strings.Title(strings.ReplaceAll(cat, "_", " "))
			}
			groups = append(groups, dto.PaymentMethodGroupResponse{
				Category: cat,
				Name:     label,
				Methods:  items,
			})
		}
	}

	return &dto.PaymentMethodsResponse{
		Groups: groups,
	}, nil
}

func (u *paymentMethodUsecase) GetByID(ctx context.Context, id uint) (*dto.PaymentMethodDetailResponse, error) {
	pm, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if pm == nil {
		return nil, entity.ErrPaymentMethodNotFound
	}

	return u.mapToPaymentMethodDetailResponse(pm), nil
}

func (u *paymentMethodUsecase) Update(ctx context.Context, id uint, req dto.UpdatePaymentMethodRequest) (*dto.PaymentMethodDetailResponse, error) {
	pm, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if pm == nil {
		return nil, entity.ErrPaymentMethodNotFound
	}

	if req.Name != nil {
		pm.Name = *req.Name
	}
	if req.Category != nil {
		pm.Category = *req.Category
	}
	if req.FeeType != nil {
		pm.FeeType = *req.FeeType
	}
	if req.FeeAmount != nil {
		pm.FeeAmount = *req.FeeAmount
	}
	if req.FeePercent != nil {
		pm.FeePercent = *req.FeePercent
	}

	oldImage := pm.Image
	if req.Image != nil {
		newImg := strings.TrimSpace(*req.Image)
		if u.storageSvc != nil && newImg != "" {
			if _, committedURL, err := u.storageSvc.CommitFile(ctx, newImg, "payments"); err == nil {
				newImg = committedURL
			}
		}
		pm.Image = newImg

		if u.storageSvc != nil && oldImage != "" && (newImg == "" || newImg != oldImage) {
			_ = u.storageSvc.Delete(ctx, oldImage)
		}
	}

	if req.Description != nil {
		pm.Description = *req.Description
	}
	if req.Instructions != nil {
		instructions := make(entity.PaymentInstructions, len(req.Instructions))
		for i, inst := range req.Instructions {
			instructions[i] = entity.PaymentInstruction{
				Title:   inst.Title,
				Content: inst.Content,
			}
		}
		pm.Instructions = instructions
	}
	if req.IsActive != nil {
		pm.IsActive = *req.IsActive
	}

	// Code is strictly immutable and cannot be edited
	if err := u.repo.Update(ctx, pm); err != nil {
		return nil, err
	}

	updated, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return u.mapToPaymentMethodDetailResponse(updated), nil
}

func (u *paymentMethodUsecase) mapToPaymentMethodDetailResponse(pm *entity.PaymentMethod) *dto.PaymentMethodDetailResponse {
	instructions := make([]dto.PaymentInstructionDTO, len(pm.Instructions))
	for i, inst := range pm.Instructions {
		instructions[i] = dto.PaymentInstructionDTO{
			Title:   inst.Title,
			Content: inst.Content,
		}
	}

	return &dto.PaymentMethodDetailResponse{
		ID:           pm.ID,
		Code:         pm.Code,
		Name:         pm.Name,
		Category:     pm.Category,
		FeeType:      pm.FeeType,
		FeeAmount:    pm.FeeAmount,
		FeePercent:   pm.FeePercent,
		Image:        u.formatFullImageURL(pm.Image),
		Description:  pm.Description,
		Instructions: instructions,
		IsActive:     pm.IsActive,
		CreatedAt:    pm.CreatedAt,
		UpdatedAt:    pm.UpdatedAt,
	}
}
