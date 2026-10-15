package test

import (
	"context"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockDiscountRepository struct {
	mock.Mock
}

func (m *MockDiscountRepository) Create(ctx context.Context, discount *entity.Discount, bookIDs []uint, tiers []entity.DiscountTier) error {
	return m.Called(ctx, discount, bookIDs, tiers).Error(0)
}

func (m *MockDiscountRepository) FindByID(ctx context.Context, id uint) (*entity.Discount, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Discount), args.Error(1)
}

func (m *MockDiscountRepository) FindByCode(ctx context.Context, code string) (*entity.Discount, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Discount), args.Error(1)
}

func (m *MockDiscountRepository) FindAll(ctx context.Context, filter repository.DiscountFilter) ([]entity.Discount, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]entity.Discount), args.Get(1).(int64), args.Error(2)
}

func (m *MockDiscountRepository) Update(ctx context.Context, discount *entity.Discount, bookIDs []uint, tiers []entity.DiscountTier) error {
	return m.Called(ctx, discount, bookIDs, tiers).Error(0)
}

func (m *MockDiscountRepository) Delete(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockDiscountRepository) ExistsByCode(ctx context.Context, code string, excludeID ...uint) (bool, error) {
	var args mock.Arguments
	if len(excludeID) > 0 {
		args = m.Called(ctx, code, excludeID[0])
	} else {
		args = m.Called(ctx, code)
	}
	return args.Bool(0), args.Error(1)
}

func (m *MockDiscountRepository) IncrementUsedCount(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}

func TestDiscountUsecase_Create_AutoUppercaseAndSuccess(t *testing.T) {
	mockDiscRepo := new(MockDiscountRepository)
	mockBookRepo := new(MockBookRepository)
	uc := usecase.NewDiscountUsecase(mockDiscRepo, mockBookRepo)
	ctx := context.Background()

	isAllBooks := true
	req := usecase.CreateDiscountInput{
		Code:          "diskonhemat10", // Input huruf kecil
		Name:          "Promo Baca Hemat",
		DiscountType:  "PERCENTAGE",
		DiscountValue: 10,
		MinPurchase:   50000,
		IsAllBooks:    isAllBooks,
		StartDate:     time.Now().Add(-1 * time.Hour),
		EndDate:       time.Now().Add(24 * time.Hour),
	}

	mockDiscRepo.On("ExistsByCode", ctx, "DISKONHEMAT10").Return(false, nil)
	mockDiscRepo.On("Create", ctx, mock.MatchedBy(func(d *entity.Discount) bool {
		return d.Code == "DISKONHEMAT10" && d.GetIsAllBooks() == true
	}), mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		d := args.Get(1).(*entity.Discount)
		d.ID = 1
	}).Return(nil)

	savedDiscount := &entity.Discount{
		Model:         gorm.Model{ID: 1},
		Code:          "DISKONHEMAT10",
		Name:          "Promo Baca Hemat",
		DiscountType:  entity.DiscountTypePercentage,
		DiscountValue: 10,
		MinPurchase:   50000,
		IsAllBooks:    &isAllBooks,
		IsActive:      true,
		StartDate:     req.StartDate,
		EndDate:       req.EndDate,
	}
	mockDiscRepo.On("FindByID", ctx, uint(1)).Return(savedDiscount, nil)

	res, err := uc.Create(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "DISKONHEMAT10", res.Code) // Terverifikasi tersimpan kapital
	mockDiscRepo.AssertExpectations(t)
}

func TestDiscountUsecase_Create_Tiered_MissingTiersError(t *testing.T) {
	mockDiscRepo := new(MockDiscountRepository)
	mockBookRepo := new(MockBookRepository)
	uc := usecase.NewDiscountUsecase(mockDiscRepo, mockBookRepo)
	ctx := context.Background()

	req := usecase.CreateDiscountInput{
		Code:         "TIEREDVOUCHER",
		Name:         "Tiered Promo",
		DiscountType: "TIERED",
		Tiers:        []usecase.DiscountTierInput{}, // Kosong
	}

	mockDiscRepo.On("ExistsByCode", ctx, "TIEREDVOUCHER").Return(false, nil)

	res, err := uc.Create(ctx, req)
	assert.ErrorIs(t, err, entity.ErrDiscountTierRequired)
	assert.Nil(t, res)
}

