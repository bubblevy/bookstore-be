package payment_test

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
		"alfamart",
	}

	for _, method := range invalidMethods {
		t.Run("invalid method: "+method, func(t *testing.T) {
			assert.False(t, payment.IsSupportedPaymentMethod(method))
		})
	}
}
