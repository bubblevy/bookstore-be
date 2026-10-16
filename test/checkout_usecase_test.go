package test

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/biteship"
	"github.com/bubblevy/bookstore/pkg/payment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

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
	args := m.Called(ctx, order, items)
	return args.Error(0)
}

func (m *MockOrderRepository) CreatePayment(ctx context.Context, p *entity.Payment) error {
	args := m.Called(ctx, p)
	return args.Error(0)
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

func (m *MockOrderRepository) UpdateOrderStatus(ctx context.Context, orderID uint, status entity.OrderStatus) error {
	args := m.Called(ctx, orderID, status)
	return args.Error(0)
}

func (m *MockOrderRepository) UpdatePaymentStatus(ctx context.Context, orderID uint, status entity.PaymentStatus) error {
	args := m.Called(ctx, orderID, status)
	return args.Error(0)
}

func (m *MockOrderRepository) MarkAsReturned(ctx context.Context, orderID uint, reason string) error {
	args := m.Called(ctx, orderID, reason)
	return args.Error(0)
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

type AnyTime struct{}

func (a AnyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

func TestCheckoutUsecase_PreviewCheckout(t *testing.T) {
	userID := uint(1)
	addrID := uint(5)
	cartItemID := uint(10)
	cartID := uint(20)
	bookID := uint(30)

	t.Run("success preview with explicit address_id", func(t *testing.T) {
		gormDB, sqlMock := setupGormSqlmock(t)
		mockOrderRepo := new(MockOrderRepository)
		mockAddressRepo := new(MockAddressRepositoryForCheckout)
		mockBiteship := new(MockBiteshipClientForCheckout)
		mockMidtrans := new(MockMidtransServiceForCheckout)

		uc := usecase.NewCheckoutUsecase(gormDB, mockOrderRepo, nil, mockAddressRepo, nil, mockBiteship, mockMidtrans)

		dummyAddr := &entity.Address{
			UserID:         userID,
			BiteshipAreaID: "IDNP6IDNC148IDND843IDZ12250",
			CityName:       "Jakarta Selatan",
		}
		dummyAddr.ID = addrID
		mockAddressRepo.On("FindByID", mock.Anything, addrID).Return(dummyAddr, nil).Once()

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "carts" WHERE user_id = $1 ORDER BY "carts"."id" LIMIT $2`)).
			WithArgs(userID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).AddRow(cartID, userID))

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "cart_items" WHERE cart_id = $1 AND id IN ($2)`)).
			WithArgs(cartID, cartItemID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "cart_id", "book_id", "quantity"}).
				AddRow(cartItemID, cartID, bookID, 2))

		cover := "cover.jpg"
		sqlMock.ExpectQuery(`SELECT (.+) FROM "books" WHERE id IN \(\$1\)`).
			WithArgs(bookID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "price", "stock", "weight_grams", "cover_image", "is_active"}).
				AddRow(bookID, "Buku Go", 100000.0, 10, 500, cover, true))

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
			PaymentMethod:  "bca_va",
		}

		res, err := uc.PreviewCheckout(context.Background(), userID, input)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, float64(200000), res.ItemsSubtotal)
		assert.Equal(t, float64(20000), res.ShippingCost)
		assert.Equal(t, float64(4000), res.AdminFee)
		assert.Equal(t, float64(224000), res.TotalPayment)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("success preview fallback to primary address when address_id is nil", func(t *testing.T) {
		gormDB, sqlMock := setupGormSqlmock(t)
		mockOrderRepo := new(MockOrderRepository)
		mockAddressRepo := new(MockAddressRepositoryForCheckout)
		mockBiteship := new(MockBiteshipClientForCheckout)
		mockMidtrans := new(MockMidtransServiceForCheckout)

		uc := usecase.NewCheckoutUsecase(gormDB, mockOrderRepo, nil, mockAddressRepo, nil, mockBiteship, mockMidtrans)

		dummyPrimaryAddr := &entity.Address{
			UserID:         userID,
			BiteshipAreaID: "IDNP6IDNC148IDND843IDZ12250",
			CityName:       "Jakarta Selatan",
			IsPrimary:      true,
		}
		dummyPrimaryAddr.ID = 9
		mockAddressRepo.On("FindPrimaryByUserID", mock.Anything, userID).Return(dummyPrimaryAddr, nil).Once()

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "carts" WHERE user_id = $1 ORDER BY "carts"."id" LIMIT $2`)).
			WithArgs(userID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).AddRow(cartID, userID))

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "cart_items" WHERE cart_id = $1 AND id IN ($2)`)).
			WithArgs(cartID, cartItemID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "cart_id", "book_id", "quantity"}).
				AddRow(cartItemID, cartID, bookID, 2))

		cover := "cover.jpg"
		sqlMock.ExpectQuery(`SELECT (.+) FROM "books" WHERE id IN \(\$1\)`).
			WithArgs(bookID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "price", "stock", "weight_grams", "cover_image", "is_active"}).
				AddRow(bookID, "Buku Go", 100000.0, 10, 500, cover, true))

		mockRates := []biteship.ShippingRatePricing{
			{CourierCode: "jne", CourierServiceCode: "reg", Price: 20000},
		}
		mockBiteship.On("CalculateRatesByAreaID", mock.Anything, dummyPrimaryAddr.BiteshipAreaID, "jne", mock.Anything).
			Return(mockRates, nil).Once()

		input := usecase.PreviewCheckoutInput{
			CartItemIDs:    []uint{cartItemID},
			AddressID:      nil,
			Courier:        "jne",
			CourierService: "reg",
			PaymentMethod:  "bca_va",
		}

		res, err := uc.PreviewCheckout(context.Background(), userID, input)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, float64(224000), res.TotalPayment)
	})

	t.Run("failed when cart items empty", func(t *testing.T) {
		gormDB, _ := setupGormSqlmock(t)
		uc := usecase.NewCheckoutUsecase(gormDB, nil, nil, nil, nil, nil, nil)

		input := usecase.PreviewCheckoutInput{
			CartItemIDs: []uint{},
			AddressID:   &addrID,
		}
		res, err := uc.PreviewCheckout(context.Background(), userID, input)
		assert.ErrorIs(t, err, entity.ErrCartEmpty)
		assert.Nil(t, res)
	})

	t.Run("failed when address unauthorized", func(t *testing.T) {
		gormDB, _ := setupGormSqlmock(t)
		mockAddressRepo := new(MockAddressRepositoryForCheckout)
		uc := usecase.NewCheckoutUsecase(gormDB, nil, nil, mockAddressRepo, nil, nil, nil)

		dummyAddr := &entity.Address{UserID: 999}
		dummyAddr.ID = addrID
		mockAddressRepo.On("FindByID", mock.Anything, addrID).Return(dummyAddr, nil).Once()

		input := usecase.PreviewCheckoutInput{
			CartItemIDs: []uint{cartItemID},
			AddressID:   &addrID,
		}
		res, err := uc.PreviewCheckout(context.Background(), userID, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "shipping address not found")
		assert.Nil(t, res)
	})
}

