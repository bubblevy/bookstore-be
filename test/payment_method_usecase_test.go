package test

import (
	"context"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPaymentMethodRepository struct {
	mock.Mock
}

func (m *MockPaymentMethodRepository) FindAll(ctx context.Context, onlyActive bool) ([]entity.PaymentMethod, error) {
	args := m.Called(ctx, onlyActive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.PaymentMethod), args.Error(1)
}

func (m *MockPaymentMethodRepository) FindByID(ctx context.Context, id uint) (*entity.PaymentMethod, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.PaymentMethod), args.Error(1)
}

func (m *MockPaymentMethodRepository) FindByCode(ctx context.Context, code string) (*entity.PaymentMethod, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.PaymentMethod), args.Error(1)
}

func (m *MockPaymentMethodRepository) Update(ctx context.Context, pm *entity.PaymentMethod) error {
	args := m.Called(ctx, pm)
	return args.Error(0)
}

func TestPaymentMethodUsecase_GetAll(t *testing.T) {
	t.Run("success returns grouped payment methods with instructions", func(t *testing.T) {
		mockRepo := new(MockPaymentMethodRepository)
		uc := usecase.NewPaymentMethodUsecase(mockRepo)

		sampleMethods := []entity.PaymentMethod{
			{
				ID:          1,
				Code:        "bca_va",
				Name:        "BCA Virtual Account",
				Category:    "virtual_account",
				FeeType:     "fixed",
				FeeAmount:   5000,
				FeePercent:  0,
				Image:       "payments/bca_logo.svg",
				Description: "Bayar via ATM BCA",
				Instructions: entity.PaymentInstructions{
					{Title: "Melalui Internet Banking", Content: "<ol><li>Login</li></ol>"},
				},
				IsActive: true,
			},
			{
				ID:          2,
				Code:        "qris",
				Name:        "QRIS",
				Category:    "ewallet",
				FeeType:     "percentage",
				FeeAmount:   0,
				FeePercent:  0.007,
				Image:       "payments/qris_logo.svg",
				Description: "Scan QR",
				Instructions: entity.PaymentInstructions{
					{Title: "Melalui Semua E-Wallet", Content: "<ol><li>Scan</li></ol>"},
				},
				IsActive: false, // in maintenance!
			},
		}

		mockRepo.On("FindAll", mock.Anything, false).Return(sampleMethods, nil).Once()

		res, err := uc.GetAll(context.Background(), false, 100000)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Len(t, res.Groups, 2)

		// First group is Virtual Account
		assert.Equal(t, "virtual_account", res.Groups[0].Category)
		assert.Len(t, res.Groups[0].Methods, 1)
		assert.Equal(t, "bca_va", res.Groups[0].Methods[0].Code)
		assert.Equal(t, 5000, res.Groups[0].Methods[0].EstimatedFee)
		assert.True(t, res.Groups[0].Methods[0].IsActive)

		// Second group is E-Wallet
		assert.Equal(t, "ewallet", res.Groups[1].Category)
		assert.Len(t, res.Groups[1].Methods, 1)
		assert.Equal(t, "qris", res.Groups[1].Methods[0].Code)
		assert.Equal(t, 700, res.Groups[1].Methods[0].EstimatedFee)
		assert.False(t, res.Groups[1].Methods[0].IsActive) // owner set maintenance

		mockRepo.AssertExpectations(t)
	})
}

func TestPaymentMethodUsecase_GetByID(t *testing.T) {
	t.Run("success returns payment method detail", func(t *testing.T) {
		mockRepo := new(MockPaymentMethodRepository)
		uc := usecase.NewPaymentMethodUsecase(mockRepo)

		pm := &entity.PaymentMethod{
			ID:          1,
			Code:        "bri_va",
			Name:        "BRI Virtual Account (BRIVA)",
			Category:    "virtual_account",
			FeeType:     "fixed",
			FeeAmount:   5000,
			Instructions: entity.PaymentInstructions{
				{Title: "Melalui Internet Banking", Content: "<ol><li>Login BRI</li></ol>"},
				{Title: "Melalui Aplikasi BRImo", Content: "<ol><li>Login BRImo</li></ol>"},
				{Title: "Melalui ATM BRI", Content: "<ol><li>ATM BRI</li></ol>"},
			},
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo.On("FindByID", mock.Anything, uint(1)).Return(pm, nil).Once()

		res, err := uc.GetByID(context.Background(), 1)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "bri_va", res.Code)
		assert.Len(t, res.Instructions, 3)

		mockRepo.AssertExpectations(t)
	})

	t.Run("returns not found error", func(t *testing.T) {
		mockRepo := new(MockPaymentMethodRepository)
		uc := usecase.NewPaymentMethodUsecase(mockRepo)

		mockRepo.On("FindByID", mock.Anything, uint(99)).Return(nil, nil).Once()

		res, err := uc.GetByID(context.Background(), 99)
		assert.ErrorIs(t, err, entity.ErrPaymentMethodNotFound)
		assert.Nil(t, res)

		mockRepo.AssertExpectations(t)
	})
}

