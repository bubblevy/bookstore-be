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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type AnyTime struct{}

func (a AnyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

type MockShipmentRepository struct {
	mock.Mock
}

func (m *MockShipmentRepository) WithTx(tx *gorm.DB) repository.ShipmentRepository {
	args := m.Called(tx)
	if res := args.Get(0); res != nil {
		return res.(repository.ShipmentRepository)
	}
	return m
}

func (m *MockShipmentRepository) Create(ctx context.Context, shipment *entity.Shipment) error {
	return m.Called(ctx, shipment).Error(0)
}

func (m *MockShipmentRepository) Update(ctx context.Context, shipment *entity.Shipment) error {
	return m.Called(ctx, shipment).Error(0)
}

func (m *MockShipmentRepository) FindByID(ctx context.Context, id uint) (*entity.Shipment, error) {
	args := m.Called(ctx, id)
	if res := args.Get(0); res != nil {
		return res.(*entity.Shipment), args.Error(1)
	}
	return nil, args.Error(1)
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

func (m *MockShipmentRepository) FindByIdentifier(ctx context.Context, identifier string) (*entity.Shipment, error) {
	args := m.Called(ctx, identifier)
	if res := args.Get(0); res != nil {
		return res.(*entity.Shipment), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockShipmentRepository) FindByInvoiceNumber(ctx context.Context, invoiceNumber string) (*entity.Shipment, error) {
	args := m.Called(ctx, invoiceNumber)
	if res := args.Get(0); res != nil {
		return res.(*entity.Shipment), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockShipmentRepository) UpdateStatus(ctx context.Context, shipmentID uint, status entity.ShippingStatus, waybillNumber *string) error {
	return m.Called(ctx, shipmentID, status, waybillNumber).Error(0)
}

func defaultStoreShippingConfig() usecase.StoreShippingConfig {
	return usecase.StoreShippingConfig{
		ShipperName:       "Toko Buku Bubble",
		ShipperPhone:      "081299998888",
		ShipperEmail:      "shipper@bookstore.com",
		OriginName:        "Gudang Utama",
		OriginPhone:       "081299998888",
		OriginAddr:        "Jl. Logistik No. 10",
		OriginAreaID:      "IDNP6IDNC148",
		OriginPostalCode:  12250,
		AvailableCouriers: "jne,sicepat,jnt",
	}
}

func TestShippingUsecase_GetShippingRates(t *testing.T) {
	userID := uint(1)
	addrID := uint(5)
	cartItemID := uint(10)
	bookID := uint(30)

	t.Run("success calculate rates returns formatted full destination address", func(t *testing.T) {
		gormDB, _ := setupGormSqlmock(t)
		mockCheckoutRepo := new(MockCheckoutRepository)
		mockAddressRepo := new(MockAddressRepositoryForCheckout)
		mockBiteship := new(MockBiteshipClientForCheckout)

		uc := usecase.NewShippingUsecase(gormDB, nil, nil, mockCheckoutRepo, mockAddressRepo, mockBiteship, defaultStoreShippingConfig())

		dummyAddr := &entity.Address{
			UserID:          userID,
			ReceiverName:    "Budi Santoso",
			Phone:           "081234567890",
			AddressLine:     "Jl. Pemuda No. 45",
			SubdistrictName: "Kramat Pela",
			DistrictName:    "Kebayoran Baru",
			CityName:        "Jakarta Selatan",
			ProvinceName:    "DKI Jakarta",
			PostalCode:      "12250",
			BiteshipAreaID:  "IDNP6IDNC148IDND843IDZ12250",
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
			{
				CourierCode:        "jne",
				CourierName:        "JNE",
				CourierServiceName: "Reguler",
				CourierServiceCode: "reg",
				Price:              20000,
				Duration:           "1-2 days",
			},
		}
		mockBiteship.On("CalculateRatesByAreaID", mock.Anything, dummyAddr.BiteshipAreaID, "jne,sicepat,jnt", mock.Anything).
			Return(mockRates, nil).Once()

		input := usecase.GetShippingRatesInput{
			CartItemIDs: []uint{cartItemID},
			AddressID:   &addrID,
		}

		res, err := uc.GetShippingRates(context.Background(), userID, input)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, 1000, res.TotalWeightGrams)
		assert.Equal(t, "Jl. Pemuda No. 45, Kel. Kramat Pela, Kec. Kebayoran Baru, Jakarta Selatan, DKI Jakarta 12250", res.DestinationArea)
		assert.Len(t, res.Options, 1)
		assert.Equal(t, 20000, res.Options[0].Price)
	})

	t.Run("empty cart item returns ErrCartEmpty", func(t *testing.T) {
		gormDB, _ := setupGormSqlmock(t)
		uc := usecase.NewShippingUsecase(gormDB, nil, nil, nil, nil, nil, defaultStoreShippingConfig())

		input := usecase.GetShippingRatesInput{CartItemIDs: []uint{}}
		res, err := uc.GetShippingRates(context.Background(), userID, input)

		assert.ErrorIs(t, err, entity.ErrCartEmpty)
		assert.Nil(t, res)
	})
}

func TestShippingUsecase_FulfillOrder(t *testing.T) {
	invoice := "INV-20261017-000101"

	t.Run("success fulfill creates shipment and updates order status to processing", func(t *testing.T) {
		gormDB, sqlMock := setupGormSqlmock(t)
		mockShipmentRepo := new(MockShipmentRepository)
		mockBiteship := new(MockBiteshipClientForCheckout)

		uc := usecase.NewShippingUsecase(gormDB, nil, mockShipmentRepo, nil, nil, mockBiteship, defaultStoreShippingConfig())

		// 1. Preload Order & Items
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "orders" WHERE invoice_number = $1 ORDER BY "orders"."id" LIMIT $2`)).
			WithArgs(invoice, 1).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "user_id", "invoice_number", "status", "recipient_name", "recipient_phone",
				"recipient_address", "destination_area_id", "postal_code", "courier", "courier_service",
			}).AddRow(1, 10, invoice, entity.OrderStatusPaid, "Budi", "081234", "Jl. Mawar", "AREA-123", "12250", "jne", "reg"))

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "order_items" WHERE "order_items"."order_id" = $1`)).
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "order_id", "title", "price", "weight_grams", "quantity"}).
				AddRow(1, 1, "Buku Go", 100000.0, 500, 2))

		// 2. Check existing shipment
		mockShipmentRepo.On("FindByOrderID", mock.Anything, uint(1)).Return(nil, nil).Once()

		// 3. Biteship API Call
		waybillID := "SOCAG0001"
		biteshipRes := &biteship.OrderResponse{
			ID: "bite-order-999",
		}
		biteshipRes.Courier.WaybillID = waybillID

		mockBiteship.On("CreateOrderByAreaID", mock.Anything, mock.MatchedBy(func(req biteship.CreateOrderByAreaIDRequest) bool {
			return req.DestinationAreaID == "AREA-123" && req.CourierCompany == "jne"
		})).Return(biteshipRes, nil).Once()

		// 4. DB Transaction: Insert Shipment (RETURNING "id") & Update Order
		sqlMock.ExpectBegin()
		sqlMock.ExpectQuery(`INSERT INTO "shipments"`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		sqlMock.ExpectExec(`UPDATE "orders" SET`).
			WillReturnResult(sqlmock.NewResult(1, 1))
		sqlMock.ExpectCommit()

		res, err := uc.FulfillOrder(context.Background(), invoice, "Handle with care")
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "bite-order-999", res.BiteshipOrderID)
		assert.Equal(t, "SOCAG0001", *res.WaybillNumber)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("order not paid returns ErrOrderNotPaid", func(t *testing.T) {
		gormDB, sqlMock := setupGormSqlmock(t)
		uc := usecase.NewShippingUsecase(gormDB, nil, nil, nil, nil, nil, defaultStoreShippingConfig())

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "orders" WHERE invoice_number = $1 ORDER BY "orders"."id" LIMIT $2`)).
			WithArgs(invoice, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "invoice_number", "status"}).
				AddRow(1, invoice, entity.OrderStatusWaitingPayment))

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "order_items" WHERE "order_items"."order_id" = $1`)).
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "order_id"}))

		res, err := uc.FulfillOrder(context.Background(), invoice, "")
		assert.ErrorIs(t, err, entity.ErrOrderNotPaid)
		assert.Nil(t, res)
	})
}

