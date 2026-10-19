package test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/cache"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type sampleData struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestNoopCache(t *testing.T) {
	c := cache.NewNoopCache()
	ctx := context.Background()

	t.Run("get returns ErrCacheMiss", func(t *testing.T) {
		var dest sampleData
		err := c.Get(ctx, "any-key", &dest)
		assert.ErrorIs(t, err, cache.ErrCacheMiss)
	})

	t.Run("set does not return error", func(t *testing.T) {
		err := c.Set(ctx, "any-key", sampleData{ID: 1, Name: "Book"}, time.Minute)
		assert.NoError(t, err)
	})

	t.Run("delete does not return error", func(t *testing.T) {
		err := c.Delete(ctx, "any-key")
		assert.NoError(t, err)
	})

	t.Run("delete prefix does not return error", func(t *testing.T) {
		err := c.DeletePrefix(ctx, "prefix:")
		assert.NoError(t, err)
	})

	t.Run("close does not return error", func(t *testing.T) {
		assert.NoError(t, c.Close())
	})
}

func TestRedisCache_DisabledOrUnreachable(t *testing.T) {
	t.Run("disabled redis returns noop cache without error", func(t *testing.T) {
		cfg := cache.Config{
			Enabled: false,
			Host:    "localhost",
		}
		c, err := cache.NewRedisCache(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, c)

		var dest sampleData
		assert.ErrorIs(t, c.Get(context.Background(), "test", &dest), cache.ErrCacheMiss)
	})

	t.Run("unreachable redis address falls back to noop cache without error", func(t *testing.T) {
		cfg := cache.Config{
			Enabled: true,
			Host:    "127.0.0.1",
			Port:    "1", // Unreachable port
		}
		c, err := cache.NewRedisCache(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, c)

		var dest sampleData
		assert.ErrorIs(t, c.Get(context.Background(), "test", &dest), cache.ErrCacheMiss)
	})
}

type MockCache struct {
	mock.Mock
}

func (m *MockCache) Get(ctx context.Context, key string, dest any) error {
	args := m.Called(ctx, key, dest)
	return args.Error(0)
}

func (m *MockCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCache) Delete(ctx context.Context, keys ...string) error {
	args := m.Called(ctx, keys)
	return args.Error(0)
}

func (m *MockCache) DeletePrefix(ctx context.Context, prefix string) error {
	args := m.Called(ctx, prefix)
	return args.Error(0)
}

func (m *MockCache) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestCategoryUsecase_Caching(t *testing.T) {
	ctx := context.Background()

	t.Run("FindAll returns cached categories on cache hit", func(t *testing.T) {
		mockRepo := new(MockCategoryRepository)
		mockCache := new(MockCache)

		uc := usecase.NewCategoryUsecaseWithCache(mockRepo, nil, mockCache)

		cachedList := []usecase.CategoryList{
			{ID: 1, Name: "Fiksi", Slug: "fiksi"},
		}

		mockCache.On("Get", ctx, "categories:all", mock.Anything).Run(func(args mock.Arguments) {
			dest := args.Get(2).(*[]usecase.CategoryList)
			*dest = cachedList
		}).Return(nil).Once()

		result, err := uc.FindAll(ctx)
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Fiksi", result[0].Name)
		mockRepo.AssertNotCalled(t, "FindAll")
		mockCache.AssertExpectations(t)
	})

	t.Run("Create invalidates categories cache", func(t *testing.T) {
		mockRepo := new(MockCategoryRepository)
		mockCache := new(MockCache)

		uc := usecase.NewCategoryUsecaseWithCache(mockRepo, nil, mockCache)

		mockRepo.On("ExistsBySlug", ctx, "komik").Return(false, nil).Once()
		mockRepo.On("Create", ctx, mock.Anything).Return(nil).Once()
		mockCache.On("DeletePrefix", ctx, "categories:").Return(nil).Once()

		res, err := uc.Create(ctx, usecase.CreateCategoryInput{Name: "Komik"})
		assert.NoError(t, err)
		assert.NotNil(t, res)
		mockCache.AssertExpectations(t)
	})
}

