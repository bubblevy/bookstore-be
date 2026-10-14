package test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// MOCK ADDRESS USECASE
// ============================================================================

type MockAddressUsecase struct {
	mock.Mock
}

func (m *MockAddressUsecase) Create(ctx context.Context, input usecase.CreateAddressInput) (*usecase.AddressResult, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.AddressResult), args.Error(1)
}

func (m *MockAddressUsecase) GetAllByUser(ctx context.Context, userID uint) ([]usecase.AddressListItem, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]usecase.AddressListItem), args.Error(1)
}

func (m *MockAddressUsecase) GetByID(ctx context.Context, id uint, userID uint) (*usecase.AddressResult, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.AddressResult), args.Error(1)
}

func (m *MockAddressUsecase) Update(ctx context.Context, input usecase.UpdateAddressInput) (*usecase.AddressResult, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.AddressResult), args.Error(1)
}

func (m *MockAddressUsecase) Delete(ctx context.Context, id uint, userID uint) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockAddressUsecase) SetPrimary(ctx context.Context, id uint, userID uint) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

// ============================================================================
// SETUP TEST HELPER
// ============================================================================

func setupAddressApp(mockUC *MockAddressUsecase, authenticated bool) *fiber.App {
	app := fiber.New()
	validate := validator.New()

	authMiddleware := func(c fiber.Ctx) error {
		if !authenticated {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    401,
				"status":  false,
				"message": "unauthorized",
			})
		}
		// Simulasi klaim token yang di-inject ke context locals
		c.Locals("user_id", uint(1))
		return c.Next()
	}

	api := app.Group("/api/v1")
	handler.NewAddressHandler(api, mockUC, validate, authMiddleware)

	return app
}

// ============================================================================
// CREATE TESTS
// ============================================================================

