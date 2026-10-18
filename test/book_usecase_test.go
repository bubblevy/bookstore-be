package test

import (
	"context"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockBookRepository struct {
	mock.Mock
}

func (m *MockBookRepository) Create(ctx context.Context, book *entity.Book) error {
	return m.Called(ctx, book).Error(0)
}

func (m *MockBookRepository) FindByID(ctx context.Context, id uint) (*entity.Book, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Book), args.Error(1)
}

func (m *MockBookRepository) FindBySlug(ctx context.Context, slug string) (*entity.Book, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Book), args.Error(1)
}

func (m *MockBookRepository) FindAll(ctx context.Context, filter repository.BookFilter) ([]entity.Book, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]entity.Book), args.Get(1).(int64), args.Error(2)
}

func (m *MockBookRepository) FindByIDs(ctx context.Context, ids []uint) ([]entity.Book, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.Book), args.Error(1)
}

func (m *MockBookRepository) CountExistingIDs(ctx context.Context, ids []uint) (int64, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockBookRepository) Update(ctx context.Context, book *entity.Book) error {
	return m.Called(ctx, book).Error(0)
}

func (m *MockBookRepository) Delete(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockBookRepository) ExistsBySlug(ctx context.Context, slug string, excludeID ...uint) (bool, error) {
	var args mock.Arguments
	if len(excludeID) > 0 {
		args = m.Called(ctx, slug, excludeID[0])
	} else {
		args = m.Called(ctx, slug)
	}
	return args.Bool(0), args.Error(1)
}

func (m *MockBookRepository) ExistsByISBN(ctx context.Context, isbn string, excludeID ...uint) (bool, error) {
	var args mock.Arguments
	if len(excludeID) > 0 {
		args = m.Called(ctx, isbn, excludeID[0])
	} else {
		args = m.Called(ctx, isbn)
	}
	return args.Bool(0), args.Error(1)
}

func (m *MockBookRepository) UpdateStock(ctx context.Context, id uint, quantity int) error {
	return m.Called(ctx, id, quantity).Error(0)
}

type MockCategoryRepositoryForBook struct {
	mock.Mock
}

func (m *MockCategoryRepositoryForBook) FindByID(ctx context.Context, id uint) (*entity.Category, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Category), args.Error(1)
}

func (m *MockCategoryRepositoryForBook) Create(ctx context.Context, c *entity.Category) error {
	return nil
}

func (m *MockCategoryRepositoryForBook) FindBySlug(ctx context.Context, s string) (*entity.Category, error) {
	return nil, nil
}

func (m *MockCategoryRepositoryForBook) FindAll(ctx context.Context) ([]entity.Category, error) {
	return nil, nil
}

func (m *MockCategoryRepositoryForBook) Update(ctx context.Context, c *entity.Category) error {
	return nil
}

func (m *MockCategoryRepositoryForBook) Delete(ctx context.Context, id uint) error {
	return nil
}

func (m *MockCategoryRepositoryForBook) ExistsBySlug(ctx context.Context, s string, e ...uint) (bool, error) {
	return false, nil
}

type MockAuthorRepositoryForBook struct {
	mock.Mock
}

func (m *MockAuthorRepositoryForBook) FindByID(ctx context.Context, id uint) (*entity.Author, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Author), args.Error(1)
}

func (m *MockAuthorRepositoryForBook) Create(ctx context.Context, a *entity.Author) error {
	return nil
}

func (m *MockAuthorRepositoryForBook) FindBySlug(ctx context.Context, s string) (*entity.Author, error) {
	return nil, nil
}

func (m *MockAuthorRepositoryForBook) FindAll(ctx context.Context, l, o int) ([]entity.Author, int64, error) {
	return nil, 0, nil
}

func (m *MockAuthorRepositoryForBook) Update(ctx context.Context, a *entity.Author) error {
	return nil
}

func (m *MockAuthorRepositoryForBook) Delete(ctx context.Context, id uint) error {
	return nil
}

