package repository

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
	"gorm.io/gorm"
)

type ShipmentRepository interface {
	WithTx(tx *gorm.DB) ShipmentRepository
	Create(ctx context.Context, shipment *entity.Shipment) error
	FindByOrderID(ctx context.Context, orderID uint) (*entity.Shipment, error)
	FindByInvoiceNumber(ctx context.Context, invoiceNumber string) (*entity.Shipment, error)
	FindByBiteshipOrderID(ctx context.Context, biteshipOrderID string) (*entity.Shipment, error)
	FindByIdentifier(ctx context.Context, identifier string) (*entity.Shipment, error)
	Update(ctx context.Context, shipment *entity.Shipment) error
}
