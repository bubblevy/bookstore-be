package test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/biteship"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"github.com/bubblevy/bookstore/pkg/payment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ==========================================
// MOCK DEFINITIONS
// ==========================================

type MockCheckoutRepository struct {
	mock.Mock
}

func (m *MockCheckoutRepository) WithTx(tx *gorm.DB) repository.CheckoutRepository {
	args := m.Called(tx)
	if res := args.Get(0); res != nil {
		return res.(repository.CheckoutRepository)
	}
	return m
}

func (m *MockCheckoutRepository) GetCartItemsWithBooks(ctx context.Context, userID uint, cartItemIDs []uint) ([]entity.CartItem, error) {
	args := m.Called(ctx, userID, cartItemIDs)
	if res := args.Get(0); res != nil {
		return res.([]entity.CartItem), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCheckoutRepository) LockAndValidateBooks(ctx context.Context, bookIDs []uint) (map[uint]*entity.Book, error) {
	args := m.Called(ctx, bookIDs)
	if res := args.Get(0); res != nil {
		return res.(map[uint]*entity.Book), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCheckoutRepository) DeductBookStock(ctx context.Context, bookID uint, qty int) error {
	return m.Called(ctx, bookID, qty).Error(0)
}

func (m *MockCheckoutRepository) RestoreBookStock(ctx context.Context, bookID uint, qty int) error {
	return m.Called(ctx, bookID, qty).Error(0)
}

func (m *MockCheckoutRepository) DeleteCartItems(ctx context.Context, cartItemIDs []uint) error {
	return m.Called(ctx, cartItemIDs).Error(0)
}

type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) WithTx(tx *gorm.DB) repository.OrderRepository {
	args := m.Called(tx)
	if res := args.Get(0); res != nil {
		return res.(repository.OrderRepository)
	}
	return m
}

func (m *MockOrderRepository) CreateOrder(ctx context.Context, order *entity.Order, items []entity.OrderItem) error {
	return m.Called(ctx, order, items).Error(0)
}

func (m *MockOrderRepository) CreatePayment(ctx context.Context, p *entity.Payment) error {
	return m.Called(ctx, p).Error(0)
}

func (m *MockOrderRepository) FindByInvoice(ctx context.Context, invoice string) (*entity.Order, error) {
	args := m.Called(ctx, invoice)
	if res := args.Get(0); res != nil {
		return res.(*entity.Order), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrderRepository) FindByID(ctx context.Context, id uint) (*entity.Order, error) {
	args := m.Called(ctx, id)
	if res := args.Get(0); res != nil {
		return res.(*entity.Order), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrderRepository) FindUserOrders(ctx context.Context, userID uint, statusFilter string) ([]entity.Order, error) {
	args := m.Called(ctx, userID, statusFilter)
	if res := args.Get(0); res != nil {
		return res.([]entity.Order), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrderRepository) FindOrderDetailByInvoice(ctx context.Context, userID uint, invoice string) (*entity.Order, error) {
	args := m.Called(ctx, userID, invoice)
	if res := args.Get(0); res != nil {
		return res.(*entity.Order), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrderRepository) UpdateOrderStatus(ctx context.Context, orderID uint, status entity.OrderStatus) error {
	return m.Called(ctx, orderID, status).Error(0)
}

func (m *MockOrderRepository) UpdatePaymentStatus(ctx context.Context, orderID uint, status entity.PaymentStatus) error {
	return m.Called(ctx, orderID, status).Error(0)
}

func (m *MockOrderRepository) MarkAsReturned(ctx context.Context, orderID uint, reason string) error {
	return m.Called(ctx, orderID, reason).Error(0)
}

type MockAddressRepositoryForCheckout struct {
	mock.Mock
}

func (m *MockAddressRepositoryForCheckout) WithTx(tx *gorm.DB) repository.AddressRepository {
	return m
}

func (m *MockAddressRepositoryForCheckout) Create(ctx context.Context, address *entity.Address) error {
	return m.Called(ctx, address).Error(0)
}

func (m *MockAddressRepositoryForCheckout) FindByID(ctx context.Context, id uint) (*entity.Address, error) {
	args := m.Called(ctx, id)
	if res := args.Get(0); res != nil {
		return res.(*entity.Address), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAddressRepositoryForCheckout) FindByUserID(ctx context.Context, userID uint) ([]entity.Address, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]entity.Address), args.Error(1)
}

func (m *MockAddressRepositoryForCheckout) FindPrimaryByUserID(ctx context.Context, userID uint) (*entity.Address, error) {
	args := m.Called(ctx, userID)
	if res := args.Get(0); res != nil {
		return res.(*entity.Address), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAddressRepositoryForCheckout) Update(ctx context.Context, address *entity.Address) error {
	return m.Called(ctx, address).Error(0)
}

func (m *MockAddressRepositoryForCheckout) Delete(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockAddressRepositoryForCheckout) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAddressRepositoryForCheckout) ResetPrimaryStatus(ctx context.Context, userID uint) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *MockAddressRepositoryForCheckout) SetPrimary(ctx context.Context, userID, addressID uint) error {
	return m.Called(ctx, userID, addressID).Error(0)
}

type MockDiscountUsecaseForCheckout struct {
	mock.Mock
}

func (m *MockDiscountUsecaseForCheckout) Create(ctx context.Context, input usecase.CreateDiscountInput) (*usecase.DiscountResult, error) {
	args := m.Called(ctx, input)
	if res := args.Get(0); res != nil {
		return res.(*usecase.DiscountResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDiscountUsecaseForCheckout) FindAll(ctx context.Context, filter repository.DiscountFilter, pq pagination.Query) ([]usecase.DiscountListItem, pagination.Meta, error) {
	args := m.Called(ctx, filter, pq)
	return args.Get(0).([]usecase.DiscountListItem), args.Get(1).(pagination.Meta), args.Error(2)
}

func (m *MockDiscountUsecaseForCheckout) FindByID(ctx context.Context, id uint) (*usecase.DiscountResult, error) {
	args := m.Called(ctx, id)
	if res := args.Get(0); res != nil {
		return res.(*usecase.DiscountResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDiscountUsecaseForCheckout) FindByCode(ctx context.Context, code string) (*usecase.DiscountResult, error) {
	args := m.Called(ctx, code)
	if res := args.Get(0); res != nil {
		return res.(*usecase.DiscountResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDiscountUsecaseForCheckout) Update(ctx context.Context, id uint, input usecase.UpdateDiscountInput) (*usecase.DiscountResult, error) {
	args := m.Called(ctx, id, input)
	if res := args.Get(0); res != nil {
		return res.(*usecase.DiscountResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDiscountUsecaseForCheckout) Delete(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockDiscountUsecaseForCheckout) ValidateAndCalculate(ctx context.Context, code string, items []usecase.CartItemInput) (*usecase.CheckDiscountResult, error) {
	args := m.Called(ctx, code, items)
	if res := args.Get(0); res != nil {
		return res.(*usecase.CheckDiscountResult), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockDiscountRepositoryForCheckout struct {
	mock.Mock
}

func (m *MockDiscountRepositoryForCheckout) Create(ctx context.Context, discount *entity.Discount, bookIDs []uint, tiers []entity.DiscountTier) error {
	return m.Called(ctx, discount, bookIDs, tiers).Error(0)
}

func (m *MockDiscountRepositoryForCheckout) FindAll(ctx context.Context, filter repository.DiscountFilter) ([]entity.Discount, int64, error) {
	args := m.Called(ctx, filter)
	if res := args.Get(0); res != nil {
		return res.([]entity.Discount), args.Get(1).(int64), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func (m *MockDiscountRepositoryForCheckout) FindByID(ctx context.Context, id uint) (*entity.Discount, error) {
	args := m.Called(ctx, id)
	if res := args.Get(0); res != nil {
		return res.(*entity.Discount), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDiscountRepositoryForCheckout) FindByCode(ctx context.Context, code string) (*entity.Discount, error) {
	args := m.Called(ctx, code)
	if res := args.Get(0); res != nil {
		return res.(*entity.Discount), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDiscountRepositoryForCheckout) Update(ctx context.Context, discount *entity.Discount, bookIDs []uint, tiers []entity.DiscountTier) error {
	return m.Called(ctx, discount, bookIDs, tiers).Error(0)
}

func (m *MockDiscountRepositoryForCheckout) Delete(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockDiscountRepositoryForCheckout) ExistsByCode(ctx context.Context, code string, excludeID ...uint) (bool, error) {
	args := m.Called(ctx, code, excludeID)
	return args.Bool(0), args.Error(1)
}

func (m *MockDiscountRepositoryForCheckout) IncrementUsedCount(ctx context.Context, discountID uint) error {
	return m.Called(ctx, discountID).Error(0)
}

type MockBiteshipClientForCheckout struct {
	mock.Mock
}

func (m *MockBiteshipClientForCheckout) SearchArea(ctx context.Context, keyword string) ([]biteship.AreaItem, error) {
	args := m.Called(ctx, keyword)
	return args.Get(0).([]biteship.AreaItem), args.Error(1)
}

func (m *MockBiteshipClientForCheckout) GetCouriers(ctx context.Context) ([]biteship.CourierDetail, error) {
	args := m.Called(ctx)
	return args.Get(0).([]biteship.CourierDetail), args.Error(1)
}

func (m *MockBiteshipClientForCheckout) CalculateRatesByAreaID(ctx context.Context, destAreaID, couriers string, items []biteship.RateItem) ([]biteship.ShippingRatePricing, error) {
	args := m.Called(ctx, destAreaID, couriers, items)
	if res := args.Get(0); res != nil {
		return res.([]biteship.ShippingRatePricing), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockBiteshipClientForCheckout) CreateOrderByAreaID(ctx context.Context, req biteship.CreateOrderByAreaIDRequest) (*biteship.OrderResponse, error) {
	args := m.Called(ctx, req)
	if res := args.Get(0); res != nil {
		return res.(*biteship.OrderResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockBiteshipClientForCheckout) GetOrderByID(ctx context.Context, orderID string) (*biteship.OrderResponse, error) {
	args := m.Called(ctx, orderID)
	if res := args.Get(0); res != nil {
		return res.(*biteship.OrderResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockBiteshipClientForCheckout) CancelOrder(ctx context.Context, orderID string, reason string) (*biteship.CancelOrderResponse, error) {
	args := m.Called(ctx, orderID, reason)
	if res := args.Get(0); res != nil {
		return res.(*biteship.CancelOrderResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockBiteshipClientForCheckout) TrackByID(ctx context.Context, trackingID string) (*biteship.TrackingDetailResult, error) {
	args := m.Called(ctx, trackingID)
	if res := args.Get(0); res != nil {
		return res.(*biteship.TrackingDetailResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockBiteshipClientForCheckout) TrackWaybill(ctx context.Context, waybillID, courierCode string) (*biteship.TrackingDetailResult, error) {
	args := m.Called(ctx, waybillID, courierCode)
	if res := args.Get(0); res != nil {
		return res.(*biteship.TrackingDetailResult), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockMidtransServiceForCheckout struct {
	mock.Mock
}

func (m *MockMidtransServiceForCheckout) Charge(ctx context.Context, orderID string, grossAmount int64, paymentType string) (*payment.ChargeResponse, error) {
	args := m.Called(ctx, orderID, grossAmount, paymentType)
	if res := args.Get(0); res != nil {
		return res.(*payment.ChargeResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMidtransServiceForCheckout) VerifyNotification(orderID, statusCode, grossAmount, signatureKey string) bool {
	return m.Called(orderID, statusCode, grossAmount, signatureKey).Bool(0)
}

func setupGormSqlmock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	sqlDB, mock, err := sqlmock.New()
	assert.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	assert.NoError(t, err)

	return gormDB, mock
}

// ==========================================
// TEST SUITE
// ==========================================

func TestCheckoutUsecase_PreviewCheckout(t *testing.T) {
	userID := uint(1)
	addrID := uint(5)
	cartItemID := uint(10)
	bookID := uint(30)

	t.Run("success preview with explicit address_id and bca_va admin fee", func(t *testing.T) {
		gormDB, _ := setupGormSqlmock(t)
		mockCheckoutRepo := new(MockCheckoutRepository)
		mockOrderRepo := new(MockOrderRepository)
		mockAddressRepo := new(MockAddressRepositoryForCheckout)
		mockBiteship := new(MockBiteshipClientForCheckout)
		mockMidtrans := new(MockMidtransServiceForCheckout)

		uc := usecase.NewCheckoutUsecase(gormDB, mockCheckoutRepo, mockOrderRepo, mockAddressRepo, nil, nil, mockBiteship, mockMidtrans)

		dummyAddr := &entity.Address{
			UserID:         userID,
			BiteshipAreaID: "IDNP6IDNC148IDND843IDZ12250",
			CityName:       "Jakarta Selatan",
		}
		dummyAddr.ID = addrID
		mockAddressRepo.On("FindByID", mock.Anything, addrID).Return(dummyAddr, nil).Once()

		dummyCartItems := []entity.CartItem{
			{
				ID:       cartItemID,
				BookID:   bookID,
				Quantity: 2,
				Book: entity.Book{
					ID:          bookID,
					Title:       "Buku Go",
					Price:       100000.0,
					Stock:       10,
					WeightGrams: 500,
					IsActive:    true,
				},
			},
		}
		mockCheckoutRepo.On("GetCartItemsWithBooks", mock.Anything, userID, []uint{cartItemID}).Return(dummyCartItems, nil).Once()

		mockRates := []biteship.ShippingRatePricing{
			{CourierCode: "jne", CourierServiceCode: "reg", Price: 20000},
		}
		mockBiteship.On("CalculateRatesByAreaID", mock.Anything, dummyAddr.BiteshipAreaID, "jne", mock.Anything).
			Return(mockRates, nil).Once()

		input := usecase.PreviewCheckoutInput{
			CartItemIDs:    []uint{cartItemID},
			AddressID:      &addrID,
			Courier:        "jne",
			CourierService: "reg",
			PaymentMethod:  "bca_va", // admin fee: 5000.0
		}

		res, err := uc.PreviewCheckout(context.Background(), userID, input)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, float64(200000), res.ItemsSubtotal)
		assert.Equal(t, float64(20000), res.ShippingCost)
		assert.Equal(t, float64(5000), res.AdminFee)
		assert.Equal(t, float64(225000), res.TotalPayment)
	})

	t.Run("success preview with voucher discount calculation", func(t *testing.T) {
		gormDB, _ := setupGormSqlmock(t)
		mockCheckoutRepo := new(MockCheckoutRepository)
		mockOrderRepo := new(MockOrderRepository)
		mockAddressRepo := new(MockAddressRepositoryForCheckout)
		mockDiscountUC := new(MockDiscountUsecaseForCheckout)
		mockDiscountRepo := new(MockDiscountRepositoryForCheckout)
		mockBiteship := new(MockBiteshipClientForCheckout)
		mockMidtrans := new(MockMidtransServiceForCheckout)

		uc := usecase.NewCheckoutUsecase(gormDB, mockCheckoutRepo, mockOrderRepo, mockAddressRepo, mockDiscountUC, mockDiscountRepo, mockBiteship, mockMidtrans)

		dummyAddr := &entity.Address{
			UserID:         userID,
			BiteshipAreaID: "IDNP6IDNC148IDND843IDZ12250",
			CityName:       "Jakarta Selatan",
		}
		dummyAddr.ID = addrID
		mockAddressRepo.On("FindByID", mock.Anything, addrID).Return(dummyAddr, nil).Once()

		dummyCartItems := []entity.CartItem{
			{
				ID:       cartItemID,
				BookID:   bookID,
				Quantity: 2,
				Book: entity.Book{
					ID:          bookID,
					Title:       "Buku Go",
					Price:       100000.0,
					Stock:       10,
					WeightGrams: 500,
					IsActive:    true,
				},
			},
		}
		mockCheckoutRepo.On("GetCartItemsWithBooks", mock.Anything, userID, []uint{cartItemID}).Return(dummyCartItems, nil).Once()

		mockRates := []biteship.ShippingRatePricing{
			{CourierCode: "jne", CourierServiceCode: "reg", Price: 20000},
		}
		mockBiteship.On("CalculateRatesByAreaID", mock.Anything, dummyAddr.BiteshipAreaID, "jne", mock.Anything).
			Return(mockRates, nil).Once()

		discCode := "HEMAT50"
		discResult := &usecase.CheckDiscountResult{
			Code:           discCode,
			Name:           "Diskon 50rb",
			DiscountAmount: 50000.0,
			FinalAmount:    150000.0,
		}
		mockDiscountUC.On("ValidateAndCalculate", mock.Anything, discCode, mock.Anything).Return(discResult, nil).Once()

		discEntity := &entity.Discount{Code: discCode}
		discEntity.ID = 77
		mockDiscountRepo.On("FindByCode", mock.Anything, discCode).Return(discEntity, nil).Once()

		input := usecase.PreviewCheckoutInput{
			CartItemIDs:    []uint{cartItemID},
			AddressID:      &addrID,
			Courier:        "jne",
			CourierService: "reg",
			DiscountCode:   &discCode,
			PaymentMethod:  "mandiri_bill", // admin fee: 4000.0
		}

		res, err := uc.PreviewCheckout(context.Background(), userID, input)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, float64(200000), res.ItemsSubtotal)
		assert.Equal(t, float64(50000), res.DiscountAmount)
		assert.Equal(t, uint(77), *res.DiscountID)
		assert.Equal(t, float64(4000), res.AdminFee)
		assert.Equal(t, float64(174000), res.TotalPayment)
	})

	t.Run("failed when cart item IDs empty", func(t *testing.T) {
		gormDB, _ := setupGormSqlmock(t)
		uc := usecase.NewCheckoutUsecase(gormDB, nil, nil, nil, nil, nil, nil, nil)

		input := usecase.PreviewCheckoutInput{
			CartItemIDs: []uint{},
			AddressID:   &addrID,
		}

		res, err := uc.PreviewCheckout(context.Background(), userID, input)
		assert.ErrorIs(t, err, entity.ErrCartEmpty)
		assert.Nil(t, res)
	})

	t.Run("failed when address not found", func(t *testing.T) {
		gormDB, _ := setupGormSqlmock(t)
		mockAddressRepo := new(MockAddressRepositoryForCheckout)
		uc := usecase.NewCheckoutUsecase(gormDB, nil, nil, mockAddressRepo, nil, nil, nil, nil)

		dummyAddr := &entity.Address{UserID: 999}
		dummyAddr.ID = addrID
		mockAddressRepo.On("FindByID", mock.Anything, addrID).Return(dummyAddr, nil).Once()

		input := usecase.PreviewCheckoutInput{
			CartItemIDs: []uint{cartItemID},
			AddressID:   &addrID,
		}

		res, err := uc.PreviewCheckout(context.Background(), userID, input)
		assert.ErrorIs(t, err, entity.ErrAddressNotFound)
		assert.Nil(t, res)
	})

	t.Run("failed when stock insufficient in preview", func(t *testing.T) {
		gormDB, _ := setupGormSqlmock(t)
		mockCheckoutRepo := new(MockCheckoutRepository)
		mockAddressRepo := new(MockAddressRepositoryForCheckout)

		uc := usecase.NewCheckoutUsecase(gormDB, mockCheckoutRepo, nil, mockAddressRepo, nil, nil, nil, nil)

		dummyAddr := &entity.Address{UserID: userID, BiteshipAreaID: "IDNP6"}
		dummyAddr.ID = addrID
		mockAddressRepo.On("FindByID", mock.Anything, addrID).Return(dummyAddr, nil).Once()

		dummyCartItems := []entity.CartItem{
			{
				ID:       cartItemID,
				BookID:   bookID,
				Quantity: 5,
				Book: entity.Book{
					ID:          bookID,
					Stock:       2,
					IsActive:    true,
					WeightGrams: 500,
				},
			},
		}
		mockCheckoutRepo.On("GetCartItemsWithBooks", mock.Anything, userID, []uint{cartItemID}).Return(dummyCartItems, nil).Once()

		input := usecase.PreviewCheckoutInput{
			CartItemIDs: []uint{cartItemID},
			AddressID:   &addrID,
		}

		res, err := uc.PreviewCheckout(context.Background(), userID, input)
		assert.ErrorIs(t, err, entity.ErrInsufficientStock)
		assert.Nil(t, res)
	})
}

func TestCheckoutUsecase_CreateOrder(t *testing.T) {
	userID := uint(1)
	addrID := uint(5)
	cartItemID := uint(10)
	bookID := uint(30)

	t.Run("success create order within tx and executes midtrans charge", func(t *testing.T) {
		gormDB, sqlMock := setupGormSqlmock(t)
		mockCheckoutRepo := new(MockCheckoutRepository)
		mockOrderRepo := new(MockOrderRepository)
		mockAddressRepo := new(MockAddressRepositoryForCheckout)
		mockBiteship := new(MockBiteshipClientForCheckout)
		mockMidtrans := new(MockMidtransServiceForCheckout)

		uc := usecase.NewCheckoutUsecase(gormDB, mockCheckoutRepo, mockOrderRepo, mockAddressRepo, nil, nil, mockBiteship, mockMidtrans)

		dummyAddr := &entity.Address{
			UserID:          userID,
			ReceiverName:    "Budi Santoso",
			Phone:           "081234567890",
			AddressLine:     "Jl. Pemuda No. 45",
			BiteshipAreaID:  "IDNP6IDNC148IDND843IDZ12250",
			ProvinceName:    "DKI Jakarta",
			CityName:        "Jakarta Selatan",
			DistrictName:    "Kebayoran Baru",
			SubdistrictName: "Kramat Pela",
			PostalCode:      "12250",
		}
		dummyAddr.ID = addrID
		mockAddressRepo.On("FindByID", mock.Anything, addrID).Return(dummyAddr, nil).Once()

		dummyCartItems := []entity.CartItem{
			{
				ID:       cartItemID,
				BookID:   bookID,
				Quantity: 2,
				Book: entity.Book{
					ID:          bookID,
					Title:       "Buku Go",
					Price:       100000.0,
					Stock:       10,
					WeightGrams: 500,
					IsActive:    true,
				},
			},
		}
		mockCheckoutRepo.On("GetCartItemsWithBooks", mock.Anything, userID, []uint{cartItemID}).Return(dummyCartItems, nil).Once()

		mockRates := []biteship.ShippingRatePricing{
			{CourierCode: "jne", CourierServiceCode: "reg", Price: 20000},
		}
		mockBiteship.On("CalculateRatesByAreaID", mock.Anything, dummyAddr.BiteshipAreaID, "jne", mock.Anything).
			Return(mockRates, nil).Once()

		sqlMock.ExpectBegin()

		mockCheckoutRepo.On("WithTx", mock.Anything).Return(mockCheckoutRepo).Once()
		mockOrderRepo.On("WithTx", mock.Anything).Return(mockOrderRepo).Once()

		bookMap := map[uint]*entity.Book{
			bookID: {ID: bookID, Stock: 10},
		}
		mockCheckoutRepo.On("LockAndValidateBooks", mock.Anything, []uint{bookID}).Return(bookMap, nil).Once()
		mockCheckoutRepo.On("DeductBookStock", mock.Anything, bookID, 2).Return(nil).Once()
		mockOrderRepo.On("CreateOrder", mock.Anything, mock.MatchedBy(func(o *entity.Order) bool {
			return o.UserID == userID &&
				o.Status == entity.OrderStatusWaitingPayment &&
				o.TotalAmount == 225000.0
		}), mock.Anything).Return(nil).Once()
		mockCheckoutRepo.On("DeleteCartItems", mock.Anything, []uint{cartItemID}).Return(nil).Once()

		sqlMock.ExpectCommit()

		va := "123456789012"
		bank := "bca"
		mockCharge := &payment.ChargeResponse{
			TransactionID: "trx-12345",
			PaymentType:   "bank_transfer",
			Bank:          bank,
			VANumber:      va,
		}
		mockMidtrans.On("Charge", mock.Anything, mock.Anything, int64(225000), "bca_va").Return(mockCharge, nil).Once()
		mockOrderRepo.On("CreatePayment", mock.Anything, mock.MatchedBy(func(p *entity.Payment) bool {
			return p.PaymentType == "bank_transfer" && p.Status == entity.PaymentStatusPending
		})).Return(nil).Once()

		input := usecase.CreateOrderInput{
			CartItemIDs:    []uint{cartItemID},
			AddressID:      &addrID,
			Courier:        "jne",
			CourierService: "reg",
			PaymentMethod:  "bca_va",
		}

		res, err := uc.CreateOrder(context.Background(), userID, input)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, string(entity.OrderStatusWaitingPayment), res.Status)
		assert.Equal(t, float64(225000), res.TotalAmount)
		assert.Equal(t, "123456789012", *res.PaymentInfo.VANumber)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("failed when lock stock insufficient in tx", func(t *testing.T) {
		gormDB, sqlMock := setupGormSqlmock(t)
		mockCheckoutRepo := new(MockCheckoutRepository)
		mockOrderRepo := new(MockOrderRepository)
		mockAddressRepo := new(MockAddressRepositoryForCheckout)
		mockBiteship := new(MockBiteshipClientForCheckout)

		uc := usecase.NewCheckoutUsecase(gormDB, mockCheckoutRepo, mockOrderRepo, mockAddressRepo, nil, nil, mockBiteship, nil)

		dummyAddr := &entity.Address{UserID: userID, BiteshipAreaID: "IDNP6"}
		dummyAddr.ID = addrID
		mockAddressRepo.On("FindByID", mock.Anything, addrID).Return(dummyAddr, nil).Once()

		dummyCartItems := []entity.CartItem{
			{
				ID:       cartItemID,
				BookID:   bookID,
				Quantity: 2,
				Book: entity.Book{
					ID:          bookID,
					Price:       100000.0,
					Stock:       10,
					WeightGrams: 500,
					IsActive:    true,
				},
			},
		}
		mockCheckoutRepo.On("GetCartItemsWithBooks", mock.Anything, userID, []uint{cartItemID}).Return(dummyCartItems, nil).Once()

		mockRates := []biteship.ShippingRatePricing{
			{CourierCode: "jne", CourierServiceCode: "reg", Price: 20000},
		}
		mockBiteship.On("CalculateRatesByAreaID", mock.Anything, dummyAddr.BiteshipAreaID, "jne", mock.Anything).
			Return(mockRates, nil).Once()

		sqlMock.ExpectBegin()

		mockCheckoutRepo.On("WithTx", mock.Anything).Return(mockCheckoutRepo).Once()
		mockOrderRepo.On("WithTx", mock.Anything).Return(mockOrderRepo).Once()

		bookMap := map[uint]*entity.Book{
			bookID: {ID: bookID, Stock: 1},
		}
		mockCheckoutRepo.On("LockAndValidateBooks", mock.Anything, []uint{bookID}).Return(bookMap, nil).Once()

		sqlMock.ExpectRollback()

		input := usecase.CreateOrderInput{
			CartItemIDs:    []uint{cartItemID},
			AddressID:      &addrID,
			Courier:        "jne",
			CourierService: "reg",
			PaymentMethod:  "bca_va",
		}

		res, err := uc.CreateOrder(context.Background(), userID, input)
		assert.ErrorIs(t, err, entity.ErrInsufficientStock)
		assert.Nil(t, res)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

func TestCheckoutUsecase_HandleMidtransWebhook(t *testing.T) {
	orderID := "INV-20261017-000100"

	t.Run("success settlement updates order to paid", func(t *testing.T) {
		gormDB, sqlMock := setupGormSqlmock(t)
		mockOrderRepo := new(MockOrderRepository)
		mockCheckoutRepo := new(MockCheckoutRepository)
		mockMidtrans := new(MockMidtransServiceForCheckout)

		uc := usecase.NewCheckoutUsecase(gormDB, mockCheckoutRepo, mockOrderRepo, nil, nil, nil, nil, mockMidtrans)

		mockMidtrans.On("VerifyNotification", orderID, "200", "100000.00", "valid-signature").
			Return(true).Once()

		sqlMock.ExpectBegin()

		mockOrderRepo.On("WithTx", mock.Anything).Return(mockOrderRepo).Once()
		mockCheckoutRepo.On("WithTx", mock.Anything).Return(mockCheckoutRepo).Once()

		dummyOrder := &entity.Order{
			ID:            1,
			InvoiceNumber: orderID,
			Status:        entity.OrderStatusWaitingPayment,
		}
		mockOrderRepo.On("FindByInvoice", mock.Anything, orderID).Return(dummyOrder, nil).Once()
		mockOrderRepo.On("UpdateOrderStatus", mock.Anything, uint(1), entity.OrderStatusPaid).Return(nil).Once()
		mockOrderRepo.On("UpdatePaymentStatus", mock.Anything, uint(1), entity.PaymentStatusSettlement).Return(nil).Once()

		sqlMock.ExpectCommit()

		input := usecase.MidtransWebhookInput{
			OrderID:           orderID,
			StatusCode:        "200",
			GrossAmount:       "100000.00",
			SignatureKey:      "valid-signature",
			TransactionStatus: "settlement",
		}

		err := uc.HandleMidtransWebhook(context.Background(), input)
		assert.NoError(t, err)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("success expire cancels order and rolls back inventory stock", func(t *testing.T) {
		gormDB, sqlMock := setupGormSqlmock(t)
		mockOrderRepo := new(MockOrderRepository)
		mockCheckoutRepo := new(MockCheckoutRepository)
		mockMidtrans := new(MockMidtransServiceForCheckout)

		uc := usecase.NewCheckoutUsecase(gormDB, mockCheckoutRepo, mockOrderRepo, nil, nil, nil, nil, mockMidtrans)

		mockMidtrans.On("VerifyNotification", orderID, "200", "100000.00", "valid-signature").
			Return(true).Once()

		sqlMock.ExpectBegin()

		mockOrderRepo.On("WithTx", mock.Anything).Return(mockOrderRepo).Once()
		mockCheckoutRepo.On("WithTx", mock.Anything).Return(mockCheckoutRepo).Once()

		dummyOrder := &entity.Order{
			ID:            1,
			InvoiceNumber: orderID,
			Status:        entity.OrderStatusWaitingPayment,
			Items: []entity.OrderItem{
				{BookID: 30, Quantity: 2},
			},
		}
		mockOrderRepo.On("FindByInvoice", mock.Anything, orderID).Return(dummyOrder, nil).Once()
		mockCheckoutRepo.On("RestoreBookStock", mock.Anything, uint(30), 2).Return(nil).Once()
		mockOrderRepo.On("UpdateOrderStatus", mock.Anything, uint(1), entity.OrderStatusCancelled).Return(nil).Once()
		mockOrderRepo.On("UpdatePaymentStatus", mock.Anything, uint(1), entity.PaymentStatusExpire).Return(nil).Once()

		sqlMock.ExpectCommit()

		input := usecase.MidtransWebhookInput{
			OrderID:           orderID,
			StatusCode:        "200",
			GrossAmount:       "100000.00",
			SignatureKey:      "valid-signature",
			TransactionStatus: "expire",
		}

		err := uc.HandleMidtransWebhook(context.Background(), input)
		assert.NoError(t, err)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("failed invalid midtrans signature returns ErrInvalidWebhookSignature without querying database", func(t *testing.T) {
		gormDB, _ := setupGormSqlmock(t)
		mockMidtrans := new(MockMidtransServiceForCheckout)
		uc := usecase.NewCheckoutUsecase(gormDB, nil, nil, nil, nil, nil, nil, mockMidtrans)

		mockMidtrans.On("VerifyNotification", orderID, "200", "100000.00", "invalid-key").
			Return(false).Once()

		input := usecase.MidtransWebhookInput{
			OrderID:      orderID,
			StatusCode:   "200",
			GrossAmount:  "100000.00",
			SignatureKey: "invalid-key",
		}

		err := uc.HandleMidtransWebhook(context.Background(), input)
		assert.ErrorIs(t, err, entity.ErrInvalidWebhookSignature)
	})
}