func (m *MockAuthorRepositoryForBook) ExistsBySlug(ctx context.Context, s string, e ...uint) (bool, error) {
	return false, nil
}

type MockPublisherRepositoryForBook struct {
	mock.Mock
}

func (m *MockPublisherRepositoryForBook) FindByID(ctx context.Context, id uint) (*entity.Publisher, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Publisher), args.Error(1)
}

func (m *MockPublisherRepositoryForBook) Create(ctx context.Context, p *entity.Publisher) error {
	return nil
}

func (m *MockPublisherRepositoryForBook) FindBySlug(ctx context.Context, s string) (*entity.Publisher, error) {
	return nil, nil
}

func (m *MockPublisherRepositoryForBook) FindAll(ctx context.Context) ([]entity.Publisher, error) {
	return nil, nil
}

func (m *MockPublisherRepositoryForBook) Update(ctx context.Context, p *entity.Publisher) error {
	return nil
}

func (m *MockPublisherRepositoryForBook) Delete(ctx context.Context, id uint) error {
	return nil
}

func (m *MockPublisherRepositoryForBook) ExistsBySlug(ctx context.Context, s string, e ...uint) (bool, error) {
	return false, nil
}

func TestBookUsecase_Create_Success(t *testing.T) {
	mockBookRepo := new(MockBookRepository)
	mockCatRepo := new(MockCategoryRepositoryForBook)
	mockAuthRepo := new(MockAuthorRepositoryForBook)
	mockPubRepo := new(MockPublisherRepositoryForBook)
	uc := usecase.NewBookUsecase(mockBookRepo, mockCatRepo, mockAuthRepo, mockPubRepo)
	ctx := context.Background()

	isbn := "9789799731234"
	origPrice := 150000
	req := usecase.CreateBookInput{
		CategoryID:    1,
		AuthorID:      1,
		PublisherID:   1,
		Title:         "Bumi Manusia",
		ISBN:          &isbn,
		Price:         135000,
		OriginalPrice: &origPrice,
		WeightGrams:   450,
		Stock:         10,
		Language:      "Indonesia",
		Length:        20.0,
		Width:         13.5,
		Height:        2.0,
		Condition:     "100% Baru & Segel Penerbit",
	}

	cat := &entity.Category{Model: gorm.Model{ID: 1}, Name: "Fiksi", Slug: "fiksi"}
	auth := &entity.Author{Model: gorm.Model{ID: 1}, Name: "Pramoedya", Slug: "pramoedya"}
	pub := &entity.Publisher{Model: gorm.Model{ID: 1}, Name: "Gramedia", Slug: "gramedia"}

	mockCatRepo.On("FindByID", ctx, uint(1)).Return(cat, nil)
	mockAuthRepo.On("FindByID", ctx, uint(1)).Return(auth, nil)
	mockPubRepo.On("FindByID", ctx, uint(1)).Return(pub, nil)
	mockBookRepo.On("ExistsBySlug", ctx, "bumi-manusia").Return(false, nil)
	mockBookRepo.On("ExistsByISBN", ctx, isbn).Return(false, nil)
	mockBookRepo.On("Create", ctx, mock.MatchedBy(func(b *entity.Book) bool {
		return b.Title == "Bumi Manusia" && b.Slug == "bumi-manusia" && b.CategoryID == 1 &&
			b.Language == "Indonesia" && b.Length == 20.0 && b.Condition == "100% Baru & Segel Penerbit" &&
			b.OriginalPrice != nil && *b.OriginalPrice == 150000.0 &&
			b.DiscountPercentage != nil && *b.DiscountPercentage == 10
	})).Run(func(args mock.Arguments) {
		b := args.Get(1).(*entity.Book)
		b.ID = 1
		b.CreatedAt = time.Now()
		b.UpdatedAt = time.Now()
	}).Return(nil)

	res, err := uc.Create(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, uint(1), res.ID)
	assert.Equal(t, "bumi-manusia", res.Slug)
	assert.Equal(t, "Fiksi", res.Category.Name)
	assert.Equal(t, &origPrice, res.OriginalPrice)
	assert.Equal(t, 10, *res.DiscountPercentage)
	assert.Equal(t, "Indonesia", res.Language)
	assert.Equal(t, 20.0, res.Length)
	assert.Equal(t, 13.5, res.Width)
	assert.Equal(t, 2.0, res.Height)
	assert.Equal(t, "100% Baru & Segel Penerbit", res.Condition)
}

