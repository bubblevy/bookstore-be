package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/biteship"
	"github.com/bubblevy/bookstore/pkg/logger"
	"github.com/bubblevy/bookstore/pkg/payment"
	"github.com/bubblevy/bookstore/pkg/sse"
	"github.com/bubblevy/bookstore/pkg/storage"
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
	Address                 *entity.Address
	Items                   []entity.OrderItem
	ItemsSubtotal           int
	TotalWeightGrams        int
	Courier                 string
	CourierService          string
	ShippingCost            int
	DiscountCode            *string
	DiscountAmount          int
	PaymentMethod           string
	AdminFee                int
	TotalPayment            int
	DiscountID              *uint
	AvailableCouriers       []dto.CourierOptionResponse
	AvailablePaymentMethods []dto.AvailablePaymentMethodItem
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
	TotalAmount     int
	PaymentDeadline time.Time
	PaymentInfo     struct {
		PaymentMethod *string
		PaymentType   string
		Bank          *string
		VANumber      *string
		BillKey       *string
		BillerCode    *string
		PaymentCode   *string
		Store         *string
		QRURL         *string
		DeeplinkURL   *string
		Instructions  []dto.PaymentInstructionDTO
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
	GetPaymentMethods(ctx context.Context, amount int) (*dto.PaymentMethodsResponse, error)
}

type checkoutUsecase struct {
	db                *gorm.DB
	checkoutRepo      repository.CheckoutRepository
	orderRepo         repository.OrderRepository
	addressRepo       repository.AddressRepository
	discountUsecase   DiscountUsecase
	discountRepo      repository.DiscountRepository
	biteshipSvc       biteship.Client
	midtransSvc       payment.MidtransService
	paymentMethodRepo repository.PaymentMethodRepository
	storageSvc        storage.StorageService
	sseHub            sse.Hub
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
	extraDeps ...interface{},
) CheckoutUsecase {
	var pmRepo repository.PaymentMethodRepository
	var storageSvc storage.StorageService
	var sseHub sse.Hub

	for _, dep := range extraDeps {
		switch v := dep.(type) {
		case repository.PaymentMethodRepository:
			pmRepo = v
		case storage.StorageService:
			storageSvc = v
		case sse.Hub:
			sseHub = v
		}
	}

	return &checkoutUsecase{
		db:                db,
		checkoutRepo:      checkoutRepo,
		orderRepo:         orderRepo,
		addressRepo:       addressRepo,
		discountUsecase:   discountUsecase,
		discountRepo:      discountRepo,
		biteshipSvc:       biteshipSvc,
		midtransSvc:       midtransSvc,
		paymentMethodRepo: pmRepo,
		storageSvc:        storageSvc,
		sseHub:            sseHub,
	}
}

func (u *checkoutUsecase) formatFullImageURL(img string) string {
	clean := strings.TrimSpace(img)
	if clean == "" {
		return ""
	}
	if strings.HasPrefix(clean, "http://") || strings.HasPrefix(clean, "https://") {
		return clean
	}
	if u.storageSvc != nil {
		return u.storageSvc.GetURL(clean)
	}
	return "/api/v1/static/" + strings.TrimLeft(clean, "/")
}

