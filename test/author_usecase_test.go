package test

import (
	"context"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockAuthorRepository struct {
	mock.Mock
}

func (m *MockAuthorRepository) Create(ctx context.Context, author *entity.Author) error {
	args := m.Called(ctx, author)
	return args.Error(0)
}

func (m *MockAuthorRepository) FindByID(ctx context.Context, id uint) (*entity.Author, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Author), args.Error(1)
}

func (m *MockAuthorRepository) FindBySlug(ctx context.Context, slug string) (*entity.Author, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Author), args.Error(1)
}

func (m *MockAuthorRepository) FindAll(ctx context.Context, limit, offset int) ([]entity.Author, int64, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]entity.Author), args.Get(1).(int64), args.Error(2)
}

func (m *MockAuthorRepository) Update(ctx context.Context, author *entity.Author) error {
	args := m.Called(ctx, author)
	return args.Error(0)
}

func (m *MockAuthorRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAuthorRepository) ExistsBySlug(ctx context.Context, slug string, excludeID ...uint) (bool, error) {
	var args mock.Arguments
	if len(excludeID) > 0 {
		args = m.Called(ctx, slug, excludeID[0])
	} else {
		args = m.Called(ctx, slug)
	}
	return args.Bool(0), args.Error(1)
}

func (m *MockAuthorRepository) FindBooksByAuthorID(ctx context.Context, authorID uint, limit, offset int) ([]entity.Book, int64, error) {
	args := m.Called(ctx, authorID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]entity.Book), args.Get(1).(int64), args.Error(2)
}

func TestAuthorUsecase_Create_Success(t *testing.T) {
	mockRepo := new(MockAuthorRepository)
	uc := usecase.NewAuthorUsecase(mockRepo)

	ctx := context.Background()
	bio := "Penulis fiksi terkemuka"
	req := usecase.CreateAuthorInput{Name: "Tere Liye", Bio: &bio}
	expectedSlug := "tere-liye"

	mockRepo.On("ExistsBySlug", ctx, expectedSlug).Return(false, nil)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(a *entity.Author) bool {
		return a.Name == req.Name && a.Slug == expectedSlug && a.Bio == req.Bio
	})).Run(func(args mock.Arguments) {
		a := args.Get(1).(*entity.Author)
		a.ID = 1
		a.CreatedAt = time.Now()
		a.UpdatedAt = time.Now()
	}).Return(nil)

	res, err := uc.Create(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, uint(1), res.ID)
	assert.Equal(t, expectedSlug, res.Slug)
	assert.Equal(t, &bio, res.Bio)
	mockRepo.AssertExpectations(t)
}

func TestAuthorUsecase_Create_SlugConflict(t *testing.T) {
	mockRepo := new(MockAuthorRepository)
	uc := usecase.NewAuthorUsecase(mockRepo)

	ctx := context.Background()
	req := usecase.CreateAuthorInput{Name: "Tere Liye"}

	mockRepo.On("ExistsBySlug", ctx, "tere-liye").Return(true, nil)

	res, err := uc.Create(ctx, req)
	assert.ErrorIs(t, err, entity.ErrAuthorSlugConflict)
	assert.Nil(t, res)
	mockRepo.AssertExpectations(t)
}

func TestAuthorUsecase_FindAll_Success(t *testing.T) {
	mockRepo := new(MockAuthorRepository)
	uc := usecase.NewAuthorUsecase(mockRepo)

	ctx := context.Background()
	pq := pagination.Query{Page: 1, PerPage: 10}

	authors := []entity.Author{
		{Model: gorm.Model{ID: 1}, Name: "Andrea Hirata", Slug: "andrea-hirata"},
		{Model: gorm.Model{ID: 2}, Name: "Tere Liye", Slug: "tere-liye"},
	}

	mockRepo.On("FindAll", ctx, pq.PerPage, pq.Offset()).Return(authors, int64(2), nil)

	items, meta, err := uc.FindAll(ctx, pq)
	assert.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, "Andrea Hirata", items[0].Name)
	assert.Equal(t, int64(2), meta.TotalItems)
	assert.Equal(t, 1, meta.CurrentPage)
	assert.Equal(t, 1, meta.TotalPages)
	mockRepo.AssertExpectations(t)
}