func TestBookUsecase_Create_AutoDiscountFromPercentageOnly(t *testing.T) {
	mockBookRepo := new(MockBookRepository)
	mockCatRepo := new(MockCategoryRepositoryForBook)
	mockAuthRepo := new(MockAuthorRepositoryForBook)
	mockPubRepo := new(MockPublisherRepositoryForBook)
	uc := usecase.NewBookUsecase(mockBookRepo, mockCatRepo, mockAuthRepo, mockPubRepo)
	ctx := context.Background()

	isbn := "9789799739999"
	discPct := 20
	req := usecase.CreateBookInput{
		CategoryID:         1,
		AuthorID:           1,
		PublisherID:        1,
		Title:              "Gadis Kretek",
		ISBN:               &isbn,
		Price:              100000,
		OriginalPrice:      nil,
		DiscountPercentage: &discPct,
		WeightGrams:        350,
		Stock:              10,
		Language:           "Indonesia",
		Length:             20.0,
		Width:              13.5,
		Height:             1.5,
		Condition:          "100% Baru & Segel Penerbit",
	}

	cat := &entity.Category{Model: gorm.Model{ID: 1}, Name: "Fiksi", Slug: "fiksi"}
	auth := &entity.Author{Model: gorm.Model{ID: 1}, Name: "Ratih Kumala", Slug: "ratih-kumala"}
	pub := &entity.Publisher{Model: gorm.Model{ID: 1}, Name: "Gramedia", Slug: "gramedia"}

	mockCatRepo.On("FindByID", ctx, uint(1)).Return(cat, nil)
	mockAuthRepo.On("FindByID", ctx, uint(1)).Return(auth, nil)
	mockPubRepo.On("FindByID", ctx, uint(1)).Return(pub, nil)
	mockBookRepo.On("ExistsBySlug", ctx, "gadis-kretek").Return(false, nil)
	mockBookRepo.On("ExistsByISBN", ctx, isbn).Return(false, nil)
	mockBookRepo.On("Create", ctx, mock.MatchedBy(func(b *entity.Book) bool {
		return b.Title == "Gadis Kretek" &&
			b.Price == 80000.0 &&
			b.OriginalPrice != nil && *b.OriginalPrice == 100000.0 &&
			b.DiscountPercentage != nil && *b.DiscountPercentage == 20
	})).Run(func(args mock.Arguments) {
		b := args.Get(1).(*entity.Book)
		b.ID = 2
		b.CreatedAt = time.Now()
		b.UpdatedAt = time.Now()
	}).Return(nil)

	res, err := uc.Create(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, 80000, res.Price)
	assert.NotNil(t, res.OriginalPrice)
	assert.Equal(t, 100000, *res.OriginalPrice)
	assert.NotNil(t, res.DiscountPercentage)
	assert.Equal(t, 20, *res.DiscountPercentage)
}

func TestBookUsecase_Create_CategoryNotFound(t *testing.T) {
	mockBookRepo := new(MockBookRepository)
	mockCatRepo := new(MockCategoryRepositoryForBook)
	mockAuthRepo := new(MockAuthorRepositoryForBook)
	mockPubRepo := new(MockPublisherRepositoryForBook)
	uc := usecase.NewBookUsecase(mockBookRepo, mockCatRepo, mockAuthRepo, mockPubRepo)
	ctx := context.Background()

	mockCatRepo.On("FindByID", ctx, uint(99)).Return(nil, nil)

	res, err := uc.Create(ctx, usecase.CreateBookInput{CategoryID: 99})
	assert.ErrorIs(t, err, entity.ErrBookCategoryNotFound)
	assert.Nil(t, res)
}

