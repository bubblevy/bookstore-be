package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/go-pdf/fpdf"
)

// GenerateInvoicePDF creates an authentic, clean professional invoice receipt PDF
// matching the structured accounting layout with clear itemized tables, balanced padding,
// financial breakdowns, and standard 3-column page footers.
func GenerateInvoicePDF(order *entity.Order) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 18)

	// Clean 3-column accounting footer with subtle background watermark
	pdf.SetFooterFunc(func() {
		// Draw watermark on each page (rendered with subtle 5% alpha)
		watermarkSub := "FAKTUR RESMI"
		if order.Status == entity.OrderStatusPaid || order.Status == entity.OrderStatusCompleted {
			watermarkSub = "LUNAS // PAID"
		} else if order.Status == entity.OrderStatusWaitingPayment {
			watermarkSub = "MENUNGGU PEMBAYARAN"
		} else if order.Status == entity.OrderStatusCancelled || order.Status == entity.OrderStatusReturned {
			watermarkSub = "DIBATALKAN // VOID"
		}
		drawWatermark(pdf, "BOOKSTORE", watermarkSub)

		pdf.SetY(-14)
		pdf.SetDrawColor(209, 213, 219) // #d1d5db light gray border
		pdf.SetLineWidth(0.2)
		pdf.SetFillColor(255, 255, 255)
		pdf.SetTextColor(75, 85, 99) // #4b5563
		pdf.SetFont("Arial", "", 7.5)

		nowStr := time.Now().Format("02/01/2006 15:04")
		pdf.CellFormat(62, 6, " "+nowStr, "1", 0, "L", true, 0, "")
		pageStr := fmt.Sprintf("Halaman %d/{nb}", pdf.PageNo())
		pdf.CellFormat(62, 6, pageStr, "1", 0, "C", true, 0, "")
		pdf.CellFormat(62, 6, "Faktur Resmi Pembelian Buku ", "1", 0, "R", true, 0, "")
	})
	pdf.AliasNbPages("{nb}")

	pdf.AddPage()

	// -------------------------------------------------------------
	// 1. TOP HEADER (Centered Title, Invoice No, Date, & Divider Line)
	// -------------------------------------------------------------
	pdf.SetY(14)
	pdf.SetFont("Arial", "B", 13)
	pdf.SetTextColor(17, 24, 39) // #111827
	pdf.CellFormat(186, 6, "FAKTUR / INVOICE PEMBELIAN", "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 8.5)
	pdf.SetTextColor(75, 85, 99)
	dateStr := order.CreatedAt.Format("02 Januari 2006 15:04 WIB")
	invoiceMetaStr := fmt.Sprintf("No. Pesanan: %s  |  Tanggal: %s", order.InvoiceNumber, dateStr)
	pdf.CellFormat(186, 5, invoiceMetaStr, "", 1, "C", false, 0, "")

	// Horizontal divider rule
	pdf.SetDrawColor(31, 41, 55)
	pdf.SetLineWidth(0.5)
	dividerY := pdf.GetY() + 2.0
	pdf.Line(12, dividerY, 198, dividerY)

	pdf.SetY(dividerY + 5.0)

	// -------------------------------------------------------------
	// 2. SECTION 1: INFORMASI PESANAN & PENGIRIMAN
	// -------------------------------------------------------------
	drawAccountingSectionHeading(pdf, "Informasi Pesanan & Pengiriman")
	pdf.Ln(4.5)

	boxW := 91.0
	boxH := 32.0
	leftX := 12.0
	rightX := leftX + boxW + 4.0 // 12 + 91 + 4 = 107
	startY := pdf.GetY()

	// Left Box: Penerima
	pdf.SetDrawColor(209, 213, 219)
	pdf.SetLineWidth(0.2)
	pdf.SetFillColor(255, 255, 255)
	pdf.Rect(leftX, startY, boxW, boxH, "FD")

	// Header strip
	pdf.SetFillColor(243, 244, 246)
	pdf.Rect(leftX, startY, boxW, 5.5, "FD")
	pdf.SetXY(leftX+3, startY+1.0)
	pdf.SetFont("Arial", "B", 7.5)
	pdf.SetTextColor(17, 24, 39)
	pdf.CellFormat(boxW-6, 3.8, "TUJUAN PENGIRIMAN & PENERIMA", "", 2, "L", false, 0, "")

	pdf.SetXY(leftX+3, startY+7.2)
	pdf.SetFont("Arial", "B", 8.0)
	pdf.CellFormat(boxW-6, 4.0, order.RecipientName, "", 2, "L", false, 0, "")

	pdf.SetFont("Arial", "", 7.5)
	pdf.SetTextColor(75, 85, 99)
	pdf.CellFormat(boxW-6, 3.8, "No. Telepon: "+order.RecipientPhone, "", 2, "L", false, 0, "")

	addr := order.RecipientAddress
	if len(addr) > 50 {
		addr = addr[:47] + "..."
	}
	pdf.SetTextColor(17, 24, 39)
	pdf.CellFormat(boxW-6, 3.8, addr, "", 2, "L", false, 0, "")

	pdf.SetTextColor(75, 85, 99)
	loc := fmt.Sprintf("%s, %s %s", order.CityName, order.ProvinceName, order.PostalCode)
	if len(loc) > 50 {
		loc = loc[:47] + "..."
	}
	pdf.CellFormat(boxW-6, 3.8, loc, "", 0, "L", false, 0, "")

	// Right Box: Kurir & Logistik
	pdf.SetFillColor(255, 255, 255)
	pdf.Rect(rightX, startY, boxW, boxH, "FD")

	// Header strip
	pdf.SetFillColor(243, 244, 246)
	pdf.Rect(rightX, startY, boxW, 5.5, "FD")
	pdf.SetXY(rightX+3, startY+1.0)
	pdf.SetFont("Arial", "B", 7.5)
	pdf.SetTextColor(17, 24, 39)
	pdf.CellFormat(boxW-6, 3.8, "KURIR & STATUS TRANSAKSI", "", 2, "L", false, 0, "")

	pdf.SetXY(rightX+3, startY+7.2)
	pdf.SetFont("Arial", "B", 8.0)
	courierStr := fmt.Sprintf("%s - %s", strings.ToUpper(order.Courier), strings.ToUpper(order.CourierService))
	pdf.CellFormat(boxW-6, 4.0, courierStr, "", 2, "L", false, 0, "")

	pdf.SetFont("Arial", "", 7.5)
	pdf.SetTextColor(75, 85, 99)
	pdf.CellFormat(boxW-6, 3.8, fmt.Sprintf("Total Berat: %d gram", order.TotalWeightGrams), "", 2, "L", false, 0, "")

	trackingNo := "-"
	if order.ShippingTrackingNumber != nil && *order.ShippingTrackingNumber != "" {
		trackingNo = *order.ShippingTrackingNumber
	}
	pdf.SetTextColor(17, 24, 39)
	pdf.CellFormat(boxW-6, 3.8, "No. Resi Pengiriman: "+trackingNo, "", 2, "L", false, 0, "")

	// Status line
	statusText := strings.ToUpper(strings.ReplaceAll(string(order.Status), "_", " "))
	pdf.SetFont("Arial", "B", 7.5)
	if order.Status == entity.OrderStatusPaid || order.Status == entity.OrderStatusCompleted {
		pdf.SetTextColor(6, 95, 70) // Emerald #065f46
		statusText = "STATUS: " + statusText + " (LUNAS)"
	} else if order.Status == entity.OrderStatusWaitingPayment {
		pdf.SetTextColor(146, 64, 14) // Amber #92400e
		statusText = "STATUS: " + statusText + " (MENUNGGU PEMBAYARAN)"
	} else {
		pdf.SetTextColor(153, 27, 27) // Red
		statusText = "STATUS: " + statusText
	}
	pdf.CellFormat(boxW-6, 3.8, statusText, "", 0, "L", false, 0, "")

	pdf.SetY(startY + boxH + 6.0)

	// -------------------------------------------------------------
	// 3. SECTION 2: RINCIAN PEMBELIAN PRODUK (Purchased Items Table)
	// -------------------------------------------------------------
	drawAccountingSectionHeading(pdf, "Rincian Pembelian Produk")
	pdf.Ln(4.5)

	wItemNo := 8.0
	wItemTitle := 96.0
	wItemPrice := 32.0
	wItemQty := 18.0
	wItemSubtotal := 32.0

	// Table Header
	pdf.SetFillColor(243, 244, 246)
	pdf.SetDrawColor(209, 213, 219)
	pdf.SetLineWidth(0.2)
	pdf.SetFont("Arial", "B", 7.5)
	pdf.SetTextColor(17, 24, 39)

	pdf.CellFormat(wItemNo, 5.5, "NO", "1", 0, "C", true, 0, "")
	pdf.CellFormat(wItemTitle, 5.5, " JUDUL BUKU", "1", 0, "L", true, 0, "")
	pdf.CellFormat(wItemPrice, 5.5, "HARGA SATUAN ", "1", 0, "R", true, 0, "")
	pdf.CellFormat(wItemQty, 5.5, "QTY", "1", 0, "C", true, 0, "")
	pdf.CellFormat(wItemSubtotal, 5.5, "SUBTOTAL ", "1", 1, "R", true, 0, "")

	pdf.SetFont("Arial", "", 7.5)

	for i, it := range order.Items {
		if pdf.GetY() > 250 {
			pdf.AddPage()
			pdf.SetFillColor(243, 244, 246)
			pdf.SetDrawColor(209, 213, 219)
			pdf.SetLineWidth(0.2)
			pdf.SetFont("Arial", "B", 7.5)
			pdf.SetTextColor(17, 24, 39)

			pdf.CellFormat(wItemNo, 5.5, "NO", "1", 0, "C", true, 0, "")
			pdf.CellFormat(wItemTitle, 5.5, " JUDUL BUKU", "1", 0, "L", true, 0, "")
			pdf.CellFormat(wItemPrice, 5.5, "HARGA SATUAN ", "1", 0, "R", true, 0, "")
			pdf.CellFormat(wItemQty, 5.5, "QTY", "1", 0, "C", true, 0, "")
			pdf.CellFormat(wItemSubtotal, 5.5, "SUBTOTAL ", "1", 1, "R", true, 0, "")
			pdf.SetFont("Arial", "", 7.5)
		}

		if i%2 == 1 {
			pdf.SetFillColor(248, 250, 252)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		title := it.Title
		if len(title) > 55 {
			title = title[:52] + "..."
		}

		pdf.SetTextColor(17, 24, 39)
		pdf.CellFormat(wItemNo, 5.5, fmt.Sprintf("%d", i+1), "1", 0, "C", true, 0, "")
		pdf.CellFormat(wItemTitle, 5.5, " "+title, "1", 0, "L", true, 0, "")
		pdf.CellFormat(wItemPrice, 5.5, FormatRupiah(it.Price)+" ", "1", 0, "R", true, 0, "")
		pdf.CellFormat(wItemQty, 5.5, fmt.Sprintf("%d", it.Quantity), "1", 0, "C", true, 0, "")
		pdf.CellFormat(wItemSubtotal, 5.5, FormatRupiah(it.Subtotal)+" ", "1", 1, "R", true, 0, "")
	}

	// -------------------------------------------------------------
	// 4. FINANCIAL SUMMARY BREAKDOWN ROWS
	// -------------------------------------------------------------
	wSummaryLabel := wItemNo + wItemTitle + wItemPrice + wItemQty

	// Subtotal Buku
	pdf.SetFillColor(255, 255, 255)
	pdf.SetFont("Arial", "", 7.5)
	pdf.SetTextColor(55, 65, 81)
	pdf.CellFormat(wSummaryLabel, 5.2, "Subtotal Pembelian Buku ", "1", 0, "R", true, 0, "")
	pdf.SetFont("Arial", "B", 7.5)
	pdf.SetTextColor(17, 24, 39)
	pdf.CellFormat(wItemSubtotal, 5.2, FormatRupiah(order.ItemsSubtotal)+" ", "1", 1, "R", true, 0, "")

	// Ongkir
	pdf.SetFont("Arial", "", 7.5)
	pdf.SetTextColor(55, 65, 81)
	pdf.CellFormat(wSummaryLabel, 5.2, "Biaya Pengiriman (Ongkir) ", "1", 0, "R", true, 0, "")
	pdf.SetFont("Arial", "B", 7.5)
	pdf.SetTextColor(17, 24, 39)
	pdf.CellFormat(wItemSubtotal, 5.2, FormatRupiah(order.ShippingCost)+" ", "1", 1, "R", true, 0, "")

	// Diskon Voucher (if any)
	if order.DiscountAmount > 0 {
		pdf.SetFont("Arial", "", 7.5)
		pdf.SetTextColor(185, 28, 28) // Red
		pdf.CellFormat(wSummaryLabel, 5.2, "Potongan Diskon Voucher ", "1", 0, "R", true, 0, "")
		pdf.SetFont("Arial", "B", 7.5)
		pdf.CellFormat(wItemSubtotal, 5.2, "- "+FormatRupiah(order.DiscountAmount)+" ", "1", 1, "R", true, 0, "")
		pdf.SetTextColor(17, 24, 39)
	}

	// Biaya Layanan / Admin (if any)
	if order.AdminFee > 0 {
		pdf.SetFont("Arial", "", 7.5)
		pdf.SetTextColor(55, 65, 81)
		pdf.CellFormat(wSummaryLabel, 5.2, "Biaya Layanan & Penanganan ", "1", 0, "R", true, 0, "")
		pdf.SetFont("Arial", "B", 7.5)
		pdf.SetTextColor(17, 24, 39)
		pdf.CellFormat(wItemSubtotal, 5.2, FormatRupiah(order.AdminFee)+" ", "1", 1, "R", true, 0, "")
	}

	// TOTAL PEMBAYARAN (Highlighted with Soft Green Tint #d1fae5)
	pdf.SetFillColor(209, 250, 229) // Soft Green #d1fae5
	pdf.SetFont("Arial", "B", 8.0)
	pdf.SetTextColor(6, 95, 70) // Emerald #065f46
	pdf.CellFormat(wSummaryLabel, 6.2, "TOTAL PEMBAYARAN ", "1", 0, "R", true, 0, "")
	pdf.SetFont("Arial", "B", 9.0)
	pdf.CellFormat(wItemSubtotal, 6.2, FormatRupiah(order.TotalAmount)+" ", "1", 1, "R", true, 0, "")

	pdf.Ln(6.0)

	// -------------------------------------------------------------
	// 5. SECTION 3: DETAIL PEMBAYARAN (Payment Instructions / Status)
	// -------------------------------------------------------------
	drawAccountingSectionHeading(pdf, "Detail Pembayaran")
	pdf.Ln(4.5)

	payW := 186.0
	payH := 18.0
	payY := pdf.GetY()

	pdf.SetDrawColor(209, 213, 219)
	pdf.SetLineWidth(0.2)
	pdf.SetFillColor(248, 250, 252) // #f8fafc
	pdf.Rect(12, payY, payW, payH, "FD")

	pdf.SetXY(15, payY+2.5)
	pdf.SetFont("Arial", "B", 7.5)
	pdf.SetTextColor(17, 24, 39)
	methodStr := "METODE PEMBAYARAN: MIDTRANS ONLINE"
	if order.Payment != nil {
		methodStr = "METODE: " + strings.ToUpper(strings.ReplaceAll(order.Payment.PaymentType, "_", " "))
		if order.Payment.Bank != nil {
			methodStr += " (" + strings.ToUpper(*order.Payment.Bank) + ")"
		}
	}
	pdf.CellFormat(90, 4.0, methodStr, "", 0, "L", false, 0, "")

	// Right side: Status / Deadline
	pdf.SetXY(105, payY+2.5)
	if order.PaidAt != nil {
		pdf.SetTextColor(6, 95, 70)
		pdf.CellFormat(90, 4.0, "DIBAYAR PADA: "+order.PaidAt.Format("02 Jan 2006 15:04 WIB"), "", 0, "R", false, 0, "")
	} else {
		pdf.SetTextColor(146, 64, 14)
		pdf.CellFormat(90, 4.0, "BATAS WAKTU: "+order.PaymentDeadline.Format("02 Jan 2006 15:04 WIB"), "", 0, "R", false, 0, "")
	}

	pdf.SetXY(15, payY+7.5)
	pdf.SetFont("Arial", "", 7.5)
	pdf.SetTextColor(75, 85, 99)
	if order.Payment != nil && order.Payment.VANumber != nil {
		pdf.CellFormat(180, 4.0, "Nomor Virtual Account (VA): "+*order.Payment.VANumber, "", 2, "L", false, 0, "")
	} else if order.Payment != nil && order.Payment.BillKey != nil {
		pdf.CellFormat(180, 4.0, "Kode Pembayaran / Bill Key: "+*order.Payment.BillKey, "", 2, "L", false, 0, "")
	} else {
		pdf.CellFormat(180, 4.0, "Pembayaran diproses otomatis melalui payment gateway.", "", 2, "L", false, 0, "")
	}

	pdf.SetFont("Arial", "I", 7.0)
	pdf.CellFormat(180, 4.0, "* Faktur ini adalah bukti pembayaran yang sah diterbitkan oleh sistem.", "", 0, "L", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed generating invoice pdf: %w", err)
	}

	return buf.Bytes(), nil
}
