package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type ReportHandler struct {
	usecase usecase.ReportUsecase
}

func NewReportHandler(
	r fiber.Router,
	reportUC usecase.ReportUsecase,
	authMiddleware fiber.Handler,
	adminMiddleware fiber.Handler,
) {
	h := &ReportHandler{usecase: reportUC}

	reports := r.Group("/reports", authMiddleware, adminMiddleware)
	reports.Get("/", h.GetOrderReport)
	reports.Get("/pdf", h.DownloadReportPDF)
}

func (h *ReportHandler) GetOrderReport(c fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "0"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "0"))

	req := dto.OrderReportFilterRequest{
		StartDate: strings.TrimSpace(c.Query("start_date")),
		EndDate:   strings.TrimSpace(c.Query("end_date")),
		Status:    strings.TrimSpace(c.Query("status")),
		Page:      page,
		PerPage:   perPage,
	}

	res, err := h.usecase.GetOrderReport(c.Context(), req)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "order report generated successfully", res)
}

func (h *ReportHandler) DownloadReportPDF(c fiber.Ctx) error {
	req := dto.OrderReportFilterRequest{
		StartDate: strings.TrimSpace(c.Query("start_date")),
		EndDate:   strings.TrimSpace(c.Query("end_date")),
		Status:    strings.TrimSpace(c.Query("status")),
	}

	pdfBytes, filename, err := h.usecase.GetOrderReportPDF(c.Context(), req)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	return c.Send(pdfBytes)
}
