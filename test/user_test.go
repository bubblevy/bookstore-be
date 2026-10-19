package test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockUserRepositoryForUserTest struct {
	mock.Mock
}

func (m *MockUserRepositoryForUserTest) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserRepositoryForUserTest) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserRepositoryForUserTest) Create(ctx context.Context, user *entity.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *MockUserRepositoryForUserTest) FindByActivationToken(ctx context.Context, token string) (*entity.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserRepositoryForUserTest) FindByResetPasswordToken(ctx context.Context, token string) (*entity.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserRepositoryForUserTest) Update(ctx context.Context, user *entity.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *MockUserRepositoryForUserTest) FindAll(ctx context.Context, filter repository.UserFilter) ([]entity.User, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]entity.User), args.Get(1).(int64), args.Error(2)
}

type MockUserUsecase struct {
	mock.Mock
}

func (m *MockUserUsecase) GetProfile(ctx context.Context, userID uint) (*usecase.UserProfileResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.UserProfileResponse), args.Error(1)
}

func (m *MockUserUsecase) FindAll(ctx context.Context, filter repository.UserFilter, pq pagination.Query) ([]usecase.UserListItem, pagination.Meta, error) {
	args := m.Called(ctx, filter, pq)
	if args.Get(0) == nil {
		return nil, args.Get(1).(pagination.Meta), args.Error(2)
	}
	return args.Get(0).([]usecase.UserListItem), args.Get(1).(pagination.Meta), args.Error(2)
}

func TestUserUsecase_GetProfile(t *testing.T) {
	ctx := context.Background()

	t.Run("success returns safe non-sensitive profile", func(t *testing.T) {
		repo := new(MockUserRepositoryForUserTest)
		uc := usecase.NewUserUsecase(repo)

		dummySecret := "very-secret-hash"
		dummyToken := "activation-secret-token"
		user := &entity.User{
			Model:           gorm.Model{ID: 10, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			Name:            "Budi Santoso",
			Email:           "budi@example.com",
			Password:        dummySecret,
			Role:            entity.RoleCustomer,
			IsActive:        true,
			ActivationToken: &dummyToken,
		}

		repo.On("FindByID", ctx, uint(10)).Return(user, nil).Once()

		res, err := uc.GetProfile(ctx, 10)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, uint(10), res.ID)
		assert.Equal(t, "Budi Santoso", res.Name)
		assert.Equal(t, "budi@example.com", res.Email)
		assert.Equal(t, entity.RoleCustomer, res.Role)
		assert.True(t, res.IsActive)

		// Verify serialized JSON omits any password/token fields
		b, err := json.Marshal(res)
		assert.NoError(t, err)
		assert.NotContains(t, string(b), dummySecret)
		assert.NotContains(t, string(b), dummyToken)
		assert.NotContains(t, string(b), "password")

		repo.AssertExpectations(t)
	})

	t.Run("returns error when user not found", func(t *testing.T) {
		repo := new(MockUserRepositoryForUserTest)
		uc := usecase.NewUserUsecase(repo)

		repo.On("FindByID", ctx, uint(999)).Return(nil, nil).Once()

		res, err := uc.GetProfile(ctx, 999)
		assert.ErrorIs(t, err, entity.ErrUserNotFound)
		assert.Nil(t, res)
		repo.AssertExpectations(t)
	})
}

func TestUserUsecase_FindAll(t *testing.T) {
	ctx := context.Background()

	t.Run("success returns paginated users", func(t *testing.T) {
		repo := new(MockUserRepositoryForUserTest)
		uc := usecase.NewUserUsecase(repo)

		pq := pagination.Query{Page: 1, PerPage: 10}
		filter := repository.UserFilter{
			Role:   "customer",
			Limit:  10,
			Offset: 0,
		}

		users := []entity.User{
			{
				Model:    gorm.Model{ID: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
				Name:     "User 1",
				Email:    "user1@example.com",
				Role:     entity.RoleCustomer,
				IsActive: true,
			},
			{
				Model:    gorm.Model{ID: 2, CreatedAt: time.Now(), UpdatedAt: time.Now()},
				Name:     "User 2",
				Email:    "user2@example.com",
				Role:     entity.RoleCustomer,
				IsActive: true,
			},
		}

		repo.On("FindAll", ctx, filter).Return(users, int64(2), nil).Once()

		items, meta, err := uc.FindAll(ctx, repository.UserFilter{Role: "customer"}, pq)
		assert.NoError(t, err)
		assert.Len(t, items, 2)
		assert.Equal(t, int64(2), meta.TotalItems)
		assert.Equal(t, 1, meta.CurrentPage)
		assert.Equal(t, "User 1", items[0].Name)
		assert.Equal(t, "user1@example.com", items[0].Email)
		repo.AssertExpectations(t)
	})
}

func setupUserTestApp(uc usecase.UserUsecase, injectedUserID any, injectedRole any) *fiber.App {
	app := fiber.New()

	authMiddleware := func(c fiber.Ctx) error {
		if injectedUserID != nil {
			c.Locals("user_id", injectedUserID)
			if injectedRole != nil {
				c.Locals("user_role", injectedRole)
			}
			return c.Next()
		}
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    fiber.StatusUnauthorized,
			"status":  false,
			"message": "unauthorized",
		})
	}

	adminMiddleware := func(c fiber.Ctx) error {
		role, ok := c.Locals("user_role").(string)
		if !ok || role != "admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"code":    fiber.StatusForbidden,
				"status":  false,
				"message": "forbidden",
			})
		}
		return c.Next()
	}

	api := app.Group("/api/v1")
	handler.NewUserHandler(api, uc, authMiddleware, adminMiddleware)

	return app
}

