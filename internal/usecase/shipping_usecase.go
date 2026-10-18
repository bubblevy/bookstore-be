package usecase

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/biteship"
	"github.com/bubblevy/bookstore/pkg/logger"
	"gorm.io/gorm"
)

type StoreShippingConfig struct {
	ShipperName       string
	ShipperPhone      string
	ShipperEmail      string
	OriginName        string
	OriginPhone       string
	OriginAddr        string
	OriginAreaID      string
	OriginPostalCode  int
	AvailableCouriers string
}

type GetShippingRatesInput struct {
	CartItemIDs []uint
	AddressID   *uint
}

type CourierOptionResult struct {
	CourierCode        string
	CourierName        string
	CourierServiceName string
	CourierServiceCode string
	Description        string
	Price              int
	ETD                string
	Type               string
}

type ShippingRatesResult struct {
	TotalWeightGrams int
	DestinationArea  string
	Options          []CourierOptionResult
}

type FulfillOrderResult struct {
	ShipmentID      uint
	InvoiceNumber   string
	BiteshipOrderID string
	WaybillNumber   *string
	Status          entity.ShippingStatus
	Courier         string
	CourierService  string
	TrackingURL     *string
}

type ShippingUsecase interface {
	SearchAreas(ctx context.Context, keyword string) ([]biteship.AreaItem, error)
	GetCouriers(ctx context.Context) ([]biteship.CourierDetail, error)
	GetShippingRates(ctx context.Context, userID uint, input GetShippingRatesInput) (*ShippingRatesResult, error)
	FulfillOrder(ctx context.Context, invoiceNumber string, orderNote string) (*FulfillOrderResult, error)
	GetOrderTracking(ctx context.Context, userID uint, invoiceNumber string) (*biteship.TrackingDetailResult, error)
	ProcessReturn(ctx context.Context, invoiceNumber string, reason string) error
	HandleWebhook(ctx context.Context, payload biteship.WebhookTrackingPayload) error
}

type shippingUsecase struct {
	db           *gorm.DB
	orderRepo    repository.OrderRepository
	shipmentRepo repository.ShipmentRepository
	checkoutRepo repository.CheckoutRepository
	addressRepo  repository.AddressRepository
	biteship     biteship.Client
	cfg          StoreShippingConfig
}

func NewShippingUsecase(
	db *gorm.DB,
	orderRepo repository.OrderRepository,
	shipmentRepo repository.ShipmentRepository,
	checkoutRepo repository.CheckoutRepository,
	addressRepo repository.AddressRepository,
	biteship biteship.Client,
	cfg StoreShippingConfig,
) ShippingUsecase {
	return &shippingUsecase{
		db:           db,
		orderRepo:    orderRepo,
		shipmentRepo: shipmentRepo,
		checkoutRepo: checkoutRepo,
		addressRepo:  addressRepo,
		biteship:     biteship,
		cfg:          cfg,
	}
}

func (u *shippingUsecase) SearchAreas(ctx context.Context, keyword string) ([]biteship.AreaItem, error) {
	return u.biteship.SearchArea(ctx, keyword)
}

func (u *shippingUsecase) GetCouriers(ctx context.Context) ([]biteship.CourierDetail, error) {
	return u.biteship.GetCouriers(ctx)
}

