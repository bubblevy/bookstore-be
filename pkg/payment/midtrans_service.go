package payment

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
)

type ChargeResponse struct {
	TransactionID string
	PaymentType   string
	Bank          string
	VANumber      string
	BillKey       string
	BillerCode    string
	PaymentCode   string
	Store         string
	QRURL         string
	DeeplinkURL   string
	Status        string
	ExpiryTime    *time.Time
	RawResponse   string
}

type MidtransService interface {
	Charge(ctx context.Context, invoiceNumber string, grossAmount int64, paymentMethod string) (*ChargeResponse, error)
	VerifyNotification(orderID, statusCode, grossAmount, signatureKey string) bool
}

type midtransService struct {
	client      coreapi.Client
	serverKey   string
	frontendURL string
}

func NewMidtransService(serverKey string, isProduction bool, frontendURL string) MidtransService {
	env := midtrans.Sandbox
	if isProduction {
		env = midtrans.Production
	}

	var client coreapi.Client
	client.New(serverKey, env)

	cleanFrontendURL := strings.TrimRight(strings.TrimSpace(frontendURL), "/")
	if cleanFrontendURL == "" {
		cleanFrontendURL = strings.TrimRight(strings.TrimSpace(os.Getenv("FRONTEND_URL")), "/")
	}
	if cleanFrontendURL == "" {
		cleanFrontendURL = "http://localhost:5173"
	}

	return &midtransService{
		client:      client,
		serverKey:   serverKey,
		frontendURL: cleanFrontendURL,
	}
}

var SupportedPaymentMethods = map[string]bool{
	"bca_va":       true,
	"bni_va":       true,
	"bri_va":       true,
	"permata_va":   true,
	"cimb_va":      true,
	"mandiri_bill": true,
	"echannel":     true,
	"qris":         true,
	"gopay":        true,
	"shopeepay":    true,
	"indomaret":    true,
	"alfamart":     true,
}

func IsSupportedPaymentMethod(paymentMethod string) bool {
	return SupportedPaymentMethods[strings.ToLower(strings.TrimSpace(paymentMethod))]
}

