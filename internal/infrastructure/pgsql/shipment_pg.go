package pgsql

import (
	"context"
	"errors"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"gorm.io/gorm"
)

type shipmentRepository struct {
	db *gorm.DB
}

func NewShipmentRepository(db *gorm.DB) repository.ShipmentRepository {
	return &shipmentRepository{db: db}
}

func (r *shipmentRepository) WithTx(tx *gorm.DB) repository.ShipmentRepository {
	return &shipmentRepository{db: tx}
}

func (r *shipmentRepository) Create(ctx context.Context, shipment *entity.Shipment) error {
	return r.db.WithContext(ctx).Create(shipment).Error
}

func (r *shipmentRepository) FindByOrderID(ctx context.Context, orderID uint) (*entity.Shipment, error) {
	var s entity.Shipment
	if err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&s).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *shipmentRepository) FindByBiteshipOrderID(ctx context.Context, biteshipOrderID string) (*entity.Shipment, error) {
	var s entity.Shipment
	if err := r.db.WithContext(ctx).Where("biteship_order_id = ?", biteshipOrderID).First(&s).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *shipmentRepository) Update(ctx context.Context, shipment *entity.Shipment) error {
	return r.db.WithContext(ctx).
		Model(&entity.Shipment{}).
		Where("id = ?", shipment.ID).
		Updates(map[string]interface{}{
			"waybill_number":       shipment.WaybillNumber,
			"biteship_tracking_id": shipment.BiteshipTrackingID,
			"status":               shipment.Status,
			"tracking_url":         shipment.TrackingURL,
			"updated_at":           gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error
}