func TestShippingUsecase_GetOrderTracking(t *testing.T) {
	invoice := "INV-20261017-000101"
	userID := uint(10)

	t.Run("success tracking with waybill number", func(t *testing.T) {
		gormDB, sqlMock := setupGormSqlmock(t)
		mockShipmentRepo := new(MockShipmentRepository)
		mockBiteship := new(MockBiteshipClientForCheckout)

		uc := usecase.NewShippingUsecase(gormDB, nil, mockShipmentRepo, nil, nil, mockBiteship, defaultStoreShippingConfig())

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "orders" WHERE invoice_number = $1 ORDER BY "orders"."id" LIMIT $2`)).
			WithArgs(invoice, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "invoice_number"}).
				AddRow(1, userID, invoice))

		waybill := "SOCAG0001"
		shipment := &entity.Shipment{
			ID:            1,
			OrderID:       1,
			WaybillNumber: &waybill,
			Courier:       "jne",
		}
		mockShipmentRepo.On("FindByOrderID", mock.Anything, uint(1)).Return(shipment, nil).Once()

		trackingRes := &biteship.TrackingDetailResult{
			WaybillID:   waybill,
			CourierCode: "jne",
			Status:      "allocated",
		}
		mockBiteship.On("TrackWaybill", mock.Anything, waybill, "jne").Return(trackingRes, nil).Once()

		res, err := uc.GetOrderTracking(context.Background(), userID, invoice)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, waybill, res.WaybillID)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("unauthorized user tracking returns ErrUnauthorizedTracking", func(t *testing.T) {
		gormDB, sqlMock := setupGormSqlmock(t)
		uc := usecase.NewShippingUsecase(gormDB, nil, nil, nil, nil, nil, defaultStoreShippingConfig())

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "orders" WHERE invoice_number = $1 ORDER BY "orders"."id" LIMIT $2`)).
			WithArgs(invoice, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "invoice_number"}).
				AddRow(1, uint(999), invoice))

		res, err := uc.GetOrderTracking(context.Background(), userID, invoice)
		assert.ErrorIs(t, err, entity.ErrUnauthorizedTracking)
		assert.Nil(t, res)
	})
}

