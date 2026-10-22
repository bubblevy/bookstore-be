package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/go-pdf/fpdf"
)

// GenerateSalesReportPDF creates a clean, professional financial & sales report PDF
// adhering strictly to the structured accounting layout with KPI performance summary,
// multi-line transaction breakdowns, top selling books, and standard 3-column page footers.
func GenerateSalesReportPDF(report dto.OrderReportResponse) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 18)

	// Clean 3-column accounting footer with subtle background watermark
	pdf.SetFooterFunc(func() {
		// Draw watermark on each page (rendered with subtle 5% alpha)
		drawWatermark(pdf, "BOOKSTORE", "LAPORAN RESMI // ANALYTICS")

		pdf.SetY(-14)
		pdf.SetDrawColor(209, 213, 219) // #d1d5db light gray border
		pdf.SetLineWidth(0.2)
		pdf.SetFillColor(255, 255, 255)
		pdf.SetTextColor(75, 85, 99) // #4b5563 dark gray
		pdf.SetFont("Arial", "", 7.5)

		// 3-column bottom bar (total width = 186mm)
		nowStr := time.Now().Format("02/01/2006 15:04")
		pdf.CellFormat(62, 6, " "+nowStr, "1", 0, "L", true, 0, "")
		pageStr := fmt.Sprintf("Halaman %d/{nb}", pdf.PageNo())
		pdf.CellFormat(62, 6, pageStr, "1", 0, "C", true, 0, "")
		pdf.CellFormat(62, 6, "Laporan Keuangan & Penjualan ", "1", 0, "R", true, 0, "")
	})
	pdf.AliasNbPages("{nb}")

	pdf.AddPage()

	// -------------------------------------------------------------
	// 1. TOP HEADER (Centered Title, Period, and Clean Dividing Line)
	// -------------------------------------------------------------
	pdf.SetY(14)
	pdf.SetFont("Arial", "B", 13)
	pdf.SetTextColor(17, 24, 39) // #111827 Dark Charcoal
	pdf.CellFormat(186, 6, "LAPORAN KEUANGAN & PENJUALAN", "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 8.5)
	pdf.SetTextColor(75, 85, 99) // #4b5563
	periodStr := fmt.Sprintf("Periode: %s - %s", report.PeriodStart, report.PeriodEnd)
	pdf.CellFormat(186, 5, periodStr, "", 1, "C", false, 0, "")

	// Horizontal divider rule
	pdf.SetDrawColor(31, 41, 55) // #1f2937
	pdf.SetLineWidth(0.5)
	dividerY := pdf.GetY() + 2.0
	pdf.Line(12, dividerY, 198, dividerY)

	pdf.SetY(dividerY + 5.0)

	// -------------------------------------------------------------
	// 2. SECTION 1: RINGKASAN KINERJA (Performance KPI Summary Table)
	// -------------------------------------------------------------
	drawAccountingSectionHeading(pdf, "Ringkasan Kinerja")
	pdf.Ln(4.5)

	kpiColW := 46.5 // 186 / 4

	// KPI Header Row
	pdf.SetDrawColor(209, 213, 219) // #d1d5db
	pdf.SetLineWidth(0.2)
	pdf.SetFillColor(243, 244, 246) // #f3f4f6
	pdf.SetTextColor(17, 24, 39)
	pdf.SetFont("Arial", "B", 7.5)

	pdf.CellFormat(kpiColW, 5.5, "TOTAL OMZET", "1", 0, "C", true, 0, "")
	pdf.CellFormat(kpiColW, 5.5, "TOTAL PENDAPATAN KOTOR", "1", 0, "C", true, 0, "")
	pdf.CellFormat(kpiColW, 5.5, "BUKU TERJUAL", "1", 0, "C", true, 0, "")
	pdf.CellFormat(kpiColW, 5.5, "TOTAL TRANSAKSI", "1", 1, "C", true, 0, "")

	// KPI Values Row (with soft pastel highlight tints matching sample)
	pdf.SetFont("Arial", "B", 8.5)

	// Cell 1: Total Omzet (Light neutral #f3f4f6)
	pdf.SetFillColor(243, 244, 246)
	pdf.SetTextColor(17, 24, 39)
	pdf.CellFormat(kpiColW, 6.5, FormatRupiah(report.Summary.NetRevenue)+" ", "1", 0, "R", true, 0, "")

	// Cell 2: Total Pendapatan Kotor (Soft pink #fee2e2)
	pdf.SetFillColor(254, 226, 226)
	pdf.SetTextColor(153, 27, 27) // #991b1b
	pdf.CellFormat(kpiColW, 6.5, FormatRupiah(report.Summary.GrossRevenue)+" ", "1", 0, "R", true, 0, "")

	// Cell 3: Buku Terjual (Soft yellow #fef3c7)
	pdf.SetFillColor(254, 243, 199)
	pdf.SetTextColor(146, 64, 14) // #92400e
	pdf.CellFormat(kpiColW, 6.5, fmt.Sprintf("%d Eksemplar ", report.Summary.TotalItemsSold), "1", 0, "R", true, 0, "")

	// Cell 4: Total Transaksi (Soft green #d1fae5)
	pdf.SetFillColor(209, 250, 229)
	pdf.SetTextColor(6, 95, 70) // #065f46
	pdf.CellFormat(kpiColW, 6.5, fmt.Sprintf("%d Pesanan ", report.Summary.TotalOrders), "1", 1, "R", true, 0, "")

	pdf.Ln(6.0)

	// -------------------------------------------------------------
	// 3. SECTION 2: DETAIL TRANSAKSI (Transaction Details Table)
	// -------------------------------------------------------------
	drawAccountingSectionHeading(pdf, "Detail Transaksi")
	pdf.Ln(4.5)

	wNo := 8.0
	wTanggal := 30.0
	wPesanan := 34.0
	wMetode := 22.0
	wDetail := 64.0
	wSubtotal := 28.0

	drawTransactionTableHeader(pdf, wNo, wTanggal, wPesanan, wMetode, wDetail, wSubtotal)

	totalOrdersAmount := 0
	sublineH := 3.8

	for i, o := range report.Orders {
		totalOrdersAmount += o.TotalAmount

		// Build multi-line product details breakdown
		type detailSubline struct {
			Label string
			Value string
		}
		var sublines []detailSubline

		qtyDesc := fmt.Sprintf("Subtotal Buku (%d pcs)", o.TotalQuantity)
		sublines = append(sublines, detailSubline{
			Label: qtyDesc,
			Value: FormatRupiah(o.ItemsSubtotal),
		})

		sublines = append(sublines, detailSubline{
			Label: "Biaya Pengiriman (Ongkir)",
			Value: FormatRupiah(o.ShippingCost),
		})

		expectedTotal := o.ItemsSubtotal + o.ShippingCost
		if o.TotalAmount < expectedTotal {
			discount := expectedTotal - o.TotalAmount
			sublines = append(sublines, detailSubline{
				Label: "Diskon Voucher",
				Value: "- " + FormatRupiah(discount),
			})
		} else if o.TotalAmount > expectedTotal {
			adminFee := o.TotalAmount - expectedTotal
			sublines = append(sublines, detailSubline{
				Label: "Biaya Admin/Layanan",
				Value: FormatRupiah(adminFee),
			})
		}

		rowH := float64(len(sublines)) * sublineH
		if rowH < 8.0 {
			rowH = 8.0
		}

		// Handle pagination overflow
		if pdf.GetY()+rowH > 275 {
			pdf.AddPage()
			drawTransactionTableHeader(pdf, wNo, wTanggal, wPesanan, wMetode, wDetail, wSubtotal)
		}

		startY := pdf.GetY()

		// Row background fill
		if i%2 == 1 {
			pdf.SetFillColor(248, 250, 252) // #f8fafc Soft Slate
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		pdf.SetDrawColor(209, 213, 219) // #d1d5db
		pdf.SetLineWidth(0.2)

		col1X := 12.0
		col2X := col1X + wNo
		col3X := col2X + wTanggal
		col4X := col3X + wPesanan
		col5X := col4X + wMetode
		col6X := col5X + wDetail

		// Draw cell borders with background
		pdf.Rect(col1X, startY, wNo, rowH, "FD")
		pdf.Rect(col2X, startY, wTanggal, rowH, "FD")
		pdf.Rect(col3X, startY, wPesanan, rowH, "FD")
		pdf.Rect(col4X, startY, wMetode, rowH, "FD")
		pdf.Rect(col5X, startY, wDetail, rowH, "FD")
		pdf.Rect(col6X, startY, wSubtotal, rowH, "FD")

		// 1. NO
		pdf.SetXY(col1X, startY+(rowH-4.0)/2)
		pdf.SetFont("Arial", "", 7.5)
		pdf.SetTextColor(17, 24, 39)
		pdf.CellFormat(wNo, 4.0, fmt.Sprintf("%d", i+1), "", 0, "C", false, 0, "")

		// 2. TANGGAL
		pdf.SetXY(col2X, startY+(rowH-7.2)/2)
		pdf.SetFont("Arial", "", 7.5)
		pdf.CellFormat(wTanggal, 3.6, o.CreatedAt.Format("02 Jan 2006"), "", 2, "C", false, 0, "")
		pdf.CellFormat(wTanggal, 3.6, o.CreatedAt.Format("15:04 WIB"), "", 0, "C", false, 0, "")

		// 3. NO. PESANAN
		pdf.SetXY(col3X+1.5, startY+(rowH-7.2)/2)
		pdf.SetFont("Arial", "B", 7.5)
		pdf.CellFormat(wPesanan-3.0, 3.6, o.InvoiceNumber, "", 2, "L", false, 0, "")
		custName := o.CustomerName
		if len(custName) > 18 {
			custName = custName[:16] + "..."
		}
		pdf.SetFont("Arial", "", 7.0)
		pdf.SetTextColor(107, 114, 128)
		pdf.CellFormat(wPesanan-3.0, 3.6, custName, "", 0, "L", false, 0, "")
		pdf.SetTextColor(17, 24, 39)

		// 4. METODE
		metode := o.PaymentMethod
		if metode == "" {
			metode = "Midtrans"
		}
		pdf.SetXY(col4X, startY+(rowH-4.0)/2)
		pdf.SetFont("Arial", "", 7.5)
		pdf.CellFormat(wMetode, 4.0, strings.ToUpper(metode), "", 0, "C", false, 0, "")

		// 5. DETAIL PRODUK (Multi-line breakdown)
		for k, sub := range sublines {
			subY := startY + float64(k)*sublineH
			pdf.SetXY(col5X+1.5, subY)
			pdf.SetFont("Arial", "", 7.0)
			if strings.HasPrefix(sub.Label, "Diskon") {
				pdf.SetTextColor(185, 28, 28) // Red for discount
			} else {
				pdf.SetTextColor(55, 65, 81)
			}
			pdf.CellFormat(wDetail-23.0, sublineH, sub.Label, "", 0, "L", false, 0, "")
			pdf.CellFormat(21.5, sublineH, sub.Value, "", 0, "R", false, 0, "")
		}
		pdf.SetTextColor(17, 24, 39)

		// 6. SUBTOTAL
		pdf.SetXY(col6X, startY+(rowH-4.0)/2)
		pdf.SetFont("Arial", "B", 8.0)
		pdf.CellFormat(wSubtotal-1.5, 4.0, FormatRupiah(o.TotalAmount)+" ", "", 0, "R", false, 0, "")

		pdf.SetY(startY + rowH)
	}

	// Transaction Table Footer: TOTAL TRANSAKSI
	wMerged := wNo + wTanggal + wPesanan + wMetode + wDetail
	pdf.SetFillColor(243, 244, 246)
	pdf.SetDrawColor(209, 213, 219)
	pdf.SetLineWidth(0.2)
	pdf.SetFont("Arial", "B", 7.5)
	pdf.SetTextColor(17, 24, 39)
	pdf.CellFormat(wMerged, 6.0, "TOTAL TRANSAKSI ", "1", 0, "R", true, 0, "")
	pdf.SetFont("Arial", "B", 8.5)
	pdf.CellFormat(wSubtotal, 6.0, FormatRupiah(totalOrdersAmount)+" ", "1", 1, "R", true, 0, "")

	pdf.Ln(6.0)

	// -------------------------------------------------------------
	// 4. SECTION 3: DETAIL BUKU TERLARIS (Top Selling Books Table)
	// -------------------------------------------------------------
	if len(report.TopBooks) > 0 {
		if pdf.GetY() > 240 {
			pdf.AddPage()
		}

		drawAccountingSectionHeading(pdf, "Detail Buku Terlaris")
		pdf.Ln(4.5)

		wRank := 8.0
		wTitle := 96.0
		wQtySold := 42.0
		wRevenue := 40.0

		// Table Header
		pdf.SetFillColor(243, 244, 246)
		pdf.SetDrawColor(209, 213, 219)
		pdf.SetLineWidth(0.2)
		pdf.SetFont("Arial", "B", 7.5)
		pdf.SetTextColor(17, 24, 39)

		pdf.CellFormat(wRank, 5.5, "NO", "1", 0, "C", true, 0, "")
		pdf.CellFormat(wTitle, 5.5, " JUDUL BUKU", "1", 0, "L", true, 0, "")
		pdf.CellFormat(wQtySold, 5.5, "JUMLAH TERJUAL", "1", 0, "C", true, 0, "")
		pdf.CellFormat(wRevenue, 5.5, "TOTAL PENDAPATAN ", "1", 1, "R", true, 0, "")

		totalTopRevenue := 0

		pdf.SetFont("Arial", "", 7.5)
		for i, b := range report.TopBooks {
			if i >= 10 {
				break
			}
			totalTopRevenue += b.TotalRevenue

			if i%2 == 1 {
				pdf.SetFillColor(248, 250, 252)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}

			title := b.Title
			if len(title) > 55 {
				title = title[:52] + "..."
			}

			pdf.CellFormat(wRank, 5.5, fmt.Sprintf("%d", i+1), "1", 0, "C", true, 0, "")
			pdf.CellFormat(wTitle, 5.5, " "+title, "1", 0, "L", true, 0, "")
			pdf.CellFormat(wQtySold, 5.5, fmt.Sprintf("%d Eksemplar", b.QuantitySold), "1", 0, "C", true, 0, "")
			pdf.CellFormat(wRevenue, 5.5, FormatRupiah(b.TotalRevenue)+" ", "1", 1, "R", true, 0, "")
		}

		// Top Books Footer
		pdf.SetFillColor(243, 244, 246)
		pdf.SetFont("Arial", "B", 7.5)
		pdf.CellFormat(wRank+wTitle+wQtySold, 6.0, "TOTAL PENDAPATAN BUKU TERLARIS ", "1", 0, "R", true, 0, "")
		pdf.SetFont("Arial", "B", 8.5)
		pdf.CellFormat(wRevenue, 6.0, FormatRupiah(totalTopRevenue)+" ", "1", 1, "R", true, 0, "")
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed generating report pdf: %w", err)
	}

	return buf.Bytes(), nil
}

// drawAccountingSectionHeading draws a clean section heading with a solid vertical black bar.
func drawAccountingSectionHeading(pdf *fpdf.Fpdf, title string) {
	currY := pdf.GetY()
	// Solid vertical black accent bar
	pdf.SetFillColor(0, 0, 0)
	pdf.Rect(12, currY, 2.2, 4.2, "F")

	// Heading Text
	pdf.SetXY(16, currY-0.4)
	pdf.SetFont("Arial", "B", 9.0)
	pdf.SetTextColor(17, 24, 39)
	pdf.CellFormat(100, 4.8, title, "", 0, "L", false, 0, "")
}

// drawTransactionTableHeader renders the column header row for transaction listings.
func drawTransactionTableHeader(pdf *fpdf.Fpdf, wNo, wTanggal, wPesanan, wMetode, wDetail, wSubtotal float64) {
	pdf.SetFillColor(243, 244, 246) // #f3f4f6
	pdf.SetDrawColor(209, 213, 219) // #d1d5db
	pdf.SetLineWidth(0.2)
	pdf.SetFont("Arial", "B", 7.5)
	pdf.SetTextColor(17, 24, 39)

	pdf.CellFormat(wNo, 5.5, "NO", "1", 0, "C", true, 0, "")
	pdf.CellFormat(wTanggal, 5.5, "TANGGAL", "1", 0, "C", true, 0, "")
	pdf.CellFormat(wPesanan, 5.5, " NO. PESANAN", "1", 0, "L", true, 0, "")
	pdf.CellFormat(wMetode, 5.5, "METODE", "1", 0, "C", true, 0, "")
	pdf.CellFormat(wDetail, 5.5, " DETAIL PRODUK", "1", 0, "L", true, 0, "")
	pdf.CellFormat(wSubtotal, 5.5, "SUBTOTAL ", "1", 1, "R", true, 0, "")
}

// drawWatermark renders an elegant diagonal background watermark across the page.
func drawWatermark(pdf *fpdf.Fpdf, mainText, subText string) {
	pdf.TransformBegin()
	// Soft 5% opacity so document text and tables remain 100% readable
	pdf.SetAlpha(0.05, "Normal")
	pdf.SetTextColor(75, 85, 99)

	// Rotate diagonally 45 degrees around center of A4 page (105mm, 148.5mm)
	pdf.TransformRotate(45, 105, 148.5)

	// Main Title
	pdf.SetFont("Arial", "B", 46)
	wMain := pdf.GetStringWidth(mainText)
	pdf.Text(105-(wMain/2), 143, mainText)

	// Subtext
	if subText != "" {
		pdf.SetFont("Arial", "B", 17)
		wSub := pdf.GetStringWidth(subText)
		pdf.Text(105-(wSub/2), 155, subText)
	}

	pdf.TransformEnd()
	pdf.SetAlpha(1.0, "Normal")
}
