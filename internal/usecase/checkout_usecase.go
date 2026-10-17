package usecase

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/biteship"
	"github.com/bubblevy/bookstore/pkg/payment"
	"gorm.io/gorm"
)

type PreviewCheckoutInput struct {
	CartItemIDs    []uint
	AddressID      *uint
	Courier        string
	CourierService string
	DiscountCode   *string
	PaymentMethod  string
}

type PreviewCheckoutResult struct {
	Address          *entity.Address
	Items            []entity.OrderItem
	ItemsSubtotal    float64
	TotalWeightGrams int
	Courier          string
	CourierService   string
	ShippingCost     float64
	DiscountCode     *string
	DiscountAmount   float64
	PaymentMethod    string
	AdminFee         float64
	TotalPayment     float64
	DiscountID       *uint
}

type CreateOrderInput struct {
	CartItemIDs    []uint
	AddressID      *uint
	Courier        string
	CourierService string
	DiscountCode   *string
	PaymentMethod  string
}

type CreateOrderResult struct {
	InvoiceNumber   string
	Status          string
	TotalAmount     float64
	PaymentDeadline time.Time
	PaymentInfo     struct {
		PaymentType string
		Bank        *string
		VANumber    *string
		BillKey     *string
		BillerCode  *string
		QRURL       *string
	}
}

type MidtransWebhookInput struct {
	TransactionTime   string
	TransactionStatus string
	TransactionID     string
	StatusMessage     string
	StatusCode        string
	SignatureKey      string
	PaymentType       string
	OrderID           string
	GrossAmount       string
	FraudStatus       string
	Currency          string
	RawJSON           []byte
}

type CheckoutUsecase interface {
	PreviewCheckout(ctx context.Context, userID uint, input PreviewCheckoutInput) (*PreviewCheckoutResult, error)
	CreateOrder(ctx context.Context, userID uint, input CreateOrderInput) (*CreateOrderResult, error)
	HandleMidtransWebhook(ctx context.Context, input MidtransWebhookInput) error
}

type checkoutUsecase struct {
	db              *gorm.DB
	checkoutRepo    repository.CheckoutRepository
	orderRepo       repository.OrderRepository
	addressRepo     repository.AddressRepository
	discountUsecase DiscountUsecase
	discountRepo    repository.DiscountRepository
	biteshipSvc     biteship.Client
	midtransSvc     payment.MidtransService
}

func NewCheckoutUsecase(
	db *gorm.DB,
	checkoutRepo repository.CheckoutRepository,
	orderRepo repository.OrderRepository,
	addressRepo repository.AddressRepository,
	discountUsecase DiscountUsecase,
	discountRepo repository.DiscountRepository,
	biteshipSvc biteship.Client,
	midtransSvc payment.MidtransService,
) CheckoutUsecase {
	return &checkoutUsecase{
		db:              db,
		checkoutRepo:    checkoutRepo,
		orderRepo:       orderRepo,
		addressRepo:     addressRepo,
		discountUsecase: discountUsecase,
		discountRepo:    discountRepo,
		biteshipSvc:     biteshipSvc,
		midtransSvc:     midtransSvc,
	}
}