func (u *checkoutUsecase) PreviewCheckout(ctx context.Context, userID uint, input PreviewCheckoutInput) (*PreviewCheckoutResult, error) {
	// 1. Fetch destination address snapshot (optional for preview, required for create order)
	var address *entity.Address
	var err error
	if u.addressRepo != nil {
		if input.AddressID != nil && *input.AddressID > 0 {
			address, err = u.addressRepo.FindByID(ctx, *input.AddressID)
			if err != nil || address == nil || address.UserID != userID {
				return nil, entity.ErrAddressNotFound
			}
		} else {
			address, _ = u.addressRepo.FindPrimaryByUserID(ctx, userID)
			if address != nil && address.UserID != userID {
				address = nil
			}
		}
	}

	// 2. Fetch cart items with book details
	var cartItems []entity.CartItem
	if u.checkoutRepo != nil {
		cartItems, err = u.checkoutRepo.GetCartItemsWithBooks(ctx, userID, input.CartItemIDs)
		if err != nil {
			return nil, err
		}
	}
	if len(input.CartItemIDs) > 0 {
		if len(cartItems) == 0 || len(cartItems) != len(input.CartItemIDs) {
			return nil, entity.ErrCartItemNotFound
		}
	} else if len(cartItems) == 0 {
		return nil, entity.ErrCartEmpty
	}

	var itemsSubtotal int
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

		subtotal := item.Book.Price * item.Quantity
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

		length := int(item.Book.Length)
		if length <= 0 {
			length = 20
		}
		width := int(item.Book.Width)
		if width <= 0 {
			width = 14
		}
		height := int(item.Book.Height)
		if height <= 0 {
			height = 2
		}

		rateItems[i] = biteship.RateItem{
			Name:     item.Book.Title,
			Value:    int64(item.Book.Price),
			Length:   length,
			Width:    width,
			Height:   height,
			Weight:   item.Book.WeightGrams,
			Quantity: item.Quantity,
		}

		discountCartItems[i] = CartItemInput{
			BookID:   item.BookID,
			Quantity: item.Quantity,
		}
	}

	// 3. Calculate shipping cost and/or fetch available couriers via Biteship
	var shippingCost int
	availableCouriers := make([]dto.CourierOptionResponse, 0)

	if address != nil && address.BiteshipAreaID != "" && u.biteshipSvc != nil {
		couriersToQuery := "jne,sicepat,jnt,anteraja,tiki,pos"
		rates, err := u.biteshipSvc.CalculateRatesByAreaID(ctx, address.BiteshipAreaID, couriersToQuery, rateItems)
		if err != nil {
			logger.Error("failed to calculate biteship rates for checkout preview", "dest_area_id", address.BiteshipAreaID, "error", err)
		} else if len(rates) > 0 {
			availableCouriers = make([]dto.CourierOptionResponse, 0, len(rates))
			for _, r := range rates {
				etd := strings.TrimSpace(r.Duration)
				if etd == "" {
					etd = strings.TrimSpace(r.ShipmentDurationRange)
				}
				if etd == "" {
					etd = "-"
				}
				availableCouriers = append(availableCouriers, dto.CourierOptionResponse{
					CourierCode:        r.CourierCode,
					CourierName:        r.CourierName,
					CourierServiceName: r.CourierServiceName,
					CourierServiceCode: r.CourierServiceCode,
					Description:        r.Description,
					Price:              int(r.Price),
					ETD:                etd,
					Type:               r.Type,
				})
			}
		}

		if input.Courier != "" && input.CourierService != "" {
			if err != nil || len(availableCouriers) == 0 {
				return nil, entity.ErrShippingRateFailed
			}

			foundService := false
			for _, opt := range availableCouriers {
				if strings.EqualFold(opt.CourierCode, input.Courier) && strings.EqualFold(opt.CourierServiceCode, input.CourierService) {
					shippingCost = opt.Price
					foundService = true
					break
				}
			}
			if !foundService {
				return nil, entity.ErrInvalidCourierService
			}
		}
	}

	// 4. Validate and calculate discount voucher if provided
	var discountAmount int
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

	// 5. Calculate base amount for payment & fee calculation
	baseAmount := itemsSubtotal + shippingCost - discountAmount
	if baseAmount < 0 {
		baseAmount = 0
	}

	calcFee := func(feeType string, flatAmount int, percent float64) int {
		if feeType == "percentage" {
			if baseAmount > 0 {
				return int(math.Round(float64(baseAmount) * percent))
			}
			return 0
		}
		return flatAmount
	}

	// 6. Build available active payment methods with calculated fee
	var availablePaymentMethods []dto.AvailablePaymentMethodItem
	var activeMethods []entity.PaymentMethod
	if u.paymentMethodRepo != nil {
		var err error
		activeMethods, err = u.paymentMethodRepo.FindAll(ctx, true)
		if err == nil {
			availablePaymentMethods = make([]dto.AvailablePaymentMethodItem, 0, len(activeMethods))
			for _, m := range activeMethods {
				availablePaymentMethods = append(availablePaymentMethods, dto.AvailablePaymentMethodItem{
					Code:         m.Code,
					Name:         m.Name,
					Category:     m.Category,
					Image:        u.formatFullImageURL(m.Image),
					EstimatedFee: calcFee(m.FeeType, m.FeeAmount, m.FeePercent),
					FeePercent:   m.FeePercent,
					FeeAmount:    m.FeeAmount,
				})
			}
		}
	}

	// 7. Validate specific payment method if selected
	adminFee := 0
	if input.PaymentMethod != "" {
		if !payment.IsSupportedPaymentMethod(input.PaymentMethod) {
			return nil, entity.ErrInvalidPaymentMethod
		}

		var pm *entity.PaymentMethod
		// Check from already loaded active methods first to avoid redundant DB query
		for i := range activeMethods {
			if strings.EqualFold(activeMethods[i].Code, input.PaymentMethod) {
				pm = &activeMethods[i]
				break
			}
		}

		if pm == nil && u.paymentMethodRepo != nil {
			var err error
			pm, err = u.paymentMethodRepo.FindByCode(ctx, input.PaymentMethod)
			if err != nil {
				return nil, err
			}
			if pm != nil && !pm.IsActive {
				return nil, entity.ErrPaymentMethodInactive
			}
		}

		if pm != nil {
			if pm.FeeType == "percentage" {
				adminFee = int(math.Round(float64(baseAmount) * pm.FeePercent))
			} else {
				adminFee = pm.FeeAmount
			}
		} else {
			switch strings.ToLower(input.PaymentMethod) {
			case "bca_va", "bni_va", "bri_va", "permata_va", "cimb_va":
				adminFee = 5000 // Flat rate Virtual Account
			case "mandiri_bill", "echannel":
				adminFee = 4000 // Mandiri Bill
			case "gopay", "qris", "shopeepay":
				adminFee = int(math.Round(float64(baseAmount) * 0.007))
			case "indomaret", "alfamart":
				adminFee = 5000 // Flat rate Convenience Store (Indomaret / Alfamart)
			default:
				adminFee = 0
			}
		}
	}

	totalPayment := baseAmount + adminFee

	return &PreviewCheckoutResult{
		Address:                 address,
		Items:                   orderItems,
		ItemsSubtotal:           itemsSubtotal,
		TotalWeightGrams:        totalWeightGrams,
		Courier:                 input.Courier,
		CourierService:          input.CourierService,
		ShippingCost:            shippingCost,
		DiscountCode:            input.DiscountCode,
		DiscountAmount:          discountAmount,
		DiscountID:              discountID,
		PaymentMethod:           input.PaymentMethod,
		AdminFee:                adminFee,
		TotalPayment:            totalPayment,
		AvailableCouriers:       availableCouriers,
		AvailablePaymentMethods: availablePaymentMethods,
	}, nil
}