func (u *shippingUsecase) GetShippingRates(ctx context.Context, userID uint, input GetShippingRatesInput) (*ShippingRatesResult, error) {
	if len(input.CartItemIDs) == 0 {
		return nil, entity.ErrCartEmpty
	}

	var address *entity.Address
	var err error
	if input.AddressID != nil && *input.AddressID > 0 {
		address, err = u.addressRepo.FindByID(ctx, *input.AddressID)
	} else {
		address, err = u.addressRepo.FindPrimaryByUserID(ctx, userID)
	}
	if err != nil || address == nil || address.UserID != userID {
		return nil, entity.ErrAddressNotFound
	}

	cartItems, err := u.checkoutRepo.GetCartItemsWithBooks(ctx, userID, input.CartItemIDs)
	if err != nil {
		return nil, err
	}
	if len(cartItems) == 0 {
		return nil, entity.ErrCartItemNotFound
	}

	var totalWeightGrams int
	rateItems := make([]biteship.RateItem, len(cartItems))
	for i, item := range cartItems {
		if !item.Book.IsActive {
			return nil, entity.ErrBookNotFound
		}
		totalWeightGrams += item.Book.WeightGrams * item.Quantity

		rateItems[i] = biteship.RateItem{
			Name:     item.Book.Title,
			Value:    int64(item.Book.Price),
			Weight:   item.Book.WeightGrams,
			Quantity: item.Quantity,
		}
	}

	couriers := strings.TrimSpace(u.cfg.AvailableCouriers)
	if couriers == "" {
		couriers = "jne,sicepat,jnt,anteraja,tiki,pos"
	}

	biteshipRates, err := u.biteship.CalculateRatesByAreaID(ctx, address.BiteshipAreaID, couriers, rateItems)
	if err != nil {
		return nil, entity.ErrShippingRateFailed
	}

	options := make([]CourierOptionResult, 0, len(biteshipRates))
	for _, rate := range biteshipRates {
		etd := strings.TrimSpace(rate.Duration)
		if etd == "" {
			etd = strings.TrimSpace(rate.ShipmentDurationRange)
		}
		if etd == "" {
			etd = "-"
		}

		options = append(options, CourierOptionResult{
			CourierCode:        rate.CourierCode,
			CourierName:        rate.CourierName,
			CourierServiceName: rate.CourierServiceName,
			CourierServiceCode: rate.CourierServiceCode,
			Description:        rate.Description,
			Price:              int(rate.Price),
			ETD:                etd,
			Type:               rate.Type,
		})
	}

	fullAddress := fmt.Sprintf("%s, Kel. %s, Kec. %s, %s, %s %s",
		address.AddressLine,
		address.SubdistrictName,
		address.DistrictName,
		address.CityName,
		address.ProvinceName,
		address.PostalCode,
	)

	return &ShippingRatesResult{
		TotalWeightGrams: totalWeightGrams,
		DestinationArea:  fullAddress,
		Options:          options,
	}, nil
}

