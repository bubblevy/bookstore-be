package test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// MOCK CATEGORY REPOSITORY
// ============================================================================

type MockCategoryRepository struct {
	mock.Mock
}

func (m *MockCategoryRepository) Create(ctx context.Context, category *entity.Category) error {
	args := m.Called(ctx, category)
	return args.Error(0)
}

func (m *MockCategoryRepository) FindByID(ctx context.Context, id uint) (*entity.Category, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindBySlug(ctx context.Context, slug string) (*entity.Category, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindAll(ctx context.Context) ([]entity.Category, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.Category), args.Error(1)
}

func (m *MockCategoryRepository) Update(ctx context.Context, category *entity.Category) error {
	args := m.Called(ctx, category)
	return args.Error(0)
}

func (m *MockCategoryRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCategoryRepository) ExistsBySlug(ctx context.Context, slug string, excludeID ...uint) (bool, error) {
	var args mock.Arguments
	if len(excludeID) > 0 {
		args = m.Called(ctx, slug, excludeID[0])
	} else {
		args = m.Called(ctx, slug)
	}
	return args.Bool(0), args.Error(1)
}

// ============================================================================
// USECASE UNIT TESTS
// ============================================================================

func TestCategoryUsecase_Create_Success(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	uc := usecase.NewCategoryUsecase(mockRepo)

	ctx := context.Background()
	img := "https://example.com/categories/science.jpg"
	req := usecase.CreateCategoryInput{
		Name:     "Sains & Teknologi",
		ImageURL: &img,
	}
	expectedSlug := "sains-and-teknologi"

	mockRepo.On("ExistsBySlug", ctx, expectedSlug).Return(false, nil)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(cat *entity.Category) bool {
		return cat.Name == req.Name && cat.Slug == expectedSlug && cat.ImageURL != nil && *cat.ImageURL == img
	})).Run(func(args mock.Arguments) {
		cat := args.Get(1).(*entity.Category)
		cat.ID = 1
		cat.CreatedAt = time.Now()
		cat.UpdatedAt = time.Now()
	}).Return(nil)

	res, err := uc.Create(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, uint(1), res.ID)
	assert.Equal(t, "Sains & Teknologi", res.Name)
	assert.Equal(t, expectedSlug, res.Slug)
	assert.Equal(t, &img, res.ImageURL)
	assert.Equal(t, int64(0), res.TotalBooks)
	mockRepo.AssertExpectations(t)
}

func TestCategoryUsecase_Create_SlugConflict(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	uc := usecase.NewCategoryUsecase(mockRepo)

	ctx := context.Background()
	req := usecase.CreateCategoryInput{Name: "Komik"}

	mockRepo.On("ExistsBySlug", ctx, "komik").Return(true, nil)

	res, err := uc.Create(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, res)
	assert.ErrorIs(t, err, entity.ErrCategorySlugConflict)
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	mockRepo.AssertExpectations(t)
}

func TestCategoryUsecase_FindAll_Success(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	uc := usecase.NewCategoryUsecase(mockRepo)

	ctx := context.Background()
	img := "https://example.com/fiction.jpg"
	categories := []entity.Category{
		{ID: 1, Name: "Fiksi", Slug: "fiksi", ImageURL: &img, TotalBooks: 5},
		{ID: 2, Name: "Sains", Slug: "sains", TotalBooks: 3},
	}

	mockRepo.On("FindAll", ctx).Return(categories, nil)

	res, err := uc.FindAll(ctx)

	assert.NoError(t, err)
	assert.Len(t, res, 2)
	assert.IsType(t, []usecase.CategoryList{}, res)
	assert.Equal(t, uint(1), res[0].ID)
	assert.Equal(t, "Fiksi", res[0].Name)
	assert.Equal(t, "fiksi", res[0].Slug)
	assert.Equal(t, &img, res[0].ImageURL)
	assert.Equal(t, int64(5), res[0].TotalBooks)
	assert.Equal(t, int64(3), res[1].TotalBooks)
	mockRepo.AssertExpectations(t)
}

func TestCategoryUsecase_FindAll_RepositoryError(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	uc := usecase.NewCategoryUsecase(mockRepo)

	ctx := context.Background()
	mockRepo.On("FindAll", ctx).Return(nil, errors.New("db connection failure"))

	res, err := uc.FindAll(ctx)

	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "failed to get categories")
	mockRepo.AssertExpectations(t)
}

