package pagination

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

type Meta struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	TotalItems  int64 `json:"total_items"`
	TotalPages  int   `json:"total_pages"`
}

type Page[T any] struct {
	Items []T  `json:"items"`
	Meta  Meta `json:"meta"`
}

type Query struct {
	Page    int
	PerPage int
}

func (q Query) Offset() int {
	return (q.Page - 1) * q.PerPage
}

func FromFiber(c fiber.Ctx) Query {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := strconv.Atoi(c.Query("per_page", "10"))
	if err != nil || perPage < 1 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100
	}

	return Query{
		Page:    page,
		PerPage: perPage,
	}
}

func BuildMeta(totalItems int64, page, perPage int) Meta {
	totalPages := 0
	if perPage > 0 {
		totalPages = int(math.Ceil(float64(totalItems) / float64(perPage)))
	}
	return Meta{
		CurrentPage: page,
		PerPage:     perPage,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
	}
}