func (u *shippingUsecase) FulfillOrder(ctx context.Context, invoiceNumber string, orderNote string) (*FulfillOrderResult, error) {
	var order entity.Order
	if err := u.db.WithContext(ctx).Preload("Items").Where("invoice_number = ?", invoiceNumber).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrOrderNotFound
		}
		return nil, err
	}

	if order.Status != entity.OrderStatusPaid {
		return nil, entity.ErrOrderNotPaid
	}

	existing, err := u.shipmentRepo.FindByOrderID(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, entity.ErrOrderAlreadyFulfilled
	}

	rateItems := make([]biteship.RateItem, 0, len(order.Items))
	for _, it := range order.Items {
		rateItems = append(rateItems, biteship.RateItem{
			Name:     it.Title,
			Value:    int64(it.Price),
			Length:   20,
			Width:    14,
			Height:   2,
			Weight:   it.WeightGrams,
			Quantity: it.Quantity,
		})
	}

	destPostalCode, _ := strconv.Atoi(strings.TrimSpace(order.PostalCode))
	biteshipReq := biteship.CreateOrderByAreaIDRequest{
		ShipperContactName:      u.cfg.ShipperName,
		ShipperContactPhone:     u.cfg.ShipperPhone,
		ShipperContactEmail:     u.cfg.ShipperEmail,
		OriginContactName:       u.cfg.OriginName,
		OriginContactPhone:      u.cfg.OriginPhone,
		OriginAddress:           u.cfg.OriginAddr,
		OriginAreaID:            u.cfg.OriginAreaID,
		OriginPostalCode:        u.cfg.OriginPostalCode,
		DestinationContactName:  order.RecipientName,
		DestinationContactPhone: order.RecipientPhone,
		DestinationAddress:      order.RecipientAddress,
		DestinationAreaID:       order.DestinationAreaID,
		DestinationPostalCode:   destPostalCode,
		CourierCompany:          strings.ToLower(order.Courier),
		CourierType:             strings.ToLower(order.CourierService),
		DeliveryType:            "now",
		OrderNote:               orderNote,
		Metadata: map[string]interface{}{
			"order_id":       order.ID,
			"invoice_number": order.InvoiceNumber,
		},
		Items: rateItems,
	}

	biteshipRes, err := u.biteship.CreateOrderByAreaID(ctx, biteshipReq)
	if err != nil {
		logger.Error("dispatch courier order failed", "invoice", invoiceNumber, "courier", order.Courier, "error", err)
		return nil, fmt.Errorf("%w: %s", entity.ErrDispatchCourierFailed, err.Error())
	}

	var notePtr *string
	if strings.TrimSpace(orderNote) != "" {
		notePtr = &orderNote
	}
	var waybill *string
	if biteshipRes.Courier.WaybillID != "" {
		waybill = &biteshipRes.Courier.WaybillID
	}
	var trackingID *string
	if biteshipRes.Courier.TrackingID != "" {
		trackingID = &biteshipRes.Courier.TrackingID
	}

	internalTrackingURL := fmt.Sprintf("/orders/%s/tracking", order.InvoiceNumber)
	shipment := &entity.Shipment{
		OrderID:            order.ID,
		BiteshipOrderID:    biteshipRes.ID,
		BiteshipTrackingID: trackingID,
		WaybillNumber:      waybill,
		Courier:            order.Courier,
		CourierService:     order.CourierService,
		Status:             entity.ShippingStatusAllocated,
		TrackingURL:        &internalTrackingURL,
		OrderNote:          notePtr,
	}

	now := time.Now()
	err = u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(shipment).Error; err != nil {
			return err
		}
		return tx.Model(&entity.Order{}).
			Where("id = ?", order.ID).
			Updates(map[string]interface{}{
				"status":                   entity.OrderStatusProcessing,
				"shipping_tracking_number": waybill,
				"updated_at":               now,
			}).Error
	})
	if err != nil {
		logger.Error("save shipment transaction failed", "invoice", invoiceNumber, "error", err)
		return nil, err
	}

	logger.Info("order fulfilled and courier dispatched",
		"invoice", order.InvoiceNumber,
		"courier", order.Courier,
		"courier_service", order.CourierService,
		"biteship_order_id", shipment.BiteshipOrderID,
		"waybill_id", biteshipRes.Courier.WaybillID,
	)

	return &FulfillOrderResult{
		ShipmentID:      shipment.ID,
		InvoiceNumber:   order.InvoiceNumber,
		BiteshipOrderID: shipment.BiteshipOrderID,
		WaybillNumber:   shipment.WaybillNumber,
		Status:          shipment.Status,
		Courier:         shipment.Courier,
		CourierService:  shipment.CourierService,
		TrackingURL:     shipment.TrackingURL,
	}, nil
}

func (u *shippingUsecase) GetOrderTracking(ctx context.Context, userID uint, invoiceNumber string) (*biteship.TrackingDetailResult, error) {
	var order entity.Order
	if err := u.db.WithContext(ctx).Where("invoice_number = ?", invoiceNumber).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrOrderNotFound
		}
		return nil, err
	}

	if userID != 0 && order.UserID != userID {
		return nil, entity.ErrUnauthorizedTracking
	}

	shipment, err := u.shipmentRepo.FindByOrderID(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	if shipment == nil {
		return nil, entity.ErrShipmentNotFound
	}

	var tracking *biteship.TrackingDetailResult
	if shipment.WaybillNumber != nil && *shipment.WaybillNumber != "" {
		tracking, err = u.biteship.TrackWaybill(ctx, *shipment.WaybillNumber, shipment.Courier)
	} else if shipment.BiteshipTrackingID != nil && *shipment.BiteshipTrackingID != "" {
		tracking, err = u.biteship.TrackByID(ctx, *shipment.BiteshipTrackingID)
	} else {
		return nil, entity.ErrTrackingNumberNotReady
	}
	if err != nil {
		return nil, err
	}

	if tracking.CourierCode == "" {
		tracking.CourierCode = strings.ToLower(shipment.Courier)
	}
	if tracking.CourierName == "" {
		tracking.CourierName = strings.ToUpper(shipment.Courier)
	}
	if tracking.Courier.Company == "" {
		tracking.Courier.Company = tracking.CourierCode
	}
	if tracking.Courier.Name == "" {
		tracking.Courier.Name = tracking.CourierName
	}

	return tracking, nil
}

