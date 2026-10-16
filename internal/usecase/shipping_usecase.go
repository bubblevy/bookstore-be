package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/biteship"
	"gorm.io/gorm"
)

type StoreShippingConfig struct {
	ShipperName  string
	ShipperPhone string
	ShipperEmail string
	OriginName   string
	OriginPhone  string
	OriginAddr   string
	OriginAreaID string
}

type FulfillOrderResult struct {
	ShipmentID      uint                  `json:"shipment_id"`
	OrderID         uint                  `json:"order_id"`
	BiteshipOrderID string                `json:"biteship_order_id"`
	WaybillNumber   *string               `json:"waybill_number,omitempty"`
	Status          entity.ShippingStatus `json:"status"`
	Courier         string                `json:"courier"`
	CourierService  string                `json:"courier_service"`
	TrackingURL     *string               `json:"tracking_url,omitempty"`
}

type ShippingUsecase interface {
	SearchAreas(ctx context.Context, keyword string) ([]biteship.AreaItem, error)
	GetCouriers(ctx context.Context) ([]biteship.CourierDetail, error)
	FulfillOrder(ctx context.Context, orderID uint, orderNote string) (*FulfillOrderResult, error)
	GetOrderTracking(ctx context.Context, userID, orderID uint) (*biteship.TrackingDetailResult, error)
	HandleWebhook(ctx context.Context, payload biteship.WebhookTrackingPayload) error
}

type shippingUsecase struct {
	db           *gorm.DB
	orderRepo    repository.OrderRepository
	shipmentRepo repository.ShipmentRepository
	biteship     biteship.Client
	cfg          StoreShippingConfig
}

func NewShippingUsecase(db *gorm.DB, orderRepo repository.OrderRepository, shipmentRepo repository.ShipmentRepository, biteship biteship.Client, cfg StoreShippingConfig) ShippingUsecase {
	return &shippingUsecase{
		db:           db,
		orderRepo:    orderRepo,
		shipmentRepo: shipmentRepo,
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

func (u *shippingUsecase) FulfillOrder(ctx context.Context, orderID uint, orderNote string) (*FulfillOrderResult, error) {
	var order entity.Order
	if err := u.db.WithContext(ctx).Preload("Items").Where("id = ?", orderID).First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entity.ErrOrderNotFound
		}
		return nil, err
	}

	if order.Status != entity.OrderStatusPaid {
		return nil, entity.ErrOrderNotPaid
	}

	existing, err := u.shipmentRepo.FindByOrderID(ctx, orderID)
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

	biteshipReq := biteship.CreateOrderByAreaIDRequest{
		ShipperContactName:      u.cfg.ShipperName,
		ShipperContactPhone:     u.cfg.ShipperPhone,
		ShipperContactEmail:     u.cfg.ShipperEmail,
		OriginContactName:       u.cfg.OriginName,
		OriginContactPhone:      u.cfg.OriginPhone,
		OriginAddress:           u.cfg.OriginAddr,
		OriginAreaID:            u.cfg.OriginAreaID,
		DestinationContactName:  order.RecipientName,
		DestinationContactPhone: order.RecipientPhone,
		DestinationAddress:      order.RecipientAddress,
		DestinationAreaID:       order.PostalCode,
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
		return nil, fmt.Errorf("%w: %s", entity.ErrDispatchCourierFailed, err.Error())
	}

	var notePtr *string
	if orderNote != "" {
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

	shipment := &entity.Shipment{
		OrderID:            order.ID,
		BiteshipOrderID:    biteshipRes.ID,
		BiteshipTrackingID: trackingID,
		WaybillNumber:      waybill,
		Courier:            order.Courier,
		CourierService:     order.CourierService,
		Status:             entity.ShippingStatusAllocated,
		TrackingURL:        &biteshipRes.Courier.TrackingURL,
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
		return nil, err
	}

	return &FulfillOrderResult{
		ShipmentID:      shipment.ID,
		OrderID:         shipment.OrderID,
		BiteshipOrderID: shipment.BiteshipOrderID,
		WaybillNumber:   shipment.WaybillNumber,
		Status:          shipment.Status,
		Courier:         shipment.Courier,
		CourierService:  shipment.CourierService,
		TrackingURL:     shipment.TrackingURL,
	}, nil
}

func (u *shippingUsecase) GetOrderTracking(ctx context.Context, userID, orderID uint) (*biteship.TrackingDetailResult, error) {
	var order entity.Order
	if err := u.db.WithContext(ctx).Where("id = ?", orderID).First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entity.ErrOrderNotFound
		}
		return nil, err
	}

	if userID != 0 && order.UserID != userID {
		return nil, entity.ErrUnauthorizedTracking
	}

	shipment, err := u.shipmentRepo.FindByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if shipment == nil {
		return nil, entity.ErrShipmentNotFound
	}

	if shipment.WaybillNumber != nil && *shipment.WaybillNumber != "" {
		return u.biteship.TrackWaybill(ctx, *shipment.WaybillNumber, shipment.Courier)
	}

	if shipment.BiteshipTrackingID != nil && *shipment.BiteshipTrackingID != "" {
		return u.biteship.TrackByID(ctx, *shipment.BiteshipTrackingID)
	}

	return nil, entity.ErrTrackingNumberNotReady
}