func TestBookUsecase_FindAll_Success(t *testing.T) {
	mockBookRepo := new(MockBookRepository)
	uc := usecase.NewBookUsecase(mockBookRepo, nil, nil, nil)
	ctx := context.Background()

	pq := pagination.Query{Page: 1, PerPage: 10}
	filter := repository.BookFilter{
		CategorySlug: "fiksi",
		Limit:        10,
		Offset:       0,
	}

	origPrice := 100000
	books := []entity.Book{
		{
			Model:         gorm.Model{ID: 1},
			Title:         "Bumi",
			Slug:          "bumi",
			Price:         95000,
			OriginalPrice: &origPrice,
			WeightGrams:   300,
			Stock:         15,
			IsActive:      true,
			Category:      entity.Category{Model: gorm.Model{ID: 1}, Name: "Fiksi", Slug: "fiksi"},
			Author:        entity.Author{Model: gorm.Model{ID: 2}, Name: "Tere Liye", Slug: "tere-liye"},
			Publisher:     entity.Publisher{Model: gorm.Model{ID: 3}, Name: "Gramedia", Slug: "gramedia"},
		},
	}

	mockBookRepo.On("FindAll", ctx, filter).Return(books, int64(1), nil)

	items, meta, err := uc.FindAll(ctx, repository.BookFilter{CategorySlug: "fiksi"}, pq)
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "Bumi", items[0].Title)
	assert.Equal(t, &origPrice, items[0].OriginalPrice)
	assert.Equal(t, 5, *items[0].DiscountPercentage)
	assert.Equal(t, int64(1), meta.TotalItems)
}

func TestBookUsecase_FindBySlug_Success(t *testing.T) {
	mockBookRepo := new(MockBookRepository)
	uc := usecase.NewBookUsecase(mockBookRepo, nil, nil, nil)
	ctx := context.Background()

	book := &entity.Book{
		Model: gorm.Model{
			ID:        1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Title:     "Bumi",
		Slug:      "bumi",
		Price:     95000,
		Language:  "Indonesia",
		Length:    20.0,
		Width:     13.5,
		Height:    2.0,
		Condition: "100% Baru & Segel Penerbit",
		Category:  entity.Category{Model: gorm.Model{ID: 1}, Name: "Fiksi", Slug: "fiksi"},
		Author:    entity.Author{Model: gorm.Model{ID: 2}, Name: "Tere Liye", Slug: "tere-liye"},
		Publisher: entity.Publisher{Model: gorm.Model{ID: 3}, Name: "Gramedia", Slug: "gramedia"},
		IsActive:  true,
	}

	mockBookRepo.On("FindBySlug", ctx, "bumi").Return(book, nil)

	res, err := uc.FindBySlug(ctx, "bumi")
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "bumi", res.Slug)
	assert.Equal(t, "Indonesia", res.Language)
	assert.Equal(t, 20.0, res.Length)
	assert.Equal(t, 13.5, res.Width)
	assert.Equal(t, 2.0, res.Height)
	assert.Equal(t, "100% Baru & Segel Penerbit", res.Condition)
}

func TestBookUsecase_Delete_Success(t *testing.T) {
	mockBookRepo := new(MockBookRepository)
	uc := usecase.NewBookUsecase(mockBookRepo, nil, nil, nil)
	ctx := context.Background()

	book := &entity.Book{Model: gorm.Model{ID: 1}}
	mockBookRepo.On("FindByID", ctx, uint(1)).Return(book, nil)
	mockBookRepo.On("Delete", ctx, uint(1)).Return(nil)

	err := uc.Delete(ctx, 1)
	assert.NoError(t, err)
}