func (u *checkoutUsecase) CreateOrder(ctx context.Context, userID uint, input CreateOrderInput) (*CreateOrderResult, error) {
	if !payment.IsSupportedPaymentMethod(input.PaymentMethod) {
		return nil, entity.ErrInvalidPaymentMethod
	}

	preview, err := u.PreviewCheckout(ctx, userID, PreviewCheckoutInput(input))
	if err != nil {
		return nil, err
	}

	if preview.Address == nil {
		return nil, entity.ErrAddressNotFound
	}
	if input.Courier == "" || input.CourierService == "" {
		return nil, errors.New("courier and courier service are required")
	}

	invoiceNumber := fmt.Sprintf("INV-%s-%06d", time.Now().Format("20060102"), time.Now().UnixNano()%1000000)
	// Default payment deadline is 3 hours for all payment methods
	deadline := time.Now().Add(3 * time.Hour)

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

	// Atomic Database Transaction: Lock Stock -> Deduct Stock -> Save Order -> Increment Voucher
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
				logger.Warn("create order failed - insufficient stock", "user_id", userID, "book_id", item.BookID)
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

		return nil
	})
	if err != nil {
		logger.Error("create order failed - transaction error", "user_id", userID, "error", err)
		return nil, err
	}

	// External payment gateway charge
	chargeRes, err := u.midtransSvc.Charge(ctx, invoiceNumber, int64(order.TotalAmount), input.PaymentMethod)
	if err != nil {
		logger.Error("create order failed - midtrans charge error", "invoice", invoiceNumber, "error", err)
		// Rollback order and restore book stock so user can retry without stock being held or cart being deleted
		_ = u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			txCheckoutRepo := u.checkoutRepo.WithTx(tx)
			txOrderRepo := u.orderRepo.WithTx(tx)
			_ = txOrderRepo.UpdateOrderStatus(ctx, order.ID, entity.OrderStatusCancelled)
			for _, item := range preview.Items {
				_ = txCheckoutRepo.RestoreBookStock(ctx, item.BookID, item.Quantity)
			}
			return nil
		})
		return nil, entity.ErrMidtransChargeFailed
	}

	// Delete cart items ONLY AFTER payment charge has successfully been created
	_ = u.checkoutRepo.DeleteCartItems(ctx, input.CartItemIDs)

	bank := nilIfEmpty(chargeRes.Bank)
	vaNumber := nilIfEmpty(chargeRes.VANumber)
	billKey := nilIfEmpty(chargeRes.BillKey)
	billerCode := nilIfEmpty(chargeRes.BillerCode)
	paymentCode := nilIfEmpty(chargeRes.PaymentCode)
	store := nilIfEmpty(chargeRes.Store)
	qrURL := nilIfEmpty(chargeRes.QRURL)
	deeplinkURL := nilIfEmpty(chargeRes.DeeplinkURL)

	var rawResp *string
	if chargeRes.RawResponse != "" && json.Valid([]byte(chargeRes.RawResponse)) {
		rawResp = &chargeRes.RawResponse
	}

	pmCode := strings.ToLower(input.PaymentMethod)
	paymentRecord := &entity.Payment{
		OrderID:               order.ID,
		PaymentMethodCode:     &pmCode,
		MidtransTransactionID: &chargeRes.TransactionID,
		PaymentType:           chargeRes.PaymentType,
		Bank:                  bank,
		VANumber:              vaNumber,
		BillKey:               billKey,
		BillerCode:            billerCode,
		PaymentCode:           paymentCode,
		Store:                 store,
		QRURL:                 qrURL,
		DeeplinkURL:           deeplinkURL,
		Status:                entity.PaymentStatusPending,
		RawResponse:           rawResp,
	}
	_ = u.orderRepo.CreatePayment(ctx, paymentRecord)

	// Synchronize deadline from payment gateway response if provided
	if chargeRes.ExpiryTime != nil {
		deadline = *chargeRes.ExpiryTime
		order.PaymentDeadline = deadline
		_ = u.orderRepo.UpdateOrderPaymentDeadline(ctx, order.ID, deadline)
	}

	logger.Info("order created successfully",
		"invoice", invoiceNumber,
		"user_id", userID,
		"total_amount", order.TotalAmount,
		"payment_method", input.PaymentMethod,
		"courier", input.Courier,
	)

	result := &CreateOrderResult{
		InvoiceNumber:   invoiceNumber,
		Status:          string(order.Status),
		TotalAmount:     order.TotalAmount,
		PaymentDeadline: deadline,
	}

	result.PaymentInfo.PaymentMethod = &input.PaymentMethod
	result.PaymentInfo.PaymentType = chargeRes.PaymentType
	result.PaymentInfo.Bank = bank
	result.PaymentInfo.VANumber = vaNumber
	result.PaymentInfo.BillKey = billKey
	result.PaymentInfo.BillerCode = billerCode
	result.PaymentInfo.PaymentCode = paymentCode
	result.PaymentInfo.Store = store
	result.PaymentInfo.QRURL = qrURL
	result.PaymentInfo.DeeplinkURL = deeplinkURL

	var instructions []dto.PaymentInstructionDTO
	if u.paymentMethodRepo != nil {
		if pm, err := u.paymentMethodRepo.FindByCode(ctx, input.PaymentMethod); err == nil && pm != nil {
			if len(pm.Instructions) > 0 {
				instructions = make([]dto.PaymentInstructionDTO, len(pm.Instructions))
				for i, inst := range pm.Instructions {
					instructions[i] = dto.PaymentInstructionDTO{
						Title:   inst.Title,
						Content: inst.Content,
					}
				}
			}
		}
	}
	result.PaymentInfo.Instructions = instructions

	return result, nil
}