func (u *shippingUsecase) HandleWebhook(ctx context.Context, payload biteship.WebhookTrackingPayload) error {
	orderID := payload.GetOrderID()
	trackingID := payload.GetTrackingID()
	waybillID := payload.GetWaybillID()
	statusStr := payload.GetStatus()

	logger.Info("biteship webhook received",
		"biteship_order_id", orderID,
		"tracking_id", trackingID,
		"waybill_id", waybillID,
		"status", statusStr,
	)

	var shipment *entity.Shipment
	var err error
	if orderID != "" {
		shipment, err = u.shipmentRepo.FindByBiteshipOrderID(ctx, orderID)
	}
	if (shipment == nil || err != nil) && orderID != "" {
		shipment, err = u.shipmentRepo.FindByIdentifier(ctx, orderID)
	}
	if (shipment == nil || err != nil) && trackingID != "" {
		shipment, err = u.shipmentRepo.FindByIdentifier(ctx, trackingID)
	}
	if (shipment == nil || err != nil) && waybillID != "" {
		shipment, err = u.shipmentRepo.FindByIdentifier(ctx, waybillID)
	}

	if err != nil || shipment == nil {
		logger.Warn("biteship webhook ignored - shipment not found",
			"biteship_order_id", orderID,
			"tracking_id", trackingID,
			"waybill_id", waybillID,
		)
		return nil
	}

	var mappedStatus entity.ShippingStatus
	var mappedOrderStatus *entity.OrderStatus

	cleanStatus := strings.ToLower(strings.TrimSpace(statusStr))
	cleanStatus = strings.ReplaceAll(cleanStatus, "_", "")
	cleanStatus = strings.ReplaceAll(cleanStatus, "-", "")
	cleanStatus = strings.ReplaceAll(cleanStatus, " ", "")

	switch cleanStatus {
	case "placed":
		mappedStatus = entity.ShippingStatusPlaced
	case "scheduled":
		mappedStatus = entity.ShippingStatusScheduled
	case "confirmed":
		mappedStatus = entity.ShippingStatusConfirmed
	case "allocated":
		mappedStatus = entity.ShippingStatusAllocated
	case "pickingup", "pickup":
		mappedStatus = entity.ShippingStatusPickingUp
	case "picked", "pickedup":
		mappedStatus = entity.ShippingStatusPicked
		st := entity.OrderStatusShipped
		mappedOrderStatus = &st
	case "droppingoff":
		mappedStatus = entity.ShippingStatusDroppingOff
		st := entity.OrderStatusShipped
		mappedOrderStatus = &st
	case "intransit", "transit", "shipped", "shipping", "delivering", "onprocess":
		mappedStatus = entity.ShippingStatusInTransit
		st := entity.OrderStatusShipped
		mappedOrderStatus = &st
	case "returnintransit", "returntransit":
		mappedStatus = entity.ShippingStatusReturnInTransit
		st := entity.OrderStatusShipped
		mappedOrderStatus = &st
	case "delivered":
		mappedStatus = entity.ShippingStatusDelivered
		st := entity.OrderStatusCompleted
		mappedOrderStatus = &st
	case "rejected":
		mappedStatus = entity.ShippingStatusRejected
	case "couriernotfound":
		mappedStatus = entity.ShippingStatusCourierNotFound
	case "cancelled", "canceled":
		mappedStatus = entity.ShippingStatusCancelled
	case "returned":
		mappedStatus = entity.ShippingStatusReturned
		st := entity.OrderStatusReturned
		mappedOrderStatus = &st
	case "onhold":
		mappedStatus = entity.ShippingStatusOnHold
	case "disposed":
		mappedStatus = entity.ShippingStatusDisposed
		st := entity.OrderStatusReturned
		mappedOrderStatus = &st
	default:
		logger.Warn("biteship webhook unhandled status", "status", statusStr, "clean_status", cleanStatus)
		return nil
	}

	shipment.Status = mappedStatus
	if waybillID != "" {
		shipment.WaybillNumber = &waybillID
	}
	if trackingID != "" {
		shipment.BiteshipTrackingID = &trackingID
	}

	now := time.Now()
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		shipmentUpdates := map[string]interface{}{
			"status":     shipment.Status,
			"updated_at": now,
		}
		if shipment.WaybillNumber != nil && *shipment.WaybillNumber != "" {
			shipmentUpdates["waybill_number"] = *shipment.WaybillNumber
		}
		if shipment.BiteshipTrackingID != nil && *shipment.BiteshipTrackingID != "" {
			shipmentUpdates["biteship_tracking_id"] = *shipment.BiteshipTrackingID
		}
		if err := tx.Model(&entity.Shipment{}).Where("id = ?", shipment.ID).Updates(shipmentUpdates).Error; err != nil {
			return err
		}

		if mappedOrderStatus != nil {
			var currentOrder entity.Order
			if err := tx.Preload("Items").Where("id = ?", shipment.OrderID).First(&currentOrder).Error; err != nil {
				return err
			}

			if currentOrder.Status == entity.OrderStatusCompleted && *mappedOrderStatus != entity.OrderStatusReturned {
				return nil
			}

			orderUpdates := map[string]interface{}{
				"status":     *mappedOrderStatus,
				"updated_at": now,
			}
			if waybillID != "" && (currentOrder.ShippingTrackingNumber == nil || *currentOrder.ShippingTrackingNumber == "") {
				orderUpdates["shipping_tracking_number"] = waybillID
			}

			if cleanStatus == "returnintransit" || cleanStatus == "returntransit" {
				reason := "Courier initiated Return to Sender (package is in transit back to origin)"
				if payload.Note != "" {
					reason = payload.Note
				}
				orderUpdates["return_reason"] = reason
			}

			if *mappedOrderStatus == entity.OrderStatusReturned {
				orderUpdates["returned_at"] = now
				reason := "Package successfully returned to warehouse"
				if cleanStatus == "disposed" {
					reason = "Package lost or damaged by courier provider during transit"
				}
				if payload.Note != "" {
					reason = payload.Note
				}
				orderUpdates["return_reason"] = reason

				if currentOrder.Status != entity.OrderStatusReturned {
					for _, item := range currentOrder.Items {
						if err := tx.Model(&entity.Book{}).
							Where("id = ?", item.BookID).
							Update("stock", gorm.Expr("stock + ?", item.Quantity)).Error; err != nil {
							return fmt.Errorf("failed restocking book ID %d: %w", item.BookID, err)
						}
					}
				}
			}

			return tx.Model(&entity.Order{}).Where("id = ?", currentOrder.ID).Updates(orderUpdates).Error
		}

		logger.Info("biteship webhook processed successfully",
			"biteship_order_id", orderID,
			"shipping_status", mappedStatus,
		)
		return nil
	})
}

