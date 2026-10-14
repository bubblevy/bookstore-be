package handler

import (
	"strconv"

	"github.com/bubblevy/bookstore/internal/delivery/http/middleware"
	"github.com/bubblevy/bookstore/pkg/rajaongkir"
	"github.com/bubblevy/bookstore/pkg/response"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type ShippingHandler struct {
	roClient *rajaongkir.Client
	validate *validator.Validate
}

type CheckCostPayload struct {
	DestinationID int    `json:"destination_id" validate:"required,gt=0"`
	WeightGrams   int    `json:"weight_grams" validate:"required,gt=0"`
	Courier       string `json:"courier"`
}

func NewShippingHandler(r fiber.Router, roClient *rajaongkir.Client, validate *validator.Validate) {
	h := &ShippingHandler{
		roClient: roClient,
		validate: validate,
	}

	routes := r.Group("/shipping")
	routes.Get("/destinations", middleware.RateLimit(middleware.StandardRateLimit), h.SearchDestinations)
	routes.Post("/cost", middleware.RateLimit(middleware.StandardRateLimit), h.CheckCost)
}

func (h *ShippingHandler) SearchDestinations(c fiber.Ctx) error {
	keyword := c.Query("search")
	if len(keyword) < 3 {
		return response.Error(c, fiber.StatusBadRequest, "search keyword must be at least 3 characters", nil)
	}

	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	destinations, err := h.roClient.SearchDestination(c.Context(), keyword, limit, offset)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "destinations retrieved successfully", destinations)
}

func (h *ShippingHandler) CheckCost(c fiber.Ctx) error {
	var payload CheckCostPayload
	if err := c.Bind().Body(&payload); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", nil)
	}

	if err := h.validate.Struct(payload); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request payload", response.FormatValidationErrors(err))
	}

	costs, err := h.roClient.CalculateCost(c.Context(), rajaongkir.CalculateDomesticCostRequest{
		DestinationID: payload.DestinationID,
		WeightGrams:   payload.WeightGrams,
		Courier:       payload.Courier,
	})
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "shipping costs calculated successfully", costs)
}