func TestShippingUsecase_ProcessReturn(t *testing.T) {
	invoice := "INV-20261017-000101"

	t.Run("success process return restocks books and sets order to returned", func(t *testing.T) {
		gormDB, sqlMock := setupGormSqlmock(t)
		uc := usecase.NewShippingUsecase(gormDB, nil, nil, nil, nil, nil, defaultStoreShippingConfig())

		sqlMock.ExpectBegin()

		// 1. Query order with preload items
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "orders" WHERE invoice_number = $1 ORDER BY "orders"."id" LIMIT $2`)).
			WithArgs(invoice, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "invoice_number", "status"}).
				AddRow(1, invoice, entity.OrderStatusShipped))

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "order_items" WHERE "order_items"."order_id" = $1`)).
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "order_id", "book_id", "quantity"}).
				AddRow(1, 1, 30, 2))

		// 2. Update Order
		sqlMock.ExpectExec(`UPDATE "orders" SET`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// 3. Update Shipment
		sqlMock.ExpectExec(`UPDATE "shipments" SET`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// 4. Restock Book (mencakup auto-update "updated_at" dan soft-delete check GORM)
		sqlMock.ExpectExec(`UPDATE "books" SET "stock"=stock \+ \$1.*WHERE id = \$3`).
			WithArgs(2, AnyTime{}, 30).
			WillReturnResult(sqlmock.NewResult(1, 1))

		sqlMock.ExpectCommit()

		err := uc.ProcessReturn(context.Background(), invoice, "Damaged in transit")
		assert.NoError(t, err)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("invalid order state returns ErrOrderInvalidReturnState", func(t *testing.T) {
		gormDB, sqlMock := setupGormSqlmock(t)
		uc := usecase.NewShippingUsecase(gormDB, nil, nil, nil, nil, nil, defaultStoreShippingConfig())

		sqlMock.ExpectBegin()
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "orders" WHERE invoice_number = $1 ORDER BY "orders"."id" LIMIT $2`)).
			WithArgs(invoice, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "invoice_number", "status"}).
				AddRow(1, invoice, entity.OrderStatusWaitingPayment))

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "order_items" WHERE "order_items"."order_id" = $1`)).
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "order_id"}))

		sqlMock.ExpectRollback()

		err := uc.ProcessReturn(context.Background(), invoice, "")
		assert.ErrorIs(t, err, entity.ErrOrderInvalidReturnState)
	})
}