func (u *checkoutUsecase) HandleMidtransWebhook(ctx context.Context, input MidtransWebhookInput) error {
	isValid := u.midtransSvc.VerifyNotification(input.OrderID, input.StatusCode, input.GrossAmount, input.SignatureKey)
	if !isValid {
		logger.Warn("midtrans webhook invalid signature", "order_id", input.OrderID, "status_code", input.StatusCode)
		return entity.ErrInvalidWebhookSignature
	}

	logger.Info("midtrans webhook received",
		"order_id", input.OrderID,
		"transaction_status", input.TransactionStatus,
		"fraud_status", input.FraudStatus,
		"gross_amount", input.GrossAmount,
	)

	var paymentEvent *sse.PaymentStatusEvent
	err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txOrderRepo := u.orderRepo.WithTx(tx)
		txCheckoutRepo := u.checkoutRepo.WithTx(tx)

		order, err := txOrderRepo.FindByInvoice(ctx, input.OrderID)
		if err != nil {
			logger.Error("midtrans webhook failed - order not found", "order_id", input.OrderID, "error", err)
			return err
		}

		if order.Status == entity.OrderStatusShipped ||
			order.Status == entity.OrderStatusCompleted ||
			order.Status == entity.OrderStatusReturned {
			logger.Info("midtrans webhook ignored - order already finished", "order_id", input.OrderID, "status", order.Status)
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
			logger.Info("midtrans webhook cancelling order - rolling back inventory stock", "order_id", order.InvoiceNumber)
			for _, item := range order.Items {
				_ = txCheckoutRepo.RestoreBookStock(ctx, item.BookID, item.Quantity)
			}
		}

		var rawResp *string
		if len(input.RawJSON) > 0 && json.Valid(input.RawJSON) {
			s := string(input.RawJSON)
			rawResp = &s
		}

		if input.TransactionTime != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", input.TransactionTime); err == nil {
				if targetOrderStatus == entity.OrderStatusPaid {
					order.PaidAt = &t
				}
			}
		}

		if targetOrderStatus != "" {
			_ = txOrderRepo.UpdateOrderStatus(ctx, order.ID, targetOrderStatus)
		}
		logger.Info("midtrans webhook processed successfully",
			"order_id", order.InvoiceNumber,
			"order_status", targetOrderStatus,
			"payment_status", targetPaymentStatus,
		)

		if rawResp != nil {
			if err := txOrderRepo.UpdatePaymentDetails(ctx, order.ID, targetPaymentStatus, rawResp, order.PaidAt); err != nil {
				return err
			}
		} else {
			if err := txOrderRepo.UpdatePaymentStatus(ctx, order.ID, targetPaymentStatus); err != nil {
				return err
			}
		}

		paymentEvent = &sse.PaymentStatusEvent{
			OrderID:       order.ID,
			InvoiceNumber: order.InvoiceNumber,
			UserID:        order.UserID,
			PaymentStatus: string(targetPaymentStatus),
			OrderStatus:   string(targetOrderStatus),
			GrossAmount:   input.GrossAmount,
			PaidAt:        order.PaidAt,
			UpdatedAt:     now,
		}

		return nil
	})

	if err == nil && u.sseHub != nil && paymentEvent != nil {
		u.sseHub.PublishPaymentStatus(*paymentEvent)
	}

	return err
}

