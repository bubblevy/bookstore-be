package test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/biteship"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockShipmentRepository struct {
	mock.Mock
}

func (m *MockShipmentRepository) WithTx(tx *gorm.DB) repository.ShipmentRepository {
	return m
}

func (m *MockShipmentRepository) Create(ctx context.Context, shipment *entity.Shipment) error {
	args := m.Called(ctx, shipment)
	if args.Get(0) == nil {
		shipment.ID = 1
	}
	return args.Error(0)
}

func (m *MockShipmentRepository) FindByOrderID(ctx context.Context, orderID uint) (*entity.Shipment, error) {
	args := m.Called(ctx, orderID)
	if res := args.Get(0); res != nil {
		return res.(*entity.Shipment), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockShipmentRepository) FindByBiteshipOrderID(ctx context.Context, biteshipOrderID string) (*entity.Shipment, error) {
	args := m.Called(ctx, biteshipOrderID)
	if res := args.Get(0); res != nil {
		return res.(*entity.Shipment), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockShipmentRepository) Update(ctx context.Context, shipment *entity.Shipment) error {
	return m.Called(ctx, shipment).Error(0)
}

func setupShippingUsecaseTest(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *MockOrderRepository, *MockShipmentRepository, *MockBiteshipClientForCheckout, usecase.ShippingUsecase) {
	gormDB, mock := setupGormSqlmock(t)
	mockOrderRepo := new(MockOrderRepository)
	mockShipmentRepo := new(MockShipmentRepository)
	mockBiteship := new(MockBiteshipClientForCheckout)

	cfg := usecase.StoreShippingConfig{
		ShipperName:  "Admin Bookstore",
		ShipperPhone: "081234567890",
		ShipperEmail: "admin@bookstore.com",
		OriginName:   "Gudang Utama",
		OriginPhone:  "081234567890",
		OriginAddr:   "Jl. Pemuda No. 88",
		OriginAreaID: "IDNP6IDNC148IDND843IDZ12250",
	}

	uc := usecase.NewShippingUsecase(gormDB, mockOrderRepo, mockShipmentRepo, mockBiteship, cfg)
	return gormDB, mock, mockOrderRepo, mockShipmentRepo, mockBiteship, uc
}

func TestShippingUsecase_FulfillOrder(t *testing.T) {
	ctx := context.Background()
	orderID := uint(100)

	t.Run("success fulfill paid order and create shipment", func(t *testing.T) {
		_, sqlMock, _, mockShipmentRepo, mockBiteship, uc := setupShippingUsecaseTest(t)

		// 1. Query Order (Tanpa WithArgs yang kaku agar cocok dengan binding LIMIT First GORM)
		sqlMock.ExpectQuery(`SELECT (.+) FROM "orders" WHERE id = \$1.*`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "user_id", "invoice_number", "status", "courier", "courier_service",
				"recipient_name", "recipient_phone", "recipient_address", "postal_code",
			}).AddRow(
				orderID, 1, "INV-20261016-000100", entity.OrderStatusPaid, "jne", "reg",
				"Budi Santoso", "081234567890", "Jl. Mawar No. 1", "IDNP6IDNC148IDND843IDZ12250",
			))

		// 2. Query Order Items (Preload)
		sqlMock.ExpectQuery(`SELECT (.+) FROM "order_items" WHERE "order_items"."order_id" = \$1`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "order_id", "book_id", "title", "price", "weight_grams", "quantity"}).
				AddRow(1, orderID, 10, "Buku Golang", 100000.0, 500, 2))

		// 3. Cek apakah sudah pernah dibuat shipment
		mockShipmentRepo.On("FindByOrderID", ctx, orderID).Return(nil, nil).Once()

		// 4. Mock Biteship CreateOrder
		waybill := "WAYBILL123456"
		trackingID := "TRACK123456"
		mockOrderRes := &biteship.OrderResponse{
			Success: true,
			ID:      "biteship-order-789",
		}
		mockOrderRes.Courier.WaybillID = waybill
		mockOrderRes.Courier.TrackingID = trackingID
		mockOrderRes.Courier.TrackingURL = "https://track.biteship.com/WAYBILL123456"

		mockBiteship.On("CreateOrderByAreaID", ctx, mock.MatchedBy(func(req biteship.CreateOrderByAreaIDRequest) bool {
			return req.CourierCompany == "jne" && req.CourierType == "reg" && len(req.Items) == 1
		})).Return(mockOrderRes, nil).Once()

		// 5. DB Transaction
		sqlMock.ExpectBegin()
		sqlMock.ExpectQuery(`INSERT INTO "shipments"`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		sqlMock.ExpectExec(`UPDATE "orders" SET .* WHERE .*"?id"? = \$\d+`).
			WillReturnResult(sqlmock.NewResult(1, 1))
		sqlMock.ExpectCommit()

		result, err := uc.FulfillOrder(ctx, orderID, "Fragile item")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, orderID, result.OrderID)
		assert.Equal(t, "biteship-order-789", result.BiteshipOrderID)
		assert.Equal(t, &waybill, result.WaybillNumber)
		assert.Equal(t, entity.ShippingStatusAllocated, result.Status)

		assert.NoError(t, sqlMock.ExpectationsWereMet())
		mockShipmentRepo.AssertExpectations(t)
		mockBiteship.AssertExpectations(t)
	})

	t.Run("failed when order not found", func(t *testing.T) {
		_, sqlMock, _, _, _, uc := setupShippingUsecaseTest(t)

		sqlMock.ExpectQuery(`SELECT (.+) FROM "orders" WHERE id = \$1.*`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))

		result, err := uc.FulfillOrder(ctx, orderID, "")
		assert.ErrorIs(t, err, entity.ErrOrderNotFound)
		assert.Nil(t, result)
	})

	t.Run("failed when order not paid", func(t *testing.T) {
		_, sqlMock, _, _, _, uc := setupShippingUsecaseTest(t)

		sqlMock.ExpectQuery(`SELECT (.+) FROM "orders" WHERE id = \$1.*`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).
				AddRow(orderID, entity.OrderStatusWaitingPayment))

		sqlMock.ExpectQuery(`SELECT (.+) FROM "order_items" WHERE "order_items"."order_id" = \$1`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))

		result, err := uc.FulfillOrder(ctx, orderID, "")
		assert.ErrorIs(t, err, entity.ErrOrderNotPaid)
		assert.Nil(t, result)
	})

	t.Run("failed when shipment already exists", func(t *testing.T) {
		_, sqlMock, _, mockShipmentRepo, _, uc := setupShippingUsecaseTest(t)

		sqlMock.ExpectQuery(`SELECT (.+) FROM "orders" WHERE id = \$1.*`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).
				AddRow(orderID, entity.OrderStatusPaid))

		sqlMock.ExpectQuery(`SELECT (.+) FROM "order_items" WHERE "order_items"."order_id" = \$1`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))

		mockShipmentRepo.On("FindByOrderID", ctx, orderID).
			Return(&entity.Shipment{ID: 1, OrderID: orderID}, nil).Once()

		result, err := uc.FulfillOrder(ctx, orderID, "")
		assert.ErrorIs(t, err, entity.ErrOrderAlreadyFulfilled)
		assert.Nil(t, result)
	})
}