func TestShippingUsecase_HandleWebhook(t *testing.T) {
	t.Run("webhook delivered updates order to completed and shipment to delivered", func(t *testing.T) {
		gormDB, sqlMock := setupGormSqlmock(t)
		mockShipmentRepo := new(MockShipmentRepository)

		uc := usecase.NewShippingUsecase(gormDB, nil, mockShipmentRepo, nil, nil, nil, defaultStoreShippingConfig())

		shipment := &entity.Shipment{
			ID:              1,
			OrderID:         10,
			BiteshipOrderID: "bite-999",
			Status:          entity.ShippingStatusInTransit,
		}
		mockShipmentRepo.On("FindByBiteshipOrderID", mock.Anything, "bite-999").Return(shipment, nil).Once()

		sqlMock.ExpectBegin()

		// 1. Update Shipment
		sqlMock.ExpectExec(`UPDATE "shipments" SET`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// 2. Preload order inside transaction
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "orders" WHERE id = $1 ORDER BY "orders"."id" LIMIT $2`)).
			WithArgs(10, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).
				AddRow(10, entity.OrderStatusShipped))

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "order_items" WHERE "order_items"."order_id" = $1`)).
			WithArgs(10).
			WillReturnRows(sqlmock.NewRows([]string{"id", "order_id"}))

		// 3. Update Order status to Completed
		sqlMock.ExpectExec(`UPDATE "orders" SET`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		sqlMock.ExpectCommit()

		payload := biteship.WebhookTrackingPayload{
			OrderID:   "bite-999",
			Status:    "delivered",
			WaybillID: "SOCAG0001",
		}

		err := uc.HandleWebhook(context.Background(), payload)
		assert.NoError(t, err)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("webhook shipped updates allocated shipment to in_transit and order to shipped", func(t *testing.T) {
		gormDB, sqlMock := setupGormSqlmock(t)
		mockShipmentRepo := new(MockShipmentRepository)

		uc := usecase.NewShippingUsecase(gormDB, nil, mockShipmentRepo, nil, nil, nil, defaultStoreShippingConfig())

		shipment := &entity.Shipment{
			ID:              2,
			OrderID:         20,
			BiteshipOrderID: "bite-order-abc",
			Status:          entity.ShippingStatusAllocated,
		}
		mockShipmentRepo.On("FindByBiteshipOrderID", mock.Anything, "bite-order-abc").Return(shipment, nil).Once()

		sqlMock.ExpectBegin()

		// 1. Update Shipment to in_transit
		sqlMock.ExpectExec(`UPDATE "shipments" SET`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// 2. Preload order inside transaction
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "orders" WHERE id = $1 ORDER BY "orders"."id" LIMIT $2`)).
			WithArgs(20, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).
				AddRow(20, entity.OrderStatusProcessing))

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "order_items" WHERE "order_items"."order_id" = $1`)).
			WithArgs(20).
			WillReturnRows(sqlmock.NewRows([]string{"id", "order_id"}))

		// 3. Update Order status to Shipped
		sqlMock.ExpectExec(`UPDATE "orders" SET`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		sqlMock.ExpectCommit()

		payload := biteship.WebhookTrackingPayload{
			ID:        "bite-order-abc",
			Status:    "shipped",
			WaybillID: "SOCAG0002",
		}

		err := uc.HandleWebhook(context.Background(), payload)
		assert.NoError(t, err)
		assert.Equal(t, entity.ShippingStatusInTransit, shipment.Status)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("webhook lookup falls back to FindByIdentifier when FindByBiteshipOrderID returns nil", func(t *testing.T) {
		gormDB, sqlMock := setupGormSqlmock(t)
		mockShipmentRepo := new(MockShipmentRepository)

		uc := usecase.NewShippingUsecase(gormDB, nil, mockShipmentRepo, nil, nil, nil, defaultStoreShippingConfig())

		shipment := &entity.Shipment{
			ID:              3,
			OrderID:         30,
			BiteshipOrderID: "bite-order-real",
			Status:          entity.ShippingStatusAllocated,
		}
		mockShipmentRepo.On("FindByBiteshipOrderID", mock.Anything, "INV-20261018-001").Return(nil, nil).Once()
		mockShipmentRepo.On("FindByIdentifier", mock.Anything, "INV-20261018-001").Return(shipment, nil).Once()

		sqlMock.ExpectBegin()

		sqlMock.ExpectExec(`UPDATE "shipments" SET`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "orders" WHERE id = $1 ORDER BY "orders"."id" LIMIT $2`)).
			WithArgs(30, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).
				AddRow(30, entity.OrderStatusProcessing))

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "order_items" WHERE "order_items"."order_id" = $1`)).
			WithArgs(30).
			WillReturnRows(sqlmock.NewRows([]string{"id", "order_id"}))

		sqlMock.ExpectExec(`UPDATE "orders" SET`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		sqlMock.ExpectCommit()

		payload := biteship.WebhookTrackingPayload{
			OrderID:   "INV-20261018-001",
			Status:    "in_transit",
			WaybillID: "WAYBILL-333",
		}

		err := uc.HandleWebhook(context.Background(), payload)
		assert.NoError(t, err)
		assert.Equal(t, entity.ShippingStatusInTransit, shipment.Status)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}
