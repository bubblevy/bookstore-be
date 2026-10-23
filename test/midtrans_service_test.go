package test

import (
	"testing"

	"github.com/bubblevy/bookstore/pkg/payment"
	"github.com/stretchr/testify/assert"
)

func TestIsSupportedPaymentMethod(t *testing.T) {
	validMethods := []string{
		"bca_va",
		"BCA_VA",
		"bni_va",
		"bri_va",
		"permata_va",
		"cimb_va",
		"mandiri_bill",
		"echannel",
		"qris",
		"gopay",
		"shopeepay",
		"ShopeePay",
		" SHOPEEPAY ",
		"indomaret",
		"INDOMARET",
		"alfamart",
		"ALFAMART",
	}

	for _, method := range validMethods {
		t.Run("valid method: "+method, func(t *testing.T) {
			assert.True(t, payment.IsSupportedPaymentMethod(method))
		})
	}

	invalidMethods := []string{
		"",
		"   ",
		"crypto",
		"bitcoin",
		"paypal",
		"dana",
		"ovo",
		"shopee",
		"credit_card",
	}

	for _, method := range invalidMethods {
		t.Run("invalid method: "+method, func(t *testing.T) {
			assert.False(t, payment.IsSupportedPaymentMethod(method))
		})
	}
}

func TestNewMidtransService(t *testing.T) {
	svc := payment.NewMidtransService("test-server-key", false, "http://localhost:5173")
	assert.NotNil(t, svc)

	// Test notification signature verification
	// signature: SHA512(order_id + status_code + gross_amount + server_key)
	// order_id: "INV-001", status_code: "200", gross_amount: "10000.00", server_key: "test-server-key"
	validSig := svc.VerifyNotification("INV-001", "200", "10000.00", "invalid-sig")
	assert.False(t, validSig)
}