func nilIfEmpty(s string) *string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func (u *checkoutUsecase) GetPaymentMethods(ctx context.Context, amount int) (*dto.PaymentMethodsResponse, error) {
	if amount < 0 {
		amount = 0
	}

	calcFee := func(feeType string, flatAmount int, percent float64) int {
		if feeType == "percentage" {
			if amount > 0 {
				return int(math.Round(float64(amount) * percent))
			}
			return 0
		}
		return flatAmount
	}

	if u.paymentMethodRepo != nil {
		methods, err := u.paymentMethodRepo.FindAll(ctx, false)
		if err == nil && len(methods) > 0 {
			categoryOrder := []string{"virtual_account", "ewallet", "convenience_store"}
			categoryLabels := map[string]string{
				"virtual_account":   "Virtual Account (Transfer Bank)",
				"ewallet":           "E-Wallet & QRIS",
				"convenience_store": "Gerai Retail (Convenience Store)",
			}

			groupsMap := make(map[string][]dto.PaymentMethodItemResponse)
			for _, m := range methods {
				item := dto.PaymentMethodItemResponse{
					ID:           m.ID,
					Code:         m.Code,
					Name:         m.Name,
					Category:     m.Category,
					FeeType:      m.FeeType,
					FeeAmount:    m.FeeAmount,
					FeePercent:   m.FeePercent,
					EstimatedFee: calcFee(m.FeeType, m.FeeAmount, m.FeePercent),
					Description:  m.Description,
					Image:        u.formatFullImageURL(m.Image),
					IsActive:     m.IsActive,
				}

				groupsMap[m.Category] = append(groupsMap[m.Category], item)
			}

			var groups []dto.PaymentMethodGroupResponse
			seenCategories := make(map[string]bool)

			for _, cat := range categoryOrder {
				items, exists := groupsMap[cat]
				if exists && len(items) > 0 {
					groups = append(groups, dto.PaymentMethodGroupResponse{
						Category: cat,
						Name:     categoryLabels[cat],
						Methods:  items,
					})
					seenCategories[cat] = true
				}
			}

			for cat, items := range groupsMap {
				if !seenCategories[cat] && len(items) > 0 {
					label, exists := categoryLabels[cat]
					if !exists {
						label = strings.Title(strings.ReplaceAll(cat, "_", " "))
					}
					groups = append(groups, dto.PaymentMethodGroupResponse{
						Category: cat,
						Name:     label,
						Methods:  items,
					})
				}
			}

			return &dto.PaymentMethodsResponse{
				Groups: groups,
			}, nil
		}
	}

	return &dto.PaymentMethodsResponse{
		Groups: []dto.PaymentMethodGroupResponse{},
	}, nil
}

