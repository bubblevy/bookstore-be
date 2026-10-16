package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/biteship"
	"github.com/bubblevy/bookstore/pkg/payment"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// =========================================================================
// DTO & TYPES
// =========================================================================

type CheckoutItemInfo struct {
	BookID      uint    `json:"book_id"`
	Title       string  `json:"title"`
	CoverImage  *string `json:"cover_image,omitempty"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
	WeightGrams int     `json:"weight_grams"`
	Subtotal    float64 `json:"subtotal"`
}

type PreviewCheckoutInput struct {
	CartItemIDs    []uint
	AddressID      *uint
	Courier        string
	CourierService string
	DiscountCode   *string
	PaymentMethod  string
}

type PreviewCheckoutResult struct {
	Items            []CheckoutItemInfo `json:"items"`
	ItemsSubtotal    float64            `json:"items_subtotal"`
	TotalWeightGrams int                `json:"total_weight_grams"`
	Courier          string             `json:"courier"`
	CourierService   string             `json:"courier_service"`
	ShippingCost     float64            `json:"shipping_cost"`
	DiscountCode     *string            `json:"discount_code,omitempty"`
	DiscountAmount   float64            `json:"discount_amount"`
	PaymentMethod    string             `json:"payment_method"`
	AdminFee         float64            `json:"admin_fee"`
	TotalPayment     float64            `json:"total_payment"`
}

type CreateOrderInput struct {
	CartItemIDs    []uint
	AddressID      *uint
	Courier        string
	CourierService string
	DiscountCode   *string
	PaymentMethod  string
}

type PaymentInfoResult struct {
	PaymentType string  `json:"payment_type"`
	Bank        *string `json:"bank,omitempty"`
	VANumber    *string `json:"va_number,omitempty"`
	BillKey     *string `json:"bill_key,omitempty"`
	BillerCode  *string `json:"biller_code,omitempty"`
	QRURL       *string `json:"qr_url,omitempty"`
}

type CreateOrderResult struct {
	InvoiceNumber   string            `json:"invoice_number"`
	Status          string            `json:"status"`
	TotalAmount     float64           `json:"total_amount"`
	PaymentDeadline time.Time         `json:"payment_deadline"`
	PaymentInfo     PaymentInfoResult `json:"payment_info"`
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
	db           *gorm.DB
	orderRepo    repository.OrderRepository
	cartRepo     repository.CartRepository
	addressRepo  repository.AddressRepository
	discountRepo repository.DiscountRepository
	biteship     biteship.Client
	midtransSvc  payment.MidtransService
}

func NewCheckoutUsecase(
	db *gorm.DB,
	orderRepo repository.OrderRepository,
	cartRepo repository.CartRepository,
	addressRepo repository.AddressRepository,
	discountRepo repository.DiscountRepository,
	biteship biteship.Client,
	midtransSvc payment.MidtransService,
) CheckoutUsecase {
	return &checkoutUsecase{
		db:           db,
		orderRepo:    orderRepo,
		cartRepo:     cartRepo,
		addressRepo:  addressRepo,
		discountRepo: discountRepo,
		biteship:     biteship,
		midtransSvc:  midtransSvc,
	}
}

// =========================================================================
// HELPER: CALCULATE BREAKDOWN
// =========================================================================

type checkoutCalculation struct {
	Address           *entity.Address
	CartID            uint
	Books             []entity.Book
	QtyMap            map[uint]int
	Items             []CheckoutItemInfo
	ItemsSubtotal     float64
	TotalWeightGrams  int
	ShippingCost      float64
	Discount          *entity.Discount
	DiscountDeduction float64
	AdminFee          float64
	FinalTotal        float64
}

func (u *checkoutUsecase) calculateAdminFee(paymentMethod string, subtotal float64) float64 {
	switch strings.ToLower(strings.TrimSpace(paymentMethod)) {
	case "bca_va", "bni_va", "bri_va", "permata_va", "echannel":
		return 4000
	case "qris", "gopay":
		return float64(int64(subtotal * 0.007))
	default:
		return 0
	}
}

func (u *checkoutUsecase) calculateBreakdown(
	ctx context.Context,
	tx *gorm.DB,
	userID uint,
	cartItemIDs []uint,
	addressID *uint,
	courier string,
	courierService string,
	discountCode *string,
	paymentMethod string,
	lockForUpdate bool,
) (*checkoutCalculation, error) {
	if len(cartItemIDs) == 0 {
		return nil, entity.ErrCartEmpty
	}

	var addr *entity.Address
	var err error

	// 1. Validasi Alamat Penerima
	if addressID != nil && *addressID > 0 {
		addr, err = u.addressRepo.FindByID(ctx, *addressID)
		if err != nil {
			return nil, fmt.Errorf("failed fetching address: %w", err)
		}
		if addr == nil || addr.UserID != userID {
			return nil, errors.New("shipping address not found")
		}
	} else {
		// Otomatis pakai primary address jika tidak dikirim
		addr, err = u.addressRepo.FindPrimaryByUserID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed fetching primary address: %w", err)
		}
		if addr == nil {
			return nil, errors.New("primary shipping address not found, please set an address first")
		}
	}

	// 2. Ambil Keranjang User
	var cart entity.Cart
	if err := tx.Where("user_id = ?", userID).First(&cart).Error; err != nil {
		return nil, fmt.Errorf("failed finding cart: %w", err)
	}

	// 3. Ambil Item Keranjang yang Dicentang
	var cartItems []entity.CartItem
	if err := tx.Where("cart_id = ? AND id IN ?", cart.ID, cartItemIDs).Find(&cartItems).Error; err != nil {
		return nil, err
	}
	if len(cartItems) != len(cartItemIDs) {
		return nil, entity.ErrCartItemNotFound
	}

	bookIDs := make([]uint, 0, len(cartItems))
	qtyMap := make(map[uint]int, len(cartItems))
	for _, item := range cartItems {
		bookIDs = append(bookIDs, item.BookID)
		qtyMap[item.BookID] = item.Quantity
	}

	// 4. Query Buku (dengan SELECT FOR UPDATE jika create order)
	var books []entity.Book
	bookQuery := tx.Where("id IN ?", bookIDs)
	if lockForUpdate {
		bookQuery = bookQuery.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := bookQuery.Find(&books).Error; err != nil {
		return nil, err
	}

	var itemsSubtotal float64
	var totalWeight int
	itemsInfo := make([]CheckoutItemInfo, 0, len(books))
	biteshipItems := make([]biteship.RateItem, 0, len(books))

	for _, b := range books {
		if !b.IsActive {
			return nil, fmt.Errorf("book %q is no longer available", b.Title)
		}
		qty := qtyMap[b.ID]
		if qty > b.Stock {
			return nil, entity.ErrInsufficientStock
		}

		lineSubtotal := b.Price * float64(qty)
		itemsSubtotal += lineSubtotal
		totalWeight += b.WeightGrams * qty

		itemsInfo = append(itemsInfo, CheckoutItemInfo{
			BookID:      b.ID,
			Title:       b.Title,
			CoverImage:  b.CoverImage,
			Price:       b.Price,
			Quantity:    qty,
			WeightGrams: b.WeightGrams,
			Subtotal:    lineSubtotal,
		})

		biteshipItems = append(biteshipItems, biteship.RateItem{
			Name:        b.Title,
			Description: b.Title,
			Value:       int64(b.Price),
			Length:      20,
			Width:       14,
			Height:      2,
			Weight:      b.WeightGrams,
			Quantity:    qty,
		})
	}

	// 5. Cek Ongkir Realtime Biteship via Area ID
	rates, err := u.biteship.CalculateRatesByAreaID(
		ctx,
		addr.BiteshipAreaID,
		strings.ToLower(strings.TrimSpace(courier)),
		biteshipItems,
	)
	if err != nil {
		return nil, fmt.Errorf("failed calculating biteship rates: %w", err)
	}

	var shippingCost float64
	foundService := false
	for _, rate := range rates {
		if strings.EqualFold(rate.CourierServiceCode, strings.TrimSpace(courierService)) {
			shippingCost = float64(rate.Price)
			foundService = true
			break
		}
	}
	if !foundService {
		return nil, fmt.Errorf("courier service %q is not available for destination %s", courierService, addr.CityName)
	}

	// 6. Validasi Diskon / Voucher (Jika Ada)
	var discount entity.Discount
	var discountDeduction float64
	var discPtr *entity.Discount
	if discountCode != nil && strings.TrimSpace(*discountCode) != "" {
		codeUpper := strings.ToUpper(strings.TrimSpace(*discountCode))
		if err := tx.Preload("Books").Preload("Tiers").Where("code = ?", codeUpper).First(&discount).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, entity.ErrDiscountNotFound
			}
			return nil, err
		}

		if err := discount.IsValidNow(); err != nil {
			return nil, err
		}

		var eligibleSubtotal float64
		eligibleBookMap := make(map[uint]bool)
		for _, b := range discount.Books {
			eligibleBookMap[b.ID] = true
		}
		for _, it := range itemsInfo {
			if discount.GetIsAllBooks() || eligibleBookMap[it.BookID] {
				eligibleSubtotal += it.Subtotal
			}
		}

		if eligibleSubtotal == 0 {
			return nil, entity.ErrDiscountNotApplicable
		}

		cut, err := discount.CalculateDiscount(eligibleSubtotal)
		if err != nil {
			return nil, err
		}
		discountDeduction = cut
		discPtr = &discount
	}

	// 7. Hitung Biaya Admin Payment Gateway & Total Akhir
	adminFee := u.calculateAdminFee(paymentMethod, itemsSubtotal)
	finalTotal := itemsSubtotal + shippingCost + adminFee - discountDeduction
	if finalTotal < 0 {
		finalTotal = 0
	}

	return &checkoutCalculation{
		Address:           addr,
		CartID:            cart.ID,
		Books:             books,
		QtyMap:            qtyMap,
		Items:             itemsInfo,
		ItemsSubtotal:     itemsSubtotal,
		TotalWeightGrams:  totalWeight,
		ShippingCost:      shippingCost,
		Discount:          discPtr,
		DiscountDeduction: discountDeduction,
		AdminFee:          adminFee,
		FinalTotal:        finalTotal,
	}, nil
}

func (u *checkoutUsecase) PreviewCheckout(ctx context.Context, userID uint, input PreviewCheckoutInput) (*PreviewCheckoutResult, error) {
	calc, err := u.calculateBreakdown(
		ctx,
		u.db.WithContext(ctx),
		userID,
		input.CartItemIDs,
		input.AddressID,
		input.Courier,
		input.CourierService,
		input.DiscountCode,
		input.PaymentMethod,
		false, // Read-only
	)
	if err != nil {
		return nil, err
	}

	var appliedCode *string
	if calc.Discount != nil {
		appliedCode = &calc.Discount.Code
	}

	return &PreviewCheckoutResult{
		Items:            calc.Items,
		ItemsSubtotal:    calc.ItemsSubtotal,
		TotalWeightGrams: calc.TotalWeightGrams,
		Courier:          input.Courier,
		CourierService:   input.CourierService,
		ShippingCost:     calc.ShippingCost,
		DiscountCode:     appliedCode,
		DiscountAmount:   calc.DiscountDeduction,
		PaymentMethod:    input.PaymentMethod,
		AdminFee:         calc.AdminFee,
		TotalPayment:     calc.FinalTotal,
	}, nil
}

func (u *checkoutUsecase) CreateOrder(ctx context.Context, userID uint, input CreateOrderInput) (*CreateOrderResult, error) {
	now := time.Now()
	invoiceNumber := fmt.Sprintf("INV-%s-%06d", now.Format("20060102"), now.UnixNano()%1000000)

	var (
		createdOrder *entity.Order
		totalGross   int64
	)

	err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		calc, err := u.calculateBreakdown(
			ctx,
			tx,
			userID,
			input.CartItemIDs,
			input.AddressID,
			input.Courier,
			input.CourierService,
			input.DiscountCode,
			input.PaymentMethod,
			true, // Kunci stok buku
		)
		if err != nil {
			return err
		}

		// Kurangi stok buku & siapkan item pesanan
		orderItems := make([]entity.OrderItem, 0, len(calc.Books))
		for _, b := range calc.Books {
			qty := calc.QtyMap[b.ID]
			if err := tx.Model(&b).Update("stock", b.Stock-qty).Error; err != nil {
				return err
			}

			subtotal := b.Price * float64(qty)
			orderItems = append(orderItems, entity.OrderItem{
				BookID:      b.ID,
				Title:       b.Title,
				CoverImage:  b.CoverImage,
				Price:       b.Price,
				WeightGrams: b.WeightGrams,
				Quantity:    qty,
				Subtotal:    subtotal,
			})
		}

		// Potong kuota voucher jika valid
		var discountID *uint
		var discountCode *string
		if calc.Discount != nil {
			discountID = &calc.Discount.ID
			discountCode = &calc.Discount.Code
			if err := tx.Model(calc.Discount).Update("used_count", calc.Discount.UsedCount+1).Error; err != nil {
				return err
			}
		}

		totalGross = int64(calc.FinalTotal)
		createdOrder = &entity.Order{
			UserID:            userID,
			InvoiceNumber:     invoiceNumber,
			Status:            entity.OrderStatusWaitingPayment,
			ItemsSubtotal:     calc.ItemsSubtotal,
			ShippingCost:      calc.ShippingCost,
			DiscountAmount:    calc.DiscountDeduction,
			AdminFee:          calc.AdminFee,
			TotalAmount:       calc.FinalTotal,
			DiscountID:        discountID,
			DiscountCode:      discountCode,
			Courier:           input.Courier,
			CourierService:    input.CourierService,
			TotalWeightGrams:  calc.TotalWeightGrams,
			RecipientName:     calc.Address.ReceiverName,
			RecipientPhone:    calc.Address.Phone,
			RecipientAddress:  calc.Address.AddressLine,
			ProvinceName:      calc.Address.ProvinceName,
			CityName:          calc.Address.CityName,
			DistrictName:      calc.Address.DistrictName,
			SubdistrictName:   calc.Address.SubdistrictName,
			DestinationAreaID: calc.Address.BiteshipAreaID,
			PostalCode:        calc.Address.PostalCode,
			PaymentDeadline:   now.Add(24 * time.Hour),
		}

		orderRepoTx := u.orderRepo.WithTx(tx)
		if err := orderRepoTx.CreateOrder(ctx, createdOrder, orderItems); err != nil {
			return err
		}

		// Hapus item yang dibeli dari keranjang
		return tx.Where("cart_id = ? AND id IN ?", calc.CartID, input.CartItemIDs).Delete(&entity.CartItem{}).Error
	})
	if err != nil {
		return nil, err
	}

	// Charge ke Midtrans Core API
	chargeRes, err := u.midtransSvc.Charge(ctx, invoiceNumber, totalGross, input.PaymentMethod)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", entity.ErrMidtransChargeFailed, err.Error())
	}

	paymentRecord := &entity.Payment{
		OrderID:               createdOrder.ID,
		MidtransTransactionID: &chargeRes.TransactionID,
		PaymentType:           chargeRes.PaymentType,
		Status:                entity.PaymentStatusPending,
	}
	if chargeRes.Bank != "" {
		paymentRecord.Bank = &chargeRes.Bank
	}
	if chargeRes.VANumber != "" {
		paymentRecord.VANumber = &chargeRes.VANumber
	}
	if chargeRes.BillKey != "" {
		paymentRecord.BillKey = &chargeRes.BillKey
		paymentRecord.BillerCode = &chargeRes.BillerCode
	}
	if chargeRes.QRURL != "" {
		paymentRecord.QRURL = &chargeRes.QRURL
	}

	if err := u.orderRepo.CreatePayment(ctx, paymentRecord); err != nil {
		return nil, fmt.Errorf("failed saving payment record: %w", err)
	}

	return &CreateOrderResult{
		InvoiceNumber:   createdOrder.InvoiceNumber,
		Status:          string(createdOrder.Status),
		TotalAmount:     createdOrder.TotalAmount,
		PaymentDeadline: createdOrder.PaymentDeadline,
		PaymentInfo: PaymentInfoResult{
			PaymentType: paymentRecord.PaymentType,
			Bank:        paymentRecord.Bank,
			VANumber:    paymentRecord.VANumber,
			BillKey:     paymentRecord.BillKey,
			BillerCode:  paymentRecord.BillerCode,
			QRURL:       paymentRecord.QRURL,
		},
	}, nil
}

func (u *checkoutUsecase) HandleMidtransWebhook(ctx context.Context, input MidtransWebhookInput) error {
	isValid := u.midtransSvc.VerifyNotification(
		input.OrderID,
		input.StatusCode,
		input.GrossAmount,
		input.SignatureKey,
	)
	if !isValid {
		return errors.New("invalid midtrans webhook signature")
	}

	rawStr := string(input.RawJSON)
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order entity.Order
		if err := tx.Preload("Items").Where("invoice_number = ?", input.OrderID).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return entity.ErrOrderNotFound
			}
			return err
		}

		if order.Status == entity.OrderStatusShipped || order.Status == entity.OrderStatusCompleted || order.Status == entity.OrderStatusReturned {
			return nil
		}

		now := time.Now()
		var targetOrderStatus entity.OrderStatus
		var targetPaymentStatus entity.PaymentStatus

		switch strings.ToLower(input.TransactionStatus) {
		case "capture":
			switch strings.ToLower(input.FraudStatus) {
			case "challenge":
				targetPaymentStatus = entity.PaymentStatusPending
				targetOrderStatus = entity.OrderStatusWaitingPayment
			case "accept":
				targetPaymentStatus = entity.PaymentStatusSettlement
				targetOrderStatus = entity.OrderStatusPaid
				order.PaidAt = &now
			default:
				targetPaymentStatus = entity.PaymentStatusDeny
				targetOrderStatus = entity.OrderStatusCancelled
				reason := fmt.Sprintf("Fraud check failed: %s", input.FraudStatus)
				order.CancelledAt = &now
				order.CancellationReason = &reason
			}
		case "settlement":
			targetPaymentStatus = entity.PaymentStatusSettlement
			targetOrderStatus = entity.OrderStatusPaid
			order.PaidAt = &now
		case "pending":
			targetPaymentStatus = entity.PaymentStatusPending
			targetOrderStatus = entity.OrderStatusWaitingPayment
		case "deny":
			targetPaymentStatus = entity.PaymentStatusDeny
			targetOrderStatus = entity.OrderStatusCancelled
			reason := "Payment denied by provider"
			order.CancelledAt = &now
			order.CancellationReason = &reason
		case "expire":
			targetPaymentStatus = entity.PaymentStatusExpire
			targetOrderStatus = entity.OrderStatusCancelled
			reason := "Payment expired"
			order.CancelledAt = &now
			order.CancellationReason = &reason
		case "cancel":
			targetPaymentStatus = entity.PaymentStatusCancel
			targetOrderStatus = entity.OrderStatusCancelled
			reason := "Payment cancelled by user"
			order.CancelledAt = &now
			order.CancellationReason = &reason
		case "refund":
			targetPaymentStatus = entity.PaymentStatusRefund
			if order.Status == entity.OrderStatusPaid || order.Status == entity.OrderStatusProcessing {
				targetOrderStatus = entity.OrderStatusCancelled
				reason := "Payment refunded to customer"
				order.CancelledAt = &now
				order.CancellationReason = &reason
			}
		default:
			return nil
		}

		// Rollback stok bila pesanan batal/kedaluwarsa
		if targetOrderStatus == entity.OrderStatusCancelled {
			for _, item := range order.Items {
				if err := tx.Model(&entity.Book{}).
					Where("id = ?", item.BookID).
					Update("stock", gorm.Expr("stock + ?", item.Quantity)).Error; err != nil {
					return err
				}
			}
		}

		// Perbarui data order tanpa memicu update asosiasi child Items
		orderUpdates := map[string]interface{}{
			"status":     targetOrderStatus,
			"updated_at": now,
		}
		if order.PaidAt != nil {
			orderUpdates["paid_at"] = order.PaidAt
		}
		if order.CancelledAt != nil {
			orderUpdates["cancelled_at"] = order.CancelledAt
			orderUpdates["cancellation_reason"] = order.CancellationReason
		}

		if err := tx.Model(&entity.Order{}).Where("id = ?", order.ID).Updates(orderUpdates).Error; err != nil {
			return err
		}

		// Perbarui data tabel payment
		updates := map[string]interface{}{
			"status":       targetPaymentStatus,
			"raw_response": rawStr,
			"updated_at":   now,
		}
		if targetPaymentStatus == entity.PaymentStatusSettlement {
			updates["paid_at"] = &now
		}

		return tx.Model(&entity.Payment{}).Where("order_id = ?", order.ID).Updates(updates).Error
	})
}