func TestUserHandler_GetMe(t *testing.T) {
	t.Run("authenticated user can view safe profile", func(t *testing.T) {
		mockUc := new(MockUserUsecase)
		app := setupUserTestApp(mockUc, uint(5), "customer")

		profile := &usecase.UserProfileResponse{
			ID:        5,
			Name:      "Budi Santoso",
			Email:     "budi@example.com",
			Role:      entity.RoleCustomer,
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockUc.On("GetProfile", mock.Anything, uint(5)).Return(profile, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var resBody struct {
			Code   int                         `json:"code"`
			Status bool                        `json:"status"`
			Data   usecase.UserProfileResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&resBody)
		assert.NoError(t, err)
		assert.True(t, resBody.Status)
		assert.Equal(t, uint(5), resBody.Data.ID)
		assert.Equal(t, "budi@example.com", resBody.Data.Email)
		mockUc.AssertExpectations(t)
	})

	t.Run("unauthorized when missing user token", func(t *testing.T) {
		mockUc := new(MockUserUsecase)
		app := setupUserTestApp(mockUc, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		mockUc.AssertNotCalled(t, "GetProfile")
	})

	t.Run("returns 404 when user not found in DB", func(t *testing.T) {
		mockUc := new(MockUserUsecase)
		app := setupUserTestApp(mockUc, uint(99), "customer")

		mockUc.On("GetProfile", mock.Anything, uint(99)).Return(nil, entity.ErrUserNotFound).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		mockUc.AssertExpectations(t)
	})
}

func TestUserHandler_FindAll(t *testing.T) {
	t.Run("admin can list users with pagination", func(t *testing.T) {
		mockUc := new(MockUserUsecase)
		app := setupUserTestApp(mockUc, uint(1), "admin")

		items := []usecase.UserListItem{
			{ID: 1, Name: "Admin User", Email: "admin@example.com", Role: entity.RoleAdmin, IsActive: true},
			{ID: 2, Name: "Customer User", Email: "cust@example.com", Role: entity.RoleCustomer, IsActive: true},
		}
		meta := pagination.Meta{
			CurrentPage: 1,
			PerPage:     10,
			TotalItems:  2,
			TotalPages:  1,
		}

		mockUc.On("FindAll", mock.Anything, repository.UserFilter{}, pagination.Query{Page: 1, PerPage: 10}).
			Return(items, meta, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&per_page=10", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var resBody struct {
			Code    int                    `json:"code"`
			Status  bool                   `json:"status"`
			Message string                 `json:"message"`
			Data    []usecase.UserListItem `json:"data"`
			Meta    pagination.Meta        `json:"meta"`
		}
		err = json.NewDecoder(resp.Body).Decode(&resBody)
		assert.NoError(t, err)
		assert.True(t, resBody.Status)
		assert.Len(t, resBody.Data, 2)
		assert.Equal(t, int64(2), resBody.Meta.TotalItems)
		mockUc.AssertExpectations(t)
	})

	t.Run("customer receives 403 forbidden when accessing admin users endpoint", func(t *testing.T) {
		mockUc := new(MockUserUsecase)
		app := setupUserTestApp(mockUc, uint(2), "customer")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		mockUc.AssertNotCalled(t, "FindAll")
	})

	t.Run("returns 500 when usecase fails", func(t *testing.T) {
		mockUc := new(MockUserUsecase)
		app := setupUserTestApp(mockUc, uint(1), "admin")

		mockUc.On("FindAll", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, pagination.Meta{}, errors.New("db error")).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockUc.AssertExpectations(t)
	})
}