func TestShippingUsecase_GetOrderTracking(t *testing.T) {
	ctx := context.Background()
	orderID := uint(100)
	userID := uint(1)

	t.Run("success track order by waybill", func(t *testing.T) {
		_, sqlMock, _, mockShipmentRepo, mockBiteship, uc := setupShippingUsecaseTest(t)

		sqlMock.ExpectQuery(`SELECT (.+) FROM "orders" WHERE id = \$1.*`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).AddRow(orderID, userID))

		waybill := "WAYBILL123"
		shipment := &entity.Shipment{
			OrderID:       orderID,
			Courier:       "jne",
			WaybillNumber: &waybill,
		}
		mockShipmentRepo.On("FindByOrderID", ctx, orderID).Return(shipment, nil).Once()

		expectedTracking := &biteship.TrackingDetailResult{
			Success:   true,
			WaybillID: waybill,
			Status:    "in_transit",
		}
		mockBiteship.On("TrackWaybill", ctx, waybill, "jne").Return(expectedTracking, nil).Once()

		res, err := uc.GetOrderTracking(ctx, userID, orderID)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, waybill, res.WaybillID)
		assert.Equal(t, "in_transit", res.Status)
	})

	t.Run("failed when unauthorized user", func(t *testing.T) {
		_, sqlMock, _, _, _, uc := setupShippingUsecaseTest(t)

		sqlMock.ExpectQuery(`SELECT (.+) FROM "orders" WHERE id = \$1.*`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).AddRow(orderID, uint(999)))

		res, err := uc.GetOrderTracking(ctx, userID, orderID)
		assert.ErrorIs(t, err, entity.ErrUnauthorizedTracking)
		assert.Nil(t, res)
	})
}

func TestShippingUsecase_HandleWebhook(t *testing.T) {
	ctx := context.Background()

	t.Run("success webhook delivered updates shipment and order completed", func(t *testing.T) {
		_, sqlMock, _, mockShipmentRepo, _, uc := setupShippingUsecaseTest(t)

		shipment := &entity.Shipment{
			ID:              1,
			OrderID:         100,
			BiteshipOrderID: "biteship-123",
		}
		mockShipmentRepo.On("FindByBiteshipOrderID", ctx, "biteship-123").Return(shipment, nil).Once()

		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(`UPDATE "shipments" SET .* WHERE .*"?id"? = \$\d+`).
			WillReturnResult(sqlmock.NewResult(1, 1))
		sqlMock.ExpectExec(`UPDATE "orders" SET .* WHERE .*"?id"? = \$\d+`).
			WillReturnResult(sqlmock.NewResult(1, 1))
		sqlMock.ExpectCommit()

		payload := biteship.WebhookTrackingPayload{
			OrderID:   "biteship-123",
			Status:    "delivered",
			WaybillID: "WAYBILL123",
			UpdatedAt: time.Now(),
		}

		err := uc.HandleWebhook(ctx, payload)
		assert.NoError(t, err)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}