func (s *midtransService) Charge(ctx context.Context, invoiceNumber string, grossAmount int64, paymentMethod string) (*ChargeResponse, error) {
	req := &coreapi.ChargeReq{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  invoiceNumber,
			GrossAmt: grossAmount,
		},
		CustomExpiry: &coreapi.CustomExpiry{
			ExpiryDuration: 3,
			Unit:           "hour",
		},
	}

	methodLower := strings.ToLower(strings.TrimSpace(paymentMethod))

	callbackURL := fmt.Sprintf("%s/orders/%s", s.frontendURL, invoiceNumber)
	if customCallback := os.Getenv("MIDTRANS_CALLBACK_URL"); customCallback != "" {
		if strings.Contains(customCallback, "%s") {
			callbackURL = fmt.Sprintf(customCallback, invoiceNumber)
		} else {
			callbackURL = strings.TrimRight(customCallback, "/") + "/" + invoiceNumber
		}
	}

	switch methodLower {
	case "bca_va":
		req.PaymentType = coreapi.PaymentTypeBankTransfer
		req.BankTransfer = &coreapi.BankTransferDetails{
			Bank: midtrans.BankBca,
		}
	case "bni_va":
		req.PaymentType = coreapi.PaymentTypeBankTransfer
		req.BankTransfer = &coreapi.BankTransferDetails{
			Bank: midtrans.BankBni,
		}
	case "bri_va":
		req.PaymentType = coreapi.PaymentTypeBankTransfer
		req.BankTransfer = &coreapi.BankTransferDetails{
			Bank: midtrans.BankBri,
		}
	case "permata_va":
		req.PaymentType = coreapi.PaymentTypeBankTransfer
		req.BankTransfer = &coreapi.BankTransferDetails{
			Bank: midtrans.BankPermata,
		}
	case "cimb_va":
		req.PaymentType = coreapi.PaymentTypeBankTransfer
		req.BankTransfer = &coreapi.BankTransferDetails{
			Bank: midtrans.BankCimb,
		}
	case "mandiri_bill", "echannel":
		req.PaymentType = coreapi.PaymentTypeEChannel
		req.EChannel = &coreapi.EChannelDetail{
			BillInfo1: "Pembayaran Toko Buku",
			BillInfo2: invoiceNumber,
		}
	case "qris":
		req.PaymentType = coreapi.PaymentTypeQris
		req.Qris = &coreapi.QrisDetails{
			Acquirer: "gopay",
		}
	case "gopay":
		req.PaymentType = coreapi.PaymentTypeGopay
		req.Gopay = &coreapi.GopayDetails{
			EnableCallback: true,
			CallbackUrl:    callbackURL,
		}
	case "shopeepay":
		req.PaymentType = coreapi.PaymentTypeShopeepay
		req.ShopeePay = &coreapi.ShopeePayDetails{
			CallbackUrl: callbackURL,
		}
	case "indomaret":
		req.PaymentType = coreapi.PaymentTypeConvenienceStore
		req.ConvStore = &coreapi.ConvStoreDetails{
			Store:   "indomaret",
			Message: "Pembayaran Toko Buku",
		}
	case "alfamart":
		req.PaymentType = coreapi.PaymentTypeConvenienceStore
		req.ConvStore = &coreapi.ConvStoreDetails{
			Store:   "alfamart",
			Message: "Pembayaran Toko Buku",
		}
	default:
		return nil, fmt.Errorf("unsupported payment method: %s", paymentMethod)
	}

	resp, apiErr := s.client.ChargeTransaction(req)
	if apiErr != nil {
		return nil, fmt.Errorf("midtrans charge failed: %s (code: %d)", apiErr.GetMessage(), apiErr.GetStatusCode())
	}

	rawJSON, _ := json.Marshal(resp)

	var expiryTime *time.Time
	if resp.ExpiryTime != "" {
		if t, err := parseMidtransExpiryTime(resp.ExpiryTime); err == nil {
			expiryTime = &t
		}
	}

	result := &ChargeResponse{
		TransactionID: resp.TransactionID,
		PaymentType:   resp.PaymentType,
		Status:        resp.TransactionStatus,
		ExpiryTime:    expiryTime,
		RawResponse:   string(rawJSON),
	}

	// Virtual Account Bank (BCA, BNI, BRI, CIMB)
	if len(resp.VaNumbers) > 0 {
		result.Bank = resp.VaNumbers[0].Bank
		result.VANumber = resp.VaNumbers[0].VANumber
	}

	// Permata VA
	if resp.PermataVaNumber != "" {
		result.Bank = "permata"
		result.VANumber = resp.PermataVaNumber
	}

	// Mandiri Bill
	if resp.BillKey != "" {
		result.Bank = "mandiri"
		result.BillKey = resp.BillKey
		result.BillerCode = resp.BillerCode
	}

	// Convenience Store (Indomaret, Alfamart)
	if resp.PaymentCode != "" {
		result.PaymentCode = resp.PaymentCode
	}
	if resp.Store != "" {
		result.Store = resp.Store
	} else if result.PaymentCode != "" {
		if strings.Contains(methodLower, "alfamart") {
			result.Store = "alfamart"
		} else {
			result.Store = "indomaret"
		}
	}

	// QRIS & E-Wallet (ShopeePay, GoPay QR / Deeplink)
	var actionDeeplink string
	var actionQRCode string
	for _, action := range resp.Actions {
		if action.Name == "deeplink-redirect" {
			actionDeeplink = action.URL
		} else if action.Name == "generate-qr-code" {
			actionQRCode = action.URL
		}
	}

	if methodLower == "qris" {
		// QRIS: hanya simpan satu di QRURL
		if actionQRCode != "" {
			result.QRURL = actionQRCode
		} else if actionDeeplink != "" {
			result.QRURL = actionDeeplink
		}
	} else if methodLower == "gopay" || methodLower == "shopeepay" {
		// GoPay & ShopeePay: prioritaskan DeeplinkURL, jangan masukkan dua-duanya ke DB
		if actionDeeplink != "" {
			result.DeeplinkURL = actionDeeplink
		} else if actionQRCode != "" {
			result.QRURL = actionQRCode
		}
	} else {
		if actionDeeplink != "" {
			result.DeeplinkURL = actionDeeplink
		} else if actionQRCode != "" {
			result.QRURL = actionQRCode
		}
	}

	return result, nil
}

func (s *midtransService) VerifyNotification(orderID, statusCode, grossAmount, signatureKey string) bool {
	input := fmt.Sprintf("%s%s%s%s", orderID, statusCode, grossAmount, s.serverKey)
	hash := sha512.Sum512([]byte(input))
	expected := hex.EncodeToString(hash[:])
	return strings.EqualFold(expected, signatureKey)
}

func parseMidtransExpiryTime(expiryStr string) (time.Time, error) {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	if loc == nil {
		loc = time.FixedZone("WIB", 7*3600)
	}

	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05 MST",
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
	}

	trimmed := strings.TrimSpace(expiryStr)
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, trimmed, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse midtrans expiry time: %s", expiryStr)
}
