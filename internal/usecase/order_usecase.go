package usecase

import (
	"context"
	"fmt"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"github.com/bubblevy/bookstore/pkg/pdf"
)

type OrderUsecase interface {
	GetUserOrders(ctx context.Context, userID uint, statusFilter string) ([]entity.Order, error)
	GetOrderDetail(ctx context.Context, userID uint, invoiceNumber string) (*entity.Order, error)
	AdminFindAllOrders(ctx context.Context, filter repository.AdminOrderFilter, pq pagination.Query) ([]dto.AdminOrderListItemResponse, pagination.Meta, error)
	AdminGetOrderDetail(ctx context.Context, invoiceNumber string) (*dto.AdminOrderDetailResponse, error)
	GetOrderInvoicePDF(ctx context.Context, userID uint, invoiceNumber string, isAdmin bool) ([]byte, string, error)
}

type orderUsecase struct {
	orderRepo repository.OrderRepository
}

func NewOrderUsecase(orderRepo repository.OrderRepository) OrderUsecase {
	return &orderUsecase{orderRepo: orderRepo}
}

func (u *orderUsecase) GetUserOrders(ctx context.Context, userID uint, statusFilter string) ([]entity.Order, error) {
	return u.orderRepo.FindUserOrders(ctx, userID, statusFilter)
}

func (u *orderUsecase) GetOrderDetail(ctx context.Context, userID uint, invoiceNumber string) (*entity.Order, error) {
	order, err := u.orderRepo.FindOrderDetailByInvoice(ctx, userID, invoiceNumber)
	if err != nil {
		return nil, entity.ErrOrderNotFound
	}
	return order, nil
}

func (u *orderUsecase) AdminFindAllOrders(ctx context.Context, filter repository.AdminOrderFilter, pq pagination.Query) ([]dto.AdminOrderListItemResponse, pagination.Meta, error) {
	orders, total, err := u.orderRepo.FindAllOrders(ctx, filter, pq)
	if err != nil {
		return nil, pagination.Meta{}, fmt.Errorf("failed fetching admin orders: %w", err)
	}

	items := make([]dto.AdminOrderListItemResponse, len(orders))
	for i, o := range orders {
		totalQty := 0
		for _, item := range o.Items {
			totalQty += item.Quantity
		}

		var custName string
		var custEmail string
		if o.User != nil {
			custName = o.User.Name
			custEmail = o.User.Email
		} else {
			custName = o.RecipientName
		}

		var paymentMethod *string
		if o.Payment != nil {
			method := o.Payment.PaymentType
			if o.Payment.Bank != nil {
				method = fmt.Sprintf("%s (%s)", o.Payment.PaymentType, *o.Payment.Bank)
			}
			paymentMethod = &method
		}

		items[i] = dto.AdminOrderListItemResponse{
			ID:            o.ID,
			InvoiceNumber: o.InvoiceNumber,
			Status:        string(o.Status),
			Customer: dto.AdminCustomerResponse{
				ID:    o.UserID,
				Name:  custName,
				Email: custEmail,
			},
			TotalAmount:   o.TotalAmount,
			TotalQuantity: totalQty,
			PaymentMethod: paymentMethod,
			CreatedAt:     o.CreatedAt,
		}
	}

	meta := pagination.BuildMeta(total, pq.Page, pq.PerPage)
	return items, meta, nil
}

func (u *orderUsecase) AdminGetOrderDetail(ctx context.Context, invoiceNumber string) (*dto.AdminOrderDetailResponse, error) {
	order, err := u.orderRepo.FindOrderDetailByInvoice(ctx, 0, invoiceNumber)
	if err != nil {
		return nil, entity.ErrOrderNotFound
	}

	var custName string
	var custEmail string
	if order.User != nil {
		custName = order.User.Name
		custEmail = order.User.Email
	} else {
		custName = order.RecipientName
	}

	itemsResp := make([]dto.OrderItemResponse, len(order.Items))
	for i, it := range order.Items {
		itemsResp[i] = dto.OrderItemResponse{
			BookID:      it.BookID,
			Title:       it.Title,
			CoverImage:  it.CoverImage,
			Price:       it.Price,
			Quantity:    it.Quantity,
			WeightGrams: it.WeightGrams,
			Subtotal:    it.Subtotal,
		}
	}

	internalTrackingURL := fmt.Sprintf("/orders/%s/tracking", order.InvoiceNumber)
	shippingStatus := "pending"
	if order.Shipment != nil {
		shippingStatus = string(order.Shipment.Status)
	}

	var paymentResp *dto.OrderPaymentResponse
	if order.Payment != nil {
		paymentResp = &dto.OrderPaymentResponse{
			PaymentType: order.Payment.PaymentType,
			Bank:        order.Payment.Bank,
			VANumber:    order.Payment.VANumber,
			BillKey:     order.Payment.BillKey,
			BillerCode:  order.Payment.BillerCode,
			QRURL:       order.Payment.QRURL,
			Status:      string(order.Payment.Status),
			PaidAt:      order.Payment.PaidAt,
		}
	}

	return &dto.AdminOrderDetailResponse{
		ID:                 order.ID,
		InvoiceNumber:      order.InvoiceNumber,
		Status:             string(order.Status),
		Customer: dto.AdminCustomerResponse{
			ID:    order.UserID,
			Name:  custName,
			Email: custEmail,
		},
		PaymentDeadline:    order.PaymentDeadline,
		PaidAt:             order.PaidAt,
		CancelledAt:        order.CancelledAt,
		CancellationReason: order.CancellationReason,
		ReturnedAt:         order.ReturnedAt,
		ReturnReason:       order.ReturnReason,
		CreatedAt:          order.CreatedAt,
		Recipient: dto.OrderRecipientResponse{
			Name:        order.RecipientName,
			Phone:       order.RecipientPhone,
			Address:     order.RecipientAddress,
			Province:    order.ProvinceName,
			City:        order.CityName,
			District:    order.DistrictName,
			Subdistrict: order.SubdistrictName,
			PostalCode:  order.PostalCode,
		},
		Shipping: dto.OrderShippingResponse{
			Courier:        order.Courier,
			CourierService: order.CourierService,
			TrackingNumber: order.ShippingTrackingNumber,
			Status:         shippingStatus,
			TrackingURL:    internalTrackingURL,
		},
		Payment: paymentResp,
		Items:   itemsResp,
		Pricing: dto.OrderPricingResponse{
			ItemsSubtotal:  order.ItemsSubtotal,
			ShippingCost:   order.ShippingCost,
			DiscountAmount: order.DiscountAmount,
			AdminFee:       order.AdminFee,
			TotalAmount:    order.TotalAmount,
		},
	}, nil
}

func (u *orderUsecase) GetOrderInvoicePDF(ctx context.Context, userID uint, invoiceNumber string, isAdmin bool) ([]byte, string, error) {
	queryUserID := userID
	if isAdmin {
		queryUserID = 0 // Admin can download any order invoice
	}

	order, err := u.orderRepo.FindOrderDetailByInvoice(ctx, queryUserID, invoiceNumber)
	if err != nil {
		return nil, "", entity.ErrOrderNotFound
	}

	if !isAdmin && order.UserID != userID {
		return nil, "", entity.ErrOrderNotFound
	}

	pdfBytes, err := pdf.GenerateInvoicePDF(order)
	if err != nil {
		return nil, "", fmt.Errorf("failed generating invoice pdf: %w", err)
	}

	filename := fmt.Sprintf("invoice-%s.pdf", order.InvoiceNumber)
	return pdfBytes, filename, nil
}
