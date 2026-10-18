package payment

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

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
	QRURL         string
	Status        string
	RawResponse   string
}

type MidtransService interface {
	Charge(ctx context.Context, invoiceNumber string, grossAmount int64, paymentMethod string) (*ChargeResponse, error)
	VerifyNotification(orderID, statusCode, grossAmount, signatureKey string) bool
}

type midtransService struct {
	client    coreapi.Client
	serverKey string
}

func NewMidtransService(serverKey string, isProduction bool) MidtransService {
	env := midtrans.Sandbox
	if isProduction {
		env = midtrans.Production
	}

	var client coreapi.Client
	client.New(serverKey, env)

	return &midtransService{
		client:    client,
		serverKey: serverKey,
	}
}

func (s *midtransService) Charge(ctx context.Context, invoiceNumber string, grossAmount int64, paymentMethod string) (*ChargeResponse, error) {
	req := &coreapi.ChargeReq{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  invoiceNumber,
			GrossAmt: grossAmount,
		},
	}

	methodLower := strings.ToLower(strings.TrimSpace(paymentMethod))

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
	case "mandiri_bill":
		req.PaymentType = coreapi.PaymentTypeEChannel
		req.EChannel = &coreapi.EChannelDetail{
			BillInfo1: "Pembayaran Toko Buku",
			BillInfo2: invoiceNumber,
		}
	case "qris", "gopay":
		req.PaymentType = coreapi.PaymentTypeQris
		req.Qris = &coreapi.QrisDetails{
			Acquirer: "gopay",
		}
	default:
		return nil, fmt.Errorf("unsupported payment method: %s", paymentMethod)
	}

	resp, apiErr := s.client.ChargeTransaction(req)
	if apiErr != nil {
		return nil, fmt.Errorf("midtrans charge failed: %s (code: %s)", apiErr.GetMessage(), apiErr.GetStatusCode())
	}

	rawJSON, _ := json.Marshal(resp)
	result := &ChargeResponse{
		TransactionID: resp.TransactionID,
		PaymentType:   resp.PaymentType,
		Status:        resp.TransactionStatus,
		RawResponse:   string(rawJSON),
	}

	// Virtual Account Bank (BCA, BNI, BRI)
	if len(resp.VaNumbers) > 0 {
		result.Bank = resp.VaNumbers[0].Bank
		result.VANumber = resp.VaNumbers[0].VANumber
	}

	//  Permata VA
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

	// QRIS
	for _, action := range resp.Actions {
		if action.Name == "generate-qr-code" {
			result.QRURL = action.URL
			break
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
