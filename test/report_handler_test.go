package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockReportUsecase struct {
	mock.Mock
}

func (m *MockReportUsecase) GetOrderReport(ctx context.Context, filter dto.OrderReportFilterRequest) (*dto.OrderReportResponse, error) {
	args := m.Called(ctx, filter)
	if res := args.Get(0); res != nil {
		return res.(*dto.OrderReportResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockReportUsecase) GetOrderReportPDF(ctx context.Context, filter dto.OrderReportFilterRequest) ([]byte, string, error) {
	args := m.Called(ctx, filter)
	if res := args.Get(0); res != nil {
		return res.([]byte), args.String(1), args.Error(2)
	}
	return nil, "", args.Error(2)
}

func setupReportApp(mockUC *MockReportUsecase) *fiber.App {
	app := fiber.New()

	authMiddleware := func(c fiber.Ctx) error {
		c.Locals("user_id", uint(1))
		c.Locals("is_admin", true)
		return c.Next()
	}

	adminMiddleware := func(c fiber.Ctx) error {
		return c.Next()
	}

	api := app.Group("/api/v1")
	handler.NewReportHandler(api, mockUC, authMiddleware, adminMiddleware)
	return app
}

func TestReportHandler_GetOrderReport(t *testing.T) {
	t.Run("success returns json report 200", func(t *testing.T) {
		mockUC := new(MockReportUsecase)
		app := setupReportApp(mockUC)

		expectedReport := &dto.OrderReportResponse{
			PeriodStart: "01 Oct 2026",
			PeriodEnd:   "31 Oct 2026",
			GeneratedAt: time.Now(),
			Summary: dto.OrderReportSummary{
				TotalOrders: 10,
				PaidOrders:  8,
				NetRevenue:  1500000,
			},
		}

		mockUC.On("GetOrderReport", mock.Anything, mock.MatchedBy(func(req dto.OrderReportFilterRequest) bool {
			return req.StartDate == "2026-10-01" && req.EndDate == "2026-10-31"
		})).Return(expectedReport, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/reports?start_date=2026-10-01&end_date=2026-10-31", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})
}

func TestReportHandler_DownloadReportPDF(t *testing.T) {
	t.Run("success download report pdf returns 200 and application/pdf", func(t *testing.T) {
		mockUC := new(MockReportUsecase)
		app := setupReportApp(mockUC)

		dummyPDF := []byte("%PDF-1.4 dummy report pdf")
		mockUC.On("GetOrderReportPDF", mock.Anything, mock.Anything).
			Return(dummyPDF, "sales-report.pdf", nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/pdf", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/pdf", resp.Header.Get("Content-Type"))
		assert.Contains(t, resp.Header.Get("Content-Disposition"), "sales-report.pdf")
		mockUC.AssertExpectations(t)
	})
}
