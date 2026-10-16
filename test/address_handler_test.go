package test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bubblevy/bookstore/internal/delivery/http/dto"
	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAddressUsecase struct {
	mock.Mock
}

func (m *MockAddressUsecase) CreateAddress(ctx context.Context, userID uint, input usecase.CreateAddressInput) (*usecase.AddressResult, error) {
	args := m.Called(ctx, userID, input)
	if res := args.Get(0); res != nil {
		return res.(*usecase.AddressResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAddressUsecase) GetAddressByID(ctx context.Context, userID, addressID uint) (*usecase.AddressResult, error) {
	args := m.Called(ctx, userID, addressID)
	if res := args.Get(0); res != nil {
		return res.(*usecase.AddressResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAddressUsecase) GetUserAddresses(ctx context.Context, userID uint) ([]usecase.AddressListItemResult, error) {
	args := m.Called(ctx, userID)
	if res := args.Get(0); res != nil {
		return res.([]usecase.AddressListItemResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAddressUsecase) GetPrimaryAddress(ctx context.Context, userID uint) (*usecase.AddressResult, error) {
	args := m.Called(ctx, userID)
	if res := args.Get(0); res != nil {
		return res.(*usecase.AddressResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAddressUsecase) UpdateAddress(ctx context.Context, userID, addressID uint, input usecase.UpdateAddressInput) (*usecase.AddressResult, error) {
	args := m.Called(ctx, userID, addressID, input)
	if res := args.Get(0); res != nil {
		return res.(*usecase.AddressResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAddressUsecase) DeleteAddress(ctx context.Context, userID, addressID uint) error {
	args := m.Called(ctx, userID, addressID)
	return args.Error(0)
}

func (m *MockAddressUsecase) SetPrimaryAddress(ctx context.Context, userID, addressID uint) error {
	args := m.Called(ctx, userID, addressID)
	return args.Error(0)
}

func setupAddressTestApp(uc usecase.AddressUsecase, injectedUserID any) *fiber.App {
	app := fiber.New()
	validate := validator.New()

	authMiddleware := func(c fiber.Ctx) error {
		if injectedUserID != nil {
			c.Locals("user_id", injectedUserID)
			return c.Next()
		}
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    fiber.StatusUnauthorized,
			"status":  false,
			"message": "unauthorized",
		})
	}

	api := app.Group("/api/v1")
	handler.NewAddressHandler(api, uc, validate, authMiddleware)
	return app
}

func TestAddressHandler_Create(t *testing.T) {
	validReq := dto.CreateAddressRequest{
		ReceiverName:    "Budi Santoso",
		Phone:           "081234567890",
		AddressLine:     "Jl. Pemuda No. 45, RT 01 / RW 03",
		BiteshipAreaID:  "IDNP6IDNC148IDND843IDZ12250",
		ProvinceName:    "DKI Jakarta",
		CityName:        "Jakarta Selatan",
		DistrictName:    "Kebayoran Baru",
		SubdistrictName: "Kramat Pela",
		PostalCode:      "12250",
		IsPrimary:       true,
	}

	expectedResult := &usecase.AddressResult{
		ID:              1,
		ReceiverName:    validReq.ReceiverName,
		Phone:           validReq.Phone,
		AddressLine:     validReq.AddressLine,
		BiteshipAreaID:  validReq.BiteshipAreaID,
		ProvinceName:    validReq.ProvinceName,
		CityName:        validReq.CityName,
		DistrictName:    validReq.DistrictName,
		SubdistrictName: validReq.SubdistrictName,
		PostalCode:      validReq.PostalCode,
		IsPrimary:       true,
	}

	t.Run("success create address returns 201", func(t *testing.T) {
		mockUC := new(MockAddressUsecase)
		app := setupAddressTestApp(mockUC, uint(10))

		mockUC.On("CreateAddress", mock.Anything, uint(10), mock.MatchedBy(func(in usecase.CreateAddressInput) bool {
			return in.ReceiverName == validReq.ReceiverName && in.BiteshipAreaID == validReq.BiteshipAreaID
		})).Return(expectedResult, nil).Once()

		body, _ := json.Marshal(validReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/addresses/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("unauthorized when no user_id in locals returns 401", func(t *testing.T) {
		mockUC := new(MockAddressUsecase)
		app := setupAddressTestApp(mockUC, nil)

		body, _ := json.Marshal(validReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/addresses/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("validation error on missing required field returns 400", func(t *testing.T) {
		mockUC := new(MockAddressUsecase)
		app := setupAddressTestApp(mockUC, uint(10))

		invalidReq := validReq
		invalidReq.ReceiverName = ""

		body, _ := json.Marshal(invalidReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/addresses/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 400 when address limit reached", func(t *testing.T) {
		mockUC := new(MockAddressUsecase)
		app := setupAddressTestApp(mockUC, uint(10))

		mockUC.On("CreateAddress", mock.Anything, uint(10), mock.Anything).
			Return(nil, entity.ErrAddressLimitReached).Once()

		body, _ := json.Marshal(validReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/addresses/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})
}

func TestAddressHandler_GetAll(t *testing.T) {
	t.Run("success get all addresses returns 200 with compact list", func(t *testing.T) {
		mockUC := new(MockAddressUsecase)
		app := setupAddressTestApp(mockUC, uint(10))

		results := []usecase.AddressListItemResult{
			{ID: 1, ReceiverName: "Budi", Phone: "08123456789", CityName: "Jakarta Selatan", DistrictName: "Kebayoran Baru", IsPrimary: true},
			{ID: 2, ReceiverName: "Kantor", Phone: "08987654321", CityName: "Jakarta Pusat", DistrictName: "Menteng", IsPrimary: false},
		}

		mockUC.On("GetUserAddresses", mock.Anything, uint(10)).Return(results, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/addresses/", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("error internal server on repository failure returns 500", func(t *testing.T) {
		mockUC := new(MockAddressUsecase)
		app := setupAddressTestApp(mockUC, uint(10))

		mockUC.On("GetUserAddresses", mock.Anything, uint(10)).
			Return(nil, errors.New("database connection broken")).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/addresses/", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})
}

func TestAddressHandler_GetPrimary(t *testing.T) {
	t.Run("success get primary returns 200", func(t *testing.T) {
		mockUC := new(MockAddressUsecase)
		app := setupAddressTestApp(mockUC, uint(10))

		res := &usecase.AddressResult{
			ID:           1,
			ReceiverName: "Budi",
			IsPrimary:    true,
		}

		mockUC.On("GetPrimaryAddress", mock.Anything, uint(10)).Return(res, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/addresses/primary", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("primary not found returns 404", func(t *testing.T) {
		mockUC := new(MockAddressUsecase)
		app := setupAddressTestApp(mockUC, uint(10))

		mockUC.On("GetPrimaryAddress", mock.Anything, uint(10)).
			Return(nil, entity.ErrAddressNotFound).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/addresses/primary", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})
}

func TestAddressHandler_GetByID(t *testing.T) {
	t.Run("success get by id returns 200 with full details", func(t *testing.T) {
		mockUC := new(MockAddressUsecase)
		app := setupAddressTestApp(mockUC, uint(10))

		res := &usecase.AddressResult{
			ID:           5,
			ReceiverName: "Budi",
			AddressLine:  "Jl. Pemuda No. 45",
			CityName:     "Jakarta Selatan",
		}

		mockUC.On("GetAddressByID", mock.Anything, uint(10), uint(5)).Return(res, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/addresses/5", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("invalid id parameter returns 400", func(t *testing.T) {
		mockUC := new(MockAddressUsecase)
		app := setupAddressTestApp(mockUC, uint(10))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/addresses/abc", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("address not found returns 404", func(t *testing.T) {
		mockUC := new(MockAddressUsecase)
		app := setupAddressTestApp(mockUC, uint(10))

		mockUC.On("GetAddressByID", mock.Anything, uint(10), uint(99)).
			Return(nil, entity.ErrAddressNotFound).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/addresses/99", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("unauthorized access to other user's address returns 403", func(t *testing.T) {
		mockUC := new(MockAddressUsecase)
		app := setupAddressTestApp(mockUC, uint(10))

		mockUC.On("GetAddressByID", mock.Anything, uint(10), uint(88)).
			Return(nil, entity.ErrUnauthorizedAddress).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/addresses/88", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})
}

func TestAddressHandler_Update(t *testing.T) {
	updateReq := dto.UpdateAddressRequest{
		ReceiverName:    "Budi Updated",
		Phone:           "089988776655",
		AddressLine:     "Jl. Thamrin No. 88",
		BiteshipAreaID:  "IDNP6IDNC148IDND843IDZ12250",
		ProvinceName:    "DKI Jakarta",
		CityName:        "Jakarta Pusat",
		DistrictName:    "Menteng",
		SubdistrictName: "Gondangdia",
		PostalCode:      "10310",
		IsPrimary:       false,
	}

	t.Run("success update address returns 200", func(t *testing.T) {
		mockUC := new(MockAddressUsecase)
		app := setupAddressTestApp(mockUC, uint(10))

		updatedRes := &usecase.AddressResult{
			ID:           1,
			ReceiverName: updateReq.ReceiverName,
			AddressLine:  updateReq.AddressLine,
		}

		mockUC.On("UpdateAddress", mock.Anything, uint(10), uint(1), mock.MatchedBy(func(in usecase.UpdateAddressInput) bool {
			return in.ReceiverName == updateReq.ReceiverName && in.PostalCode == updateReq.PostalCode
		})).Return(updatedRes, nil).Once()

		body, _ := json.Marshal(updateReq)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/addresses/1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("update non-existing address returns 404", func(t *testing.T) {
		mockUC := new(MockAddressUsecase)
		app := setupAddressTestApp(mockUC, uint(10))

		mockUC.On("UpdateAddress", mock.Anything, uint(10), uint(999), mock.Anything).
			Return(nil, entity.ErrAddressNotFound).Once()

		body, _ := json.Marshal(updateReq)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/addresses/999", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})
}

func TestAddressHandler_Delete(t *testing.T) {
	t.Run("success delete non-primary returns 200", func(t *testing.T) {
		mockUC := new(MockAddressUsecase)
		app := setupAddressTestApp(mockUC, uint(10))

		mockUC.On("DeleteAddress", mock.Anything, uint(10), uint(2)).Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/addresses/2", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("delete primary address returns 400 bad request", func(t *testing.T) {
		mockUC := new(MockAddressUsecase)
		app := setupAddressTestApp(mockUC, uint(10))

		mockUC.On("DeleteAddress", mock.Anything, uint(10), uint(1)).
			Return(entity.ErrPrimaryAddressDelete).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/addresses/1", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})
}

func TestAddressHandler_SetPrimary(t *testing.T) {
	t.Run("success set primary returns 200", func(t *testing.T) {
		mockUC := new(MockAddressUsecase)
		app := setupAddressTestApp(mockUC, uint(10))

		mockUC.On("SetPrimaryAddress", mock.Anything, uint(10), uint(3)).Return(nil).Once()

		req := httptest.NewRequest(http.MethodPatch, "/api/v1/addresses/3/primary", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})

	t.Run("set primary address not found returns 404", func(t *testing.T) {
		mockUC := new(MockAddressUsecase)
		app := setupAddressTestApp(mockUC, uint(10))

		mockUC.On("SetPrimaryAddress", mock.Anything, uint(10), uint(99)).
			Return(entity.ErrAddressNotFound).Once()

		req := httptest.NewRequest(http.MethodPatch, "/api/v1/addresses/99/primary", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
		mockUC.AssertExpectations(t)
	})
}