func TestBookUsecase_Caching(t *testing.T) {
	ctx := context.Background()

	t.Run("FindBySlug returns cached book on cache hit", func(t *testing.T) {
		mockBookRepo := new(MockBookRepository)
		mockCache := new(MockCache)

		uc := usecase.NewBookUsecaseWithCache(mockBookRepo, nil, nil, nil, nil, mockCache)

		cachedBook := usecase.BookDetailResult{
			ID:    10,
			Title: "Clean Code",
			Slug:  "clean-code",
		}

		mockCache.On("Get", ctx, "books:slug:clean-code", mock.Anything).Run(func(args mock.Arguments) {
			dest := args.Get(2).(*usecase.BookDetailResult)
			*dest = cachedBook
		}).Return(nil).Once()

		result, err := uc.FindBySlug(ctx, "clean-code")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Clean Code", result.Title)
		mockBookRepo.AssertNotCalled(t, "FindBySlug")
		mockCache.AssertExpectations(t)
	})

	t.Run("FindAll returns cached book list on cache hit", func(t *testing.T) {
		mockBookRepo := new(MockBookRepository)
		mockCache := new(MockCache)

		uc := usecase.NewBookUsecaseWithCache(mockBookRepo, nil, nil, nil, nil, mockCache)

		filter := repository.BookFilter{CategorySlug: "programming"}
		pq := pagination.Query{Page: 1, PerPage: 10}

		cachedItems := []usecase.BookListItem{
			{ID: 1, Title: "Refactoring", Slug: "refactoring"},
		}
		cachedMeta := pagination.Meta{CurrentPage: 1, PerPage: 10, TotalItems: 1, TotalPages: 1}

		mockCache.On("Get", ctx, mock.MatchedBy(func(k string) bool {
			return strings.HasPrefix(k, "books:list:") && strings.Contains(k, "cat=programming")
		}), mock.Anything).Run(func(args mock.Arguments) {
			dest := args.Get(2)
			b, _ := json.Marshal(map[string]any{
				"items": cachedItems,
				"meta":  cachedMeta,
			})
			_ = json.Unmarshal(b, dest)
		}).Return(nil).Once()

		items, meta, err := uc.FindAll(ctx, filter, pq)
		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, "Refactoring", items[0].Title)
		assert.Equal(t, int64(1), meta.TotalItems)
		mockBookRepo.AssertNotCalled(t, "FindAll")
		mockCache.AssertExpectations(t)
	})

	t.Run("FindAll queries repository and saves cache on cache miss", func(t *testing.T) {
		mockBookRepo := new(MockBookRepository)
		mockCache := new(MockCache)

		uc := usecase.NewBookUsecaseWithCache(mockBookRepo, nil, nil, nil, nil, mockCache)

		filter := repository.BookFilter{CategorySlug: "programming"}
		pq := pagination.Query{Page: 1, PerPage: 10}

		books := []entity.Book{
			{Model: gorm.Model{ID: 1}, Title: "Refactoring", Slug: "refactoring", Price: 100000},
		}

		mockCache.On("Get", ctx, mock.MatchedBy(func(k string) bool {
			return strings.HasPrefix(k, "books:list:")
		}), mock.Anything).Return(cache.ErrCacheMiss).Once()

		expectedFilter := filter
		expectedFilter.Limit = 10
		expectedFilter.Offset = 0
		mockBookRepo.On("FindAll", ctx, expectedFilter).Return(books, int64(1), nil).Once()

		mockCache.On("Set", ctx, mock.MatchedBy(func(k string) bool {
			return strings.HasPrefix(k, "books:list:")
		}), mock.Anything, 5*time.Minute).Return(nil).Once()

		items, meta, err := uc.FindAll(ctx, filter, pq)
		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, "Refactoring", items[0].Title)
		assert.Equal(t, int64(1), meta.TotalItems)
		mockBookRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})
}