func (u *checkoutUsecase) PreviewCheckout(ctx context.Context, userID uint, input PreviewCheckoutInput) (*PreviewCheckoutResult, error) {
	if len(input.CartItemIDs) == 0 {
		return nil, entity.ErrCartEmpty
	}

	// 1. Fetch destination address snapshot
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

	// 2. Fetch cart items with book details
	cartItems, err := u.checkoutRepo.GetCartItemsWithBooks(ctx, userID, input.CartItemIDs)
	if err != nil {
		return nil, err
	}
	if len(cartItems) == 0 {
		return nil, entity.ErrCartItemNotFound
	}

	var itemsSubtotal float64
	var totalWeightGrams int
	orderItems := make([]entity.OrderItem, len(cartItems))
	rateItems := make([]biteship.RateItem, len(cartItems))
	discountCartItems := make([]CartItemInput, len(cartItems))

	for i, item := range cartItems {
		if !item.Book.IsActive {
			return nil, entity.ErrBookNotFound
		}
		if item.Book.Stock < item.Quantity {
			return nil, entity.ErrInsufficientStock
		}

		subtotal := item.Book.Price * float64(item.Quantity)
		itemsSubtotal += subtotal
		totalWeightGrams += item.Book.WeightGrams * item.Quantity

		orderItems[i] = entity.OrderItem{
			BookID:      item.BookID,
			Title:       item.Book.Title,
			CoverImage:  item.Book.CoverImage,
			Price:       item.Book.Price,
			WeightGrams: item.Book.WeightGrams,
			Quantity:    item.Quantity,
			Subtotal:    subtotal,
		}

		rateItems[i] = biteship.RateItem{
			Name:     item.Book.Title,
			Value:    int64(item.Book.Price),
			Weight:   item.Book.WeightGrams,
			Quantity: item.Quantity,
		}

		discountCartItems[i] = CartItemInput{
			BookID:   item.BookID,
			Quantity: item.Quantity,
		}
	}

	// 3. Calculate shipping cost via Biteship Area ID
	rates, err := u.biteshipSvc.CalculateRatesByAreaID(ctx, address.BiteshipAreaID, input.Courier, rateItems)
	if err != nil || len(rates) == 0 {
		return nil, entity.ErrShippingRateFailed
	}

	var shippingCost float64
	foundService := false
	for _, r := range rates {
		if strings.EqualFold(r.CourierServiceCode, input.CourierService) {
			shippingCost = float64(r.Price)
			foundService = true
			break
		}
	}
	if !foundService {
		shippingCost = float64(rates[0].Price)
	}

	// 4. Validate and calculate discount voucher if provided
	var discountAmount float64
	var discountID *uint
	if input.DiscountCode != nil && strings.TrimSpace(*input.DiscountCode) != "" {
		code := strings.TrimSpace(*input.DiscountCode)
		discResult, err := u.discountUsecase.ValidateAndCalculate(ctx, code, discountCartItems)
		if err != nil {
			return nil, err
		}
		discountAmount = discResult.DiscountAmount

		discEntity, err := u.discountRepo.FindByCode(ctx, code)
		if err == nil && discEntity != nil {
			discountID = &discEntity.ID
		}
	}

	// 5. Calculate administrative fees
	adminFee := 0.0
	switch strings.ToLower(input.PaymentMethod) {
	case "bca_va", "bni_va", "bri_va", "permata_va", "cimb_va":
		adminFee = 5000.0 // Flat rate Virtual Account
	case "mandiri_bill", "echannel":
		adminFee = 4000.0 // Mandiri Bill
	case "gopay", "qris", "shopeepay":
		adminFee = math.Round(itemsSubtotal * 0.007)
	case "credit_card":
		adminFee = math.Round((itemsSubtotal * 0.029) + 2000.0)
	case "alfamart", "indomaret":
		adminFee = 5000.0 // Flat rate Convenience Store
	default:
		adminFee = 0.0
	}

	totalPayment := itemsSubtotal + shippingCost + adminFee - discountAmount
	if totalPayment < 0 {
		totalPayment = 0
	}

	return &PreviewCheckoutResult{
		Address:          address,
		Items:            orderItems,
		ItemsSubtotal:    itemsSubtotal,
		TotalWeightGrams: totalWeightGrams,
		Courier:          input.Courier,
		CourierService:   input.CourierService,
		ShippingCost:     shippingCost,
		DiscountCode:     input.DiscountCode,
		DiscountAmount:   discountAmount,
		DiscountID:       discountID,
		PaymentMethod:    input.PaymentMethod,
		AdminFee:         adminFee,
		TotalPayment:     totalPayment,
	}, nil
}