func TestDiscountUsecase_Create_SpecificBooks_BookNotFound(t *testing.T) {
	mockDiscRepo := new(MockDiscountRepository)
	mockBookRepo := new(MockBookRepository)
	uc := usecase.NewDiscountUsecase(mockDiscRepo, mockBookRepo)
	ctx := context.Background()

	req := usecase.CreateDiscountInput{
		Code:         "BOOKSPECIFIC",
		Name:         "Book Specific",
		DiscountType: "FIXED",
		IsAllBooks:   false,
		BookIDs:      []uint{1, 99}, // ID 99 tidak ada di database
	}

	mockDiscRepo.On("ExistsByCode", ctx, "BOOKSPECIFIC").Return(false, nil)
	mockBookRepo.On("CountExistingIDs", ctx, []uint{1, 99}).Return(int64(1), nil) // Hanya 1 buku ditemukan

	res, err := uc.Create(ctx, req)
	assert.ErrorIs(t, err, entity.ErrDiscountBookNotFound)
	assert.Nil(t, res)
}

func TestDiscountUsecase_ValidateAndCalculate_TieredDiscount(t *testing.T) {
	mockDiscRepo := new(MockDiscountRepository)
	mockBookRepo := new(MockBookRepository)
	uc := usecase.NewDiscountUsecase(mockDiscRepo, mockBookRepo)
	ctx := context.Background()

	isAllBooks := true
	maxReward20k := 20000.0
	voucher := &entity.Discount{
		Model:        gorm.Model{ID: 1},
		Code:         "TIEREDHEMAT",
		Name:         "Diskon Bertingkat",
		DiscountType: entity.DiscountTypeTiered,
		MinPurchase:  100000,
		IsAllBooks:   &isAllBooks,
		IsActive:     true,
		StartDate:    time.Now().Add(-1 * time.Hour),
		EndDate:      time.Now().Add(24 * time.Hour),
		Tiers: []entity.DiscountTier{
			{ID: 1, MinSpend: 100000, RewardType: entity.DiscountTypeFixed, RewardValue: 5000},
			{ID: 2, MinSpend: 120000, RewardType: entity.DiscountTypeFixed, RewardValue: 7000},
			{ID: 3, MinSpend: 150000, RewardType: entity.DiscountTypePercentage, RewardValue: 10, MaxReward: &maxReward20k},
		},
	}

	mockDiscRepo.On("FindByCode", ctx, "TIEREDHEMAT").Return(voucher, nil)

	// Belanja 2 pcs Buku seharga @Rp65.000 = Rp130.000 (Masuk Tier 2 -> Potongan Rp7.000)
	cartItems := []usecase.CartItemInput{
		{BookID: 1, Quantity: 2},
	}
	dbBooks := []entity.Book{
		{Model: gorm.Model{ID: 1}, Price: 65000, IsActive: true},
	}
	mockBookRepo.On("FindByIDs", ctx, []uint{1}).Return(dbBooks, nil)

	res, err := uc.ValidateAndCalculate(ctx, "tieredhemat", cartItems)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, 7000.0, res.DiscountAmount)
	assert.Equal(t, 123000.0, res.FinalAmount)
}

func TestDiscountUsecase_ValidateAndCalculate_MinPurchaseNotMet(t *testing.T) {
	mockDiscRepo := new(MockDiscountRepository)
	mockBookRepo := new(MockBookRepository)
	uc := usecase.NewDiscountUsecase(mockDiscRepo, mockBookRepo)
	ctx := context.Background()

	isAllBooks := true
	voucher := &entity.Discount{
		Model:         gorm.Model{ID: 1},
		Code:          "MIN100K",
		DiscountType:  entity.DiscountTypeFixed,
		DiscountValue: 10000,
		MinPurchase:   100000,
		IsAllBooks:    &isAllBooks,
		IsActive:      true,
		StartDate:     time.Now().Add(-1 * time.Hour),
		EndDate:       time.Now().Add(24 * time.Hour),
	}

	mockDiscRepo.On("FindByCode", ctx, "MIN100K").Return(voucher, nil)

	// Belanja hanya Rp40.000
	cartItems := []usecase.CartItemInput{
		{BookID: 1, Quantity: 1},
	}
	dbBooks := []entity.Book{
		{Model: gorm.Model{ID: 1}, Price: 40000, IsActive: true},
	}
	mockBookRepo.On("FindByIDs", ctx, []uint{1}).Return(dbBooks, nil)

	res, err := uc.ValidateAndCalculate(ctx, "MIN100K", cartItems)
	assert.ErrorIs(t, err, entity.ErrDiscountMinPurchase)
	assert.Nil(t, res)
}