func TestAuthorUsecase_FindBySlug_Success(t *testing.T) {
	mockRepo := new(MockAuthorRepository)
	uc := usecase.NewAuthorUsecase(mockRepo)

	ctx := context.Background()
	bookPq := pagination.Query{Page: 1, PerPage: 10}
	bio := "Biografi Tere Liye"

	author := &entity.Author{
		Model: gorm.Model{ID: 1},
		Name:  "Tere Liye",
		Slug:  "tere-liye",
		Bio:   &bio,
	}

	cover := "bumi.jpg"
	books := []entity.Book{
		{
			ID:         101,
			AuthorID:   1,
			Title:      "Bumi",
			Slug:       "bumi",
			Price:      95000,
			CoverImage: &cover,
			IsActive:   true,
		},
	}

	mockRepo.On("FindBySlug", ctx, "tere-liye").Return(author, nil)
	mockRepo.On("FindBooksByAuthorID", ctx, uint(1), bookPq.PerPage, bookPq.Offset()).Return(books, int64(1), nil)

	res, err := uc.FindBySlug(ctx, "tere-liye", bookPq)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, uint(1), res.ID)
	assert.Equal(t, "Tere Liye", res.Name)
	assert.Len(t, res.Books, 1)
	assert.Equal(t, "Bumi", res.Books[0].Title)
	assert.Equal(t, &cover, res.Books[0].Cover)
	assert.Equal(t, int64(1), res.BooksMeta.TotalItems)
	mockRepo.AssertExpectations(t)
}

func TestAuthorUsecase_FindBySlug_NotFound(t *testing.T) {
	mockRepo := new(MockAuthorRepository)
	uc := usecase.NewAuthorUsecase(mockRepo)

	ctx := context.Background()
	bookPq := pagination.Query{Page: 1, PerPage: 10}

	mockRepo.On("FindBySlug", ctx, "unknown").Return(nil, nil)

	res, err := uc.FindBySlug(ctx, "unknown", bookPq)
	assert.ErrorIs(t, err, entity.ErrAuthorNotFound)
	assert.Nil(t, res)
	mockRepo.AssertExpectations(t)
}

func TestAuthorUsecase_Update_Success(t *testing.T) {
	mockRepo := new(MockAuthorRepository)
	uc := usecase.NewAuthorUsecase(mockRepo)

	ctx := context.Background()
	existing := &entity.Author{Model: gorm.Model{ID: 1}, Name: "Tere", Slug: "tere"}
	newBio := "Bio baru"
	req := usecase.UpdateAuthorInput{Name: "Tere Liye", Bio: &newBio}
	expectedSlug := "tere-liye"

	mockRepo.On("FindByID", ctx, uint(1)).Return(existing, nil)
	mockRepo.On("ExistsBySlug", ctx, expectedSlug, uint(1)).Return(false, nil)
	mockRepo.On("Update", ctx, mock.MatchedBy(func(a *entity.Author) bool {
		return a.ID == 1 && a.Name == req.Name && a.Slug == expectedSlug && a.Bio == req.Bio
	})).Return(nil)

	res, err := uc.Update(ctx, 1, req)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, expectedSlug, res.Slug)
	mockRepo.AssertExpectations(t)
}

func TestAuthorUsecase_Delete_Success(t *testing.T) {
	mockRepo := new(MockAuthorRepository)
	uc := usecase.NewAuthorUsecase(mockRepo)

	ctx := context.Background()
	existing := &entity.Author{Model: gorm.Model{ID: 1}, Name: "Tere Liye"}

	mockRepo.On("FindByID", ctx, uint(1)).Return(existing, nil)
	mockRepo.On("Delete", ctx, uint(1)).Return(nil)

	err := uc.Delete(ctx, 1)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuthorUsecase_Delete_NotFound(t *testing.T) {
	mockRepo := new(MockAuthorRepository)
	uc := usecase.NewAuthorUsecase(mockRepo)

	ctx := context.Background()
	mockRepo.On("FindByID", ctx, uint(999)).Return(nil, nil)

	err := uc.Delete(ctx, 999)
	assert.ErrorIs(t, err, entity.ErrAuthorNotFound)
	mockRepo.AssertExpectations(t)
}