func (u *checkoutUsecase) CreateOrder(ctx context.Context, userID uint, input CreateOrderInput) (*CreateOrderResult, error) {
	preview, err := u.PreviewCheckout(ctx, userID, PreviewCheckoutInput(input))
	if err != nil {
		return nil, err
	}

	invoiceNumber := fmt.Sprintf("INV-%s-%06d", time.Now().Format("20060102"), time.Now().UnixNano()%1000000)
	deadline := time.Now().Add(24 * time.Hour)

	order := &entity.Order{
		UserID:            userID,
		InvoiceNumber:     invoiceNumber,
		Status:            entity.OrderStatusWaitingPayment,
		ItemsSubtotal:     preview.ItemsSubtotal,
		ShippingCost:      preview.ShippingCost,
		DiscountAmount:    preview.DiscountAmount,
		DiscountID:        preview.DiscountID,
		AdminFee:          preview.AdminFee,
		TotalAmount:       preview.TotalPayment,
		DiscountCode:      input.DiscountCode,
		Courier:           input.Courier,
		CourierService:    input.CourierService,
		TotalWeightGrams:  preview.TotalWeightGrams,
		RecipientName:     preview.Address.ReceiverName,
		RecipientPhone:    preview.Address.Phone,
		RecipientAddress:  preview.Address.AddressLine,
		ProvinceName:      preview.Address.ProvinceName,
		CityName:          preview.Address.CityName,
		DistrictName:      preview.Address.DistrictName,
		SubdistrictName:   preview.Address.SubdistrictName,
		PostalCode:        preview.Address.PostalCode,
		DestinationAreaID: preview.Address.BiteshipAreaID,
		PaymentDeadline:   deadline,
	}

	// Atomic Database Transaction: Lock Stock -> Deduct Stock -> Save Order -> Increment Voucher -> Clear Cart
	err = u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCheckoutRepo := u.checkoutRepo.WithTx(tx)
		txOrderRepo := u.orderRepo.WithTx(tx)

		bookIDs := make([]uint, len(preview.Items))
		for i, item := range preview.Items {
			bookIDs[i] = item.BookID
		}

		bookMap, err := txCheckoutRepo.LockAndValidateBooks(ctx, bookIDs)
		if err != nil {
			return err
		}

		for _, item := range preview.Items {
			book, ok := bookMap[item.BookID]
			if !ok || book.Stock < item.Quantity {
				return entity.ErrInsufficientStock
			}
			if err := txCheckoutRepo.DeductBookStock(ctx, item.BookID, item.Quantity); err != nil {
				return err
			}
		}

		if err := txOrderRepo.CreateOrder(ctx, order, preview.Items); err != nil {
			return err
		}

		if preview.DiscountID != nil && *preview.DiscountID > 0 {
			if err := u.discountRepo.IncrementUsedCount(ctx, *preview.DiscountID); err != nil {
				return err
			}
		}

		return txCheckoutRepo.DeleteCartItems(ctx, input.CartItemIDs)
	})
	if err != nil {
		return nil, err
	}

	// External payment gateway charge
	chargeRes, err := u.midtransSvc.Charge(ctx, invoiceNumber, int64(order.TotalAmount), input.PaymentMethod)
	if err != nil {
		return nil, entity.ErrMidtransChargeFailed
	}

	bank := nilIfEmpty(chargeRes.Bank)
	vaNumber := nilIfEmpty(chargeRes.VANumber)
	billKey := nilIfEmpty(chargeRes.BillKey)
	billerCode := nilIfEmpty(chargeRes.BillerCode)
	qrURL := nilIfEmpty(chargeRes.QRURL)

	paymentRecord := &entity.Payment{
		OrderID:               order.ID,
		MidtransTransactionID: &chargeRes.TransactionID,
		PaymentType:           chargeRes.PaymentType,
		Bank:                  bank,
		VANumber:              vaNumber,
		BillKey:               billKey,
		BillerCode:            billerCode,
		QRURL:                 qrURL,
		Status:                entity.PaymentStatusPending,
	}
	_ = u.orderRepo.CreatePayment(ctx, paymentRecord)

	result := &CreateOrderResult{
		InvoiceNumber:   invoiceNumber,
		Status:          string(order.Status),
		TotalAmount:     order.TotalAmount,
		PaymentDeadline: deadline,
	}
	result.PaymentInfo.PaymentType = chargeRes.PaymentType
	result.PaymentInfo.Bank = bank
	result.PaymentInfo.VANumber = vaNumber
	result.PaymentInfo.BillKey = billKey
	result.PaymentInfo.BillerCode = billerCode
	result.PaymentInfo.QRURL = qrURL

	return result, nil
}

func (u *checkoutUsecase) HandleMidtransWebhook(ctx context.Context, input MidtransWebhookInput) error {
	isValid := u.midtransSvc.VerifyNotification(input.OrderID, input.StatusCode, input.GrossAmount, input.SignatureKey)
	if !isValid {
		return entity.ErrInvalidWebhookSignature
	}

	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txOrderRepo := u.orderRepo.WithTx(tx)
		txCheckoutRepo := u.checkoutRepo.WithTx(tx)

		order, err := txOrderRepo.FindByInvoice(ctx, input.OrderID)
		if err != nil {
			return err
		}

		if order.Status == entity.OrderStatusShipped ||
			order.Status == entity.OrderStatusCompleted ||
			order.Status == entity.OrderStatusReturned {
			return nil
		}

		now := time.Now()
		var targetOrderStatus entity.OrderStatus
		var targetPaymentStatus entity.PaymentStatus

		switch strings.ToLower(input.TransactionStatus) {
		case "capture":
			if strings.ToLower(input.FraudStatus) == "accept" {
				targetPaymentStatus = entity.PaymentStatusSettlement
				targetOrderStatus = entity.OrderStatusPaid
				order.PaidAt = &now
			} else {
				targetPaymentStatus = entity.PaymentStatusDeny
				targetOrderStatus = entity.OrderStatusCancelled
			}
		case "settlement":
			targetPaymentStatus = entity.PaymentStatusSettlement
			targetOrderStatus = entity.OrderStatusPaid
			order.PaidAt = &now
		case "pending":
			targetPaymentStatus = entity.PaymentStatusPending
			targetOrderStatus = entity.OrderStatusWaitingPayment
		case "deny", "expire", "cancel":
			targetPaymentStatus = entity.PaymentStatus(strings.ToLower(input.TransactionStatus))
			targetOrderStatus = entity.OrderStatusCancelled
			order.CancelledAt = &now
		default:
			return nil
		}

		// Rollback inventory stock and voucher usage when order is cancelled/expired
		if targetOrderStatus == entity.OrderStatusCancelled && order.Status != entity.OrderStatusCancelled {
			for _, item := range order.Items {
				_ = txCheckoutRepo.RestoreBookStock(ctx, item.BookID, item.Quantity)
			}
		}

		if targetOrderStatus != "" {
			_ = txOrderRepo.UpdateOrderStatus(ctx, order.ID, targetOrderStatus)
		}
		return txOrderRepo.UpdatePaymentStatus(ctx, order.ID, targetPaymentStatus)
	})
}

func nilIfEmpty(s string) *string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