func TestCheckoutUsecase_CreateOrder(t *testing.T) {
	userID := uint(1)
	addrID := uint(5)
	cartItemID := uint(10)
	cartID := uint(20)
	bookID := uint(30)

	t.Run("success create order deducts stock and executes midtrans charge", func(t *testing.T) {
		gormDB, sqlMock := setupGormSqlmock(t)
		mockOrderRepo := new(MockOrderRepository)
		mockAddressRepo := new(MockAddressRepositoryForCheckout)
		mockBiteship := new(MockBiteshipClientForCheckout)
		mockMidtrans := new(MockMidtransServiceForCheckout)

		uc := usecase.NewCheckoutUsecase(gormDB, mockOrderRepo, nil, mockAddressRepo, nil, mockBiteship, mockMidtrans)

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

		sqlMock.ExpectBegin()

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "carts" WHERE user_id = $1 ORDER BY "carts"."id" LIMIT $2`)).
			WithArgs(userID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).AddRow(cartID, userID))

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "cart_items" WHERE cart_id = $1 AND id IN ($2)`)).
			WithArgs(cartID, cartItemID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "cart_id", "book_id", "quantity"}).
				AddRow(cartItemID, cartID, bookID, 2))

		cover := "cover.jpg"
		sqlMock.ExpectQuery(`SELECT (.+) FROM "books" WHERE id IN \(\$1\).*FOR UPDATE`).
			WithArgs(bookID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "price", "stock", "weight_grams", "cover_image", "is_active"}).
				AddRow(bookID, "Buku Go", 100000.0, 10, 500, cover, true))

		mockRates := []biteship.ShippingRatePricing{
			{CourierCode: "jne", CourierServiceCode: "reg", Price: 20000},
		}
		mockBiteship.On("CalculateRatesByAreaID", mock.Anything, dummyAddr.BiteshipAreaID, "jne", mock.Anything).
			Return(mockRates, nil).Once()

		sqlMock.ExpectExec(`UPDATE "books" SET "stock"=\$1,"updated_at"=\$2 WHERE .*"?id"? = \$3`).
			WithArgs(8, AnyTime{}, bookID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mockOrderRepo.On("WithTx", mock.Anything).Return(mockOrderRepo).Once()
		mockOrderRepo.On("CreateOrder", mock.Anything, mock.MatchedBy(func(o *entity.Order) bool {
			return o.UserID == userID &&
				o.Status == entity.OrderStatusWaitingPayment &&
				o.DestinationAreaID == dummyAddr.BiteshipAreaID &&
				o.PostalCode == dummyAddr.PostalCode &&
				o.TotalAmount == 224000
		}), mock.Anything).Return(nil).Once()

		sqlMock.ExpectExec(`(DELETE FROM "cart_items"|UPDATE "cart_items" SET "deleted_at")`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		sqlMock.ExpectCommit()

		mockCharge := &payment.ChargeResponse{
			TransactionID: "trx-12345",
			PaymentType:   "bank_transfer",
			Bank:          "bca",
			VANumber:      "123456789012",
		}
		mockMidtrans.On("Charge", mock.Anything, mock.Anything, int64(224000), "bca_va").Return(mockCharge, nil).Once()
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
		assert.Equal(t, "waiting_payment", res.Status)
		assert.Equal(t, float64(224000), res.TotalAmount)
		assert.Equal(t, "123456789012", *res.PaymentInfo.VANumber)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("failed when stock insufficient rolls back transaction", func(t *testing.T) {
		gormDB, sqlMock := setupGormSqlmock(t)
		mockAddressRepo := new(MockAddressRepositoryForCheckout)
		uc := usecase.NewCheckoutUsecase(gormDB, nil, nil, mockAddressRepo, nil, nil, nil)

		dummyAddr := &entity.Address{UserID: userID, BiteshipAreaID: "IDNP6IDNC148IDND843IDZ12250"}
		dummyAddr.ID = addrID
		mockAddressRepo.On("FindByID", mock.Anything, addrID).Return(dummyAddr, nil).Once()

		sqlMock.ExpectBegin()

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "carts" WHERE user_id = $1 ORDER BY "carts"."id" LIMIT $2`)).
			WithArgs(userID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).AddRow(cartID, userID))

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "cart_items" WHERE cart_id = $1 AND id IN ($2)`)).
			WithArgs(cartID, cartItemID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "cart_id", "book_id", "quantity"}).
				AddRow(cartItemID, cartID, bookID, 5))

		sqlMock.ExpectQuery(`SELECT (.+) FROM "books" WHERE id IN \(\$1\).*FOR UPDATE`).
			WithArgs(bookID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "price", "stock", "weight_grams", "is_active"}).
				AddRow(bookID, "Buku Go", 100000.0, 2, 500, true))

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
	orderID := "inv-20261016-000100"

	t.Run("success settlement updates order and payment to paid", func(t *testing.T) {
		gormDB, sqlMock := setupGormSqlmock(t)
		mockMidtrans := new(MockMidtransServiceForCheckout)
		uc := usecase.NewCheckoutUsecase(gormDB, nil, nil, nil, nil, nil, mockMidtrans)

		mockMidtrans.On("VerifyNotification", orderID, "200", "100000.00", "valid-signature").
			Return(true).Once()

		sqlMock.ExpectBegin()

		sqlMock.ExpectQuery(`SELECT (.+) FROM "orders" WHERE invoice_number = \$1.*LIMIT \$2`).
			WithArgs(orderID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "invoice_number", "status"}).
				AddRow(1, orderID, entity.OrderStatusWaitingPayment))

		sqlMock.ExpectQuery(`SELECT (.+) FROM "order_items" WHERE "order_items"."order_id" = \$1`).
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "order_id", "book_id", "quantity"}).
				AddRow(10, 1, 30, 2))

		sqlMock.ExpectExec(`UPDATE "orders" SET .* WHERE .*"?id"? = \$\d+`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		sqlMock.ExpectExec(`UPDATE "payments" SET`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		sqlMock.ExpectCommit()

		input := usecase.MidtransWebhookInput{
			OrderID:           orderID,
			StatusCode:        "200",
			GrossAmount:       "100000.00",
			SignatureKey:      "valid-signature",
			TransactionStatus: "settlement",
			RawJSON:           []byte(`{"status":"settlement"}`),
		}

		err := uc.HandleMidtransWebhook(context.Background(), input)
		assert.NoError(t, err)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("success expire cancels order and rolls back book stock", func(t *testing.T) {
		gormDB, sqlMock := setupGormSqlmock(t)
		mockMidtrans := new(MockMidtransServiceForCheckout)
		uc := usecase.NewCheckoutUsecase(gormDB, nil, nil, nil, nil, nil, mockMidtrans)

		mockMidtrans.On("VerifyNotification", orderID, "200", "100000.00", "valid-signature").
			Return(true).Once()

		sqlMock.ExpectBegin()

		sqlMock.ExpectQuery(`SELECT (.+) FROM "orders" WHERE invoice_number = \$1.*LIMIT \$2`).
			WithArgs(orderID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "invoice_number", "status"}).
				AddRow(1, orderID, entity.OrderStatusWaitingPayment))

		sqlMock.ExpectQuery(`SELECT (.+) FROM "order_items" WHERE "order_items"."order_id" = \$1`).
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "order_id", "book_id", "quantity"}).
				AddRow(10, 1, 30, 2))

		sqlMock.ExpectExec(`UPDATE "books" SET "stock"=stock \+ \$1,"updated_at"=\$2 WHERE id = \$3.*`).
			WithArgs(2, AnyTime{}, 30).
			WillReturnResult(sqlmock.NewResult(1, 1))

		sqlMock.ExpectExec(`UPDATE "orders" SET .* WHERE .*"?id"? = \$\d+`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		sqlMock.ExpectExec(`UPDATE "payments" SET`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		sqlMock.ExpectCommit()

		input := usecase.MidtransWebhookInput{
			OrderID:           orderID,
			StatusCode:        "200",
			GrossAmount:       "100000.00",
			SignatureKey:      "valid-signature",
			TransactionStatus: "expire",
			RawJSON:           []byte(`{"status":"expire"}`),
		}

		err := uc.HandleMidtransWebhook(context.Background(), input)
		assert.NoError(t, err)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("failed invalid signature returns error without db query", func(t *testing.T) {
		gormDB, _ := setupGormSqlmock(t)
		mockMidtrans := new(MockMidtransServiceForCheckout)
		uc := usecase.NewCheckoutUsecase(gormDB, nil, nil, nil, nil, nil, mockMidtrans)

		mockMidtrans.On("VerifyNotification", orderID, "200", "100000.00", "invalid-key").
			Return(false).Once()

		input := usecase.MidtransWebhookInput{
			OrderID:      orderID,
			StatusCode:   "200",
			GrossAmount:  "100000.00",
			SignatureKey: "invalid-key",
		}

		err := uc.HandleMidtransWebhook(context.Background(), input)
		assert.Error(t, err)
		assert.Equal(t, "invalid midtrans webhook signature", err.Error())
	})
}