func TestAddressHandler_Create_Success(t *testing.T) {
	mockUC := new(MockAddressUsecase)
	app := setupAddressApp(mockUC, true)

	expectedInput := usecase.CreateAddressInput{
		UserID:        1,
		ReceiverName:  "Budi Santoso",
		Phone:         "081234567890",
		AddressLine:   "Jl. Mawar No. 10",
		DestinationID: 101,
		ProvinceName:  "Jawa Tengah",
		CityName:      "Semarang",
		DistrictName:  "Semarang Barat",
		PostalCode:    "50149",
		IsPrimary:     true,
	}

	expectedResult := &usecase.AddressResult{
		ID:            10,
		ReceiverName:  "Budi Santoso",
		Phone:         "081234567890",
		AddressLine:   "Jl. Mawar No. 10",
		DestinationID: 101,
		ProvinceName:  "Jawa Tengah",
		CityName:      "Semarang",
		DistrictName:  "Semarang Barat",
		PostalCode:    "50149",
		IsPrimary:     true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	mockUC.On("Create", mock.Anything, expectedInput).Return(expectedResult, nil)

	payload := `{
		"receiver_name": "Budi Santoso",
		"phone": "081234567890",
		"address_line": "Jl. Mawar No. 10",
		"destination_id": 101,
		"province_name": "Jawa Tengah",
		"city_name": "Semarang",
		"district_name": "Semarang Barat",
		"postal_code": "50149",
		"is_primary": true
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/addresses", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Code int                   `json:"code"`
		Data usecase.AddressResult `json:"data"`
	}
	_ = json.Unmarshal(body, &res)

	assert.Equal(t, 201, res.Code)
	assert.Equal(t, uint(10), res.Data.ID)
	assert.Equal(t, "Budi Santoso", res.Data.ReceiverName)
	mockUC.AssertExpectations(t)
}

func TestAddressHandler_Create_LimitReached(t *testing.T) {
	mockUC := new(MockAddressUsecase)
	app := setupAddressApp(mockUC, true)

	mockUC.On("Create", mock.Anything, mock.Anything).Return(nil, entity.ErrAddressLimitReached)

	payload := `{
		"receiver_name": "Budi Santoso",
		"phone": "081234567890",
		"address_line": "Jl. Mawar No. 10",
		"destination_id": 101,
		"province_name": "Jawa Tengah",
		"city_name": "Semarang",
		"district_name": "Semarang Barat",
		"postal_code": "50149"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/addresses", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestAddressHandler_Create_InvalidPayload(t *testing.T) {
	mockUC := new(MockAddressUsecase)
	app := setupAddressApp(mockUC, true)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/addresses", strings.NewReader(`{invalid json`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mockUC.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// ============================================================================
// FIND ALL TESTS
// ============================================================================

func TestAddressHandler_FindAll_Success(t *testing.T) {
	mockUC := new(MockAddressUsecase)
	app := setupAddressApp(mockUC, true)

	expected := []usecase.AddressListItem{
		{ID: 1, ReceiverName: "Rumah", CityName: "Semarang", IsPrimary: true},
		{ID: 2, ReceiverName: "Kantor", CityName: "Jakarta", IsPrimary: false},
	}

	mockUC.On("GetAllByUser", mock.Anything, uint(1)).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/addresses", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Code int                       `json:"code"`
		Data []usecase.AddressListItem `json:"data"`
	}
	_ = json.Unmarshal(body, &res)

	assert.Equal(t, 200, res.Code)
	assert.Len(t, res.Data, 2)
	assert.Equal(t, "Rumah", res.Data[0].ReceiverName)
	mockUC.AssertExpectations(t)
}

func TestAddressHandler_FindAll_InternalError(t *testing.T) {
	mockUC := new(MockAddressUsecase)
	app := setupAddressApp(mockUC, true)

	mockUC.On("GetAllByUser", mock.Anything, uint(1)).Return(nil, errors.New("db query failed"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/addresses", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

// ============================================================================
// FIND BY ID TESTS
// ============================================================================

func TestAddressHandler_FindByID_Success(t *testing.T) {
	mockUC := new(MockAddressUsecase)
	app := setupAddressApp(mockUC, true)

	expected := &usecase.AddressResult{
		ID:           5,
		ReceiverName: "Rumah",
		CityName:     "Semarang",
	}

	mockUC.On("GetByID", mock.Anything, uint(5), uint(1)).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/addresses/5", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Code int                   `json:"code"`
		Data usecase.AddressResult `json:"data"`
	}
	_ = json.Unmarshal(body, &res)

	assert.Equal(t, 200, res.Code)
	assert.Equal(t, uint(5), res.Data.ID)
	mockUC.AssertExpectations(t)
}

func TestAddressHandler_FindByID_NotFound(t *testing.T) {
	mockUC := new(MockAddressUsecase)
	app := setupAddressApp(mockUC, true)

	mockUC.On("GetByID", mock.Anything, uint(99), uint(1)).Return(nil, entity.ErrAddressNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/addresses/99", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestAddressHandler_FindByID_InvalidParam(t *testing.T) {
	mockUC := new(MockAddressUsecase)
	app := setupAddressApp(mockUC, true)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/addresses/abc", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mockUC.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything, mock.Anything)
}

// ============================================================================
// UPDATE TESTS
// ============================================================================

func TestAddressHandler_Update_Success(t *testing.T) {
	mockUC := new(MockAddressUsecase)
	app := setupAddressApp(mockUC, true)

	expectedInput := usecase.UpdateAddressInput{
		ID:            5,
		UserID:        1,
		ReceiverName:  "Budi Edit",
		Phone:         "081234567899",
		AddressLine:   "Jl. Melati Baru No. 1",
		DestinationID: 102,
		ProvinceName:  "Jawa Tengah",
		CityName:      "Semarang",
		DistrictName:  "Semarang Tengah",
		PostalCode:    "50132",
		IsPrimary:     false,
	}

	expectedResult := &usecase.AddressResult{
		ID:           5,
		ReceiverName: "Budi Edit",
		Phone:        "081234567899",
	}

	mockUC.On("Update", mock.Anything, expectedInput).Return(expectedResult, nil)

	payload := `{
		"receiver_name": "Budi Edit",
		"phone": "081234567899",
		"address_line": "Jl. Melati Baru No. 1",
		"destination_id": 102,
		"province_name": "Jawa Tengah",
		"city_name": "Semarang",
		"district_name": "Semarang Tengah",
		"postal_code": "50132",
		"is_primary": false
	}`

	req := httptest.NewRequest(http.MethodPut, "/api/v1/addresses/5", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Code int                   `json:"code"`
		Data usecase.AddressResult `json:"data"`
	}
	_ = json.Unmarshal(body, &res)

	assert.Equal(t, 200, res.Code)
	assert.Equal(t, "Budi Edit", res.Data.ReceiverName)
	mockUC.AssertExpectations(t)
}

func TestAddressHandler_Update_NotFound(t *testing.T) {
	mockUC := new(MockAddressUsecase)
	app := setupAddressApp(mockUC, true)

	mockUC.On("Update", mock.Anything, mock.Anything).Return(nil, entity.ErrAddressNotFound)

	payload := `{
		"receiver_name": "Budi Edit",
		"phone": "081234567899",
		"address_line": "Jl. Melati Baru No. 1",
		"destination_id": 102,
		"province_name": "Jawa Tengah",
		"city_name": "Semarang",
		"district_name": "Semarang Tengah",
		"postal_code": "50132"
	}`

	req := httptest.NewRequest(http.MethodPut, "/api/v1/addresses/99", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

// ============================================================================
// DELETE TESTS
// ============================================================================

func TestAddressHandler_Delete_Success(t *testing.T) {
	mockUC := new(MockAddressUsecase)
	app := setupAddressApp(mockUC, true)

	mockUC.On("Delete", mock.Anything, uint(5), uint(1)).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/addresses/5", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestAddressHandler_Delete_CannotDeletePrimary(t *testing.T) {
	mockUC := new(MockAddressUsecase)
	app := setupAddressApp(mockUC, true)

	mockUC.On("Delete", mock.Anything, uint(5), uint(1)).Return(entity.ErrCannotDeletePrimaryAddress)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/addresses/5", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestAddressHandler_Delete_NotFound(t *testing.T) {
	mockUC := new(MockAddressUsecase)
	app := setupAddressApp(mockUC, true)

	mockUC.On("Delete", mock.Anything, uint(99), uint(1)).Return(entity.ErrAddressNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/addresses/99", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

// ============================================================================
// SET PRIMARY TESTS
// ============================================================================

func TestAddressHandler_SetPrimary_Success(t *testing.T) {
	mockUC := new(MockAddressUsecase)
	app := setupAddressApp(mockUC, true)

	mockUC.On("SetPrimary", mock.Anything, uint(5), uint(1)).Return(nil)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/addresses/5/primary", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

func TestAddressHandler_SetPrimary_NotFound(t *testing.T) {
	mockUC := new(MockAddressUsecase)
	app := setupAddressApp(mockUC, true)

	mockUC.On("SetPrimary", mock.Anything, uint(99), uint(1)).Return(entity.ErrAddressNotFound)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/addresses/99/primary", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	mockUC.AssertExpectations(t)
}

// ============================================================================
// UNAUTHORIZED TEST
// ============================================================================

func TestAddressHandler_Unauthorized(t *testing.T) {
	mockUC := new(MockAddressUsecase)
	// authenticated = false
	app := setupAddressApp(mockUC, false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/addresses", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