func (u *shippingUsecase) ProcessReturn(ctx context.Context, invoiceNumber string, reason string) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order entity.Order
		if err := tx.Preload("Items").Where("invoice_number = ?", invoiceNumber).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return entity.ErrOrderNotFound
			}
			return err
		}

		if order.Status != entity.OrderStatusShipped && order.Status != entity.OrderStatusCompleted {
			return entity.ErrOrderInvalidReturnState
		}

		if order.Status == entity.OrderStatusReturned {
			return entity.ErrOrderAlreadyReturned
		}

		now := time.Now()
		returnReason := strings.TrimSpace(reason)
		if returnReason == "" {
			returnReason = "Package returned to warehouse"
		}

		if err := tx.Model(&entity.Order{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
			"status":        entity.OrderStatusReturned,
			"returned_at":   now,
			"return_reason": returnReason,
			"updated_at":    now,
		}).Error; err != nil {
			return err
		}

		if err := tx.Model(&entity.Shipment{}).Where("order_id = ?", order.ID).Updates(map[string]interface{}{
			"status":     entity.ShippingStatusReturned,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}

		for _, item := range order.Items {
			if err := tx.Model(&entity.Book{}).
				Where("id = ?", item.BookID).
				Update("stock", gorm.Expr("stock + ?", item.Quantity)).Error; err != nil {
				return fmt.Errorf("failed restocking book ID %d: %w", item.BookID, err)
			}
		}

		return nil
	})
}