func TestCategoryUsecase_FindBySlug_Success(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	uc := usecase.NewCategoryUsecase(mockRepo)

	ctx := context.Background()
	img := "https://example.com/fiction-popular.jpg"
	category := &entity.Category{
		ID:         1,
		Name:       "Fiksi Populer",
		Slug:       "fiksi-populer",
		ImageURL:   &img,
		TotalBooks: 12,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	mockRepo.On("FindBySlug", ctx, "fiksi-populer").Return(category, nil)

	res, err := uc.FindBySlug(ctx, "fiksi-populer")

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, uint(1), res.ID)
	assert.Equal(t, "Fiksi Populer", res.Name)
	assert.Equal(t, "fiksi-populer", res.Slug)
	assert.Equal(t, &img, res.ImageURL)
	assert.Equal(t, int64(12), res.TotalBooks)
	mockRepo.AssertExpectations(t)
}

func TestCategoryUsecase_FindBySlug_NotFound(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	uc := usecase.NewCategoryUsecase(mockRepo)

	ctx := context.Background()
	mockRepo.On("FindBySlug", ctx, "unknown").Return(nil, nil)

	res, err := uc.FindBySlug(ctx, "unknown")

	assert.Error(t, err)
	assert.Nil(t, res)
	assert.ErrorIs(t, err, entity.ErrCategoryNotFound)
	mockRepo.AssertExpectations(t)
}

func TestCategoryUsecase_Update_Success(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	uc := usecase.NewCategoryUsecase(mockRepo)

	ctx := context.Background()
	existingCat := &entity.Category{
		ID:        1,
		Name:      "Fiksi",
		Slug:      "fiksi",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	newImg := "https://example.com/scifi.jpg"
	req := usecase.UpdateCategoryInput{
		Name:     "Fiksi Ilmiah",
		ImageURL: &newImg,
	}
	expectedSlug := "fiksi-ilmiah"

	mockRepo.On("FindByID", ctx, uint(1)).Return(existingCat, nil)
	mockRepo.On("ExistsBySlug", ctx, expectedSlug, uint(1)).Return(false, nil)
	mockRepo.On("Update", ctx, mock.MatchedBy(func(cat *entity.Category) bool {
		return cat.ID == 1 && cat.Name == "Fiksi Ilmiah" && cat.Slug == expectedSlug && cat.ImageURL != nil && *cat.ImageURL == newImg
	})).Return(nil)

	res, err := uc.Update(ctx, 1, req)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, uint(1), res.ID)
	assert.Equal(t, "Fiksi Ilmiah", res.Name)
	assert.Equal(t, expectedSlug, res.Slug)
	assert.Equal(t, &newImg, res.ImageURL)
	mockRepo.AssertExpectations(t)
}

func TestCategoryUsecase_Update_NotFound(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	uc := usecase.NewCategoryUsecase(mockRepo)

	ctx := context.Background()
	req := usecase.UpdateCategoryInput{Name: "Komik Baru"}

	mockRepo.On("FindByID", ctx, uint(99)).Return(nil, nil)

	res, err := uc.Update(ctx, 99, req)

	assert.Error(t, err)
	assert.Nil(t, res)
	assert.ErrorIs(t, err, entity.ErrCategoryNotFound)
	mockRepo.AssertNotCalled(t, "ExistsBySlug", mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	mockRepo.AssertExpectations(t)
}

func TestCategoryUsecase_Update_SlugConflict(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	uc := usecase.NewCategoryUsecase(mockRepo)

	ctx := context.Background()
	existingCat := &entity.Category{
		ID:   1,
		Name: "Fiksi",
		Slug: "fiksi",
	}

	req := usecase.UpdateCategoryInput{Name: "Sains"}
	expectedSlug := "sains"

	mockRepo.On("FindByID", ctx, uint(1)).Return(existingCat, nil)
	mockRepo.On("ExistsBySlug", ctx, expectedSlug, uint(1)).Return(true, nil)

	res, err := uc.Update(ctx, 1, req)

	assert.Error(t, err)
	assert.Nil(t, res)
	assert.ErrorIs(t, err, entity.ErrCategorySlugConflict)
	mockRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	mockRepo.AssertExpectations(t)
}

func TestCategoryUsecase_Delete_Success(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	uc := usecase.NewCategoryUsecase(mockRepo)

	ctx := context.Background()
	existingCat := &entity.Category{ID: 1, Name: "Fiksi"}

	mockRepo.On("FindByID", ctx, uint(1)).Return(existingCat, nil)
	mockRepo.On("Delete", ctx, uint(1)).Return(nil)

	err := uc.Delete(ctx, 1)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestCategoryUsecase_Delete_NotFound(t *testing.T) {
	mockRepo := new(MockCategoryRepository)
	uc := usecase.NewCategoryUsecase(mockRepo)

	ctx := context.Background()

	mockRepo.On("FindByID", ctx, uint(99)).Return(nil, nil)

	err := uc.Delete(ctx, 99)

	assert.Error(t, err)
	assert.ErrorIs(t, err, entity.ErrCategoryNotFound)
	mockRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	mockRepo.AssertExpectations(t)
}