func TestPaymentMethodUsecase_Update(t *testing.T) {
	t.Run("success toggles is_active maintenance and updates instructions without changing code", func(t *testing.T) {
		mockRepo := new(MockPaymentMethodRepository)
		uc := usecase.NewPaymentMethodUsecase(mockRepo)

		existing := &entity.PaymentMethod{
			ID:          1,
			Code:        "shopeepay",
			Name:        "ShopeePay",
			Category:    "ewallet",
			FeeType:     "percentage",
			FeePercent:  0.007,
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockRepo.On("FindByID", mock.Anything, uint(1)).Return(existing, nil).Once()

		inactiveFalse := false
		newName := "ShopeePay Maintenance"
		req := dto.UpdatePaymentMethodRequest{
			Name:     &newName,
			IsActive: &inactiveFalse,
			Instructions: []dto.PaymentInstructionDTO{
				{Title: "Cara Melakukan Pembayaran", Content: "<ol><li>Klik Bayar</li></ol>"},
			},
		}

		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(pm *entity.PaymentMethod) bool {
			return pm.ID == 1 && pm.Code == "shopeepay" && pm.IsActive == false && pm.Name == "ShopeePay Maintenance"
		})).Return(nil).Once()

		updatedPM := &entity.PaymentMethod{
			ID:        1,
			Code:      "shopeepay",
			Name:      "ShopeePay Maintenance",
			Category:  "ewallet",
			IsActive:  false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		mockRepo.On("FindByID", mock.Anything, uint(1)).Return(updatedPM, nil).Once()

		res, err := uc.Update(context.Background(), 1, req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "shopeepay", res.Code) // Code cannot be modified
		assert.False(t, res.IsActive)

		mockRepo.AssertExpectations(t)
	})

	t.Run("success commits uploaded image and deletes old image and formats full URL", func(t *testing.T) {
		mockRepo := new(MockPaymentMethodRepository)
		mockStorage := new(MockStorageService)
		uc := usecase.NewPaymentMethodUsecase(mockRepo, mockStorage)

		existing := &entity.PaymentMethod{
			ID:        1,
			Code:      "qris",
			Name:      "QRIS",
			Image:     "payments/old_qris.png",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo.On("FindByID", mock.Anything, uint(1)).Return(existing, nil).Once()

		tempImg := "uploads/new_qris.png"
		committedImg := "payments/new_qris.png"
		fullImgURL := "http://localhost:8080/api/v1/static/payments/new_qris.png"

		mockStorage.On("CommitFile", mock.Anything, tempImg, "payments").Return("payments/new_qris.png", committedImg, nil).Once()
		mockStorage.On("Delete", mock.Anything, "payments/old_qris.png").Return(nil).Once()
		mockStorage.On("GetURL", committedImg).Return(fullImgURL).Once()

		req := dto.UpdatePaymentMethodRequest{
			Image: &tempImg,
		}

		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(pm *entity.PaymentMethod) bool {
			return pm.ID == 1 && pm.Image == committedImg
		})).Return(nil).Once()

		updatedPM := &entity.PaymentMethod{
			ID:        1,
			Code:      "qris",
			Name:      "QRIS",
			Image:     committedImg,
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		mockRepo.On("FindByID", mock.Anything, uint(1)).Return(updatedPM, nil).Once()

		res, err := uc.Update(context.Background(), 1, req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, fullImgURL, res.Image)

		mockRepo.AssertExpectations(t)
		mockStorage.AssertExpectations(t)
	})
}