func (u *shippingUsecase) HandleWebhook(ctx context.Context, payload biteship.WebhookTrackingPayload) error {
	shipment, err := u.shipmentRepo.FindByBiteshipOrderID(ctx, payload.OrderID)
	if err != nil || shipment == nil {
		return nil
	}

	var mappedStatus entity.ShippingStatus
	var mappedOrderStatus *entity.OrderStatus

	switch strings.ToLower(strings.TrimSpace(payload.Status)) {
	case "placed":
		mappedStatus = entity.ShippingStatusPlaced
	case "scheduled":
		mappedStatus = entity.ShippingStatusScheduled
	case "confirmed":
		mappedStatus = entity.ShippingStatusConfirmed
	case "allocated":
		mappedStatus = entity.ShippingStatusAllocated
	case "picking_up":
		mappedStatus = entity.ShippingStatusPickingUp
	case "picked":
		mappedStatus = entity.ShippingStatusPicked
		st := entity.OrderStatusShipped
		mappedOrderStatus = &st
	case "dropping_off":
		mappedStatus = entity.ShippingStatusDroppingOff
		st := entity.OrderStatusShipped
		mappedOrderStatus = &st
	case "in_transit":
		mappedStatus = entity.ShippingStatusInTransit
		st := entity.OrderStatusShipped
		mappedOrderStatus = &st
	case "delivered":
		mappedStatus = entity.ShippingStatusDelivered
		st := entity.OrderStatusCompleted
		mappedOrderStatus = &st
	case "rejected":
		mappedStatus = entity.ShippingStatusRejected
	case "courier_not_found":
		mappedStatus = entity.ShippingStatusCourierNotFound
	case "cancelled":
		mappedStatus = entity.ShippingStatusCancelled
	case "returned":
		mappedStatus = entity.ShippingStatusReturned
	case "on_hold":
		mappedStatus = entity.ShippingStatusOnHold
	case "disposed":
		mappedStatus = entity.ShippingStatusDisposed
	default:
		return nil
	}

	shipment.Status = mappedStatus
	if payload.WaybillID != "" {
		shipment.WaybillNumber = &payload.WaybillID
	}

	now := time.Now()
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.Shipment{}).
			Where("id = ?", shipment.ID).
			Updates(map[string]interface{}{
				"status":         shipment.Status,
				"waybill_number": shipment.WaybillNumber,
				"updated_at":     now,
			}).Error; err != nil {
			return err
		}

		if mappedOrderStatus != nil {
			orderUpdates := map[string]interface{}{
				"status":     *mappedOrderStatus,
				"updated_at": now,
			}
			if payload.WaybillID != "" {
				orderUpdates["shipping_tracking_number"] = payload.WaybillID
			}
			return tx.Model(&entity.Order{}).Where("id = ?", shipment.OrderID).Updates(orderUpdates).Error
		}

		return nil
	})
}
