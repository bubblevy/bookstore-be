package test

import (
	"context"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPublisherRepository struct {
	mock.Mock
}

func (m *MockPublisherRepository) Create(ctx context.Context, publisher *entity.Publisher) error {
	args := m.Called(ctx, publisher)
	return args.Error(0)
}

func (m *MockPublisherRepository) FindByID(ctx context.Context, id uint) (*entity.Publisher, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Publisher), args.Error(1)
}

func (m *MockPublisherRepository) FindBySlug(ctx context.Context, slug string) (*entity.Publisher, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Publisher), args.Error(1)
}

func (m *MockPublisherRepository) FindAll(ctx context.Context) ([]entity.Publisher, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.Publisher), args.Error(1)
}

func (m *MockPublisherRepository) Update(ctx context.Context, publisher *entity.Publisher) error {
	args := m.Called(ctx, publisher)
	return args.Error(0)
}

func (m *MockPublisherRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPublisherRepository) ExistsBySlug(ctx context.Context, slug string, excludeID ...uint) (bool, error) {
	var args mock.Arguments
	if len(excludeID) > 0 {
		args = m.Called(ctx, slug, excludeID[0])
	} else {
		args = m.Called(ctx, slug)
	}
	return args.Bool(0), args.Error(1)
}

func TestPublisherUsecase_Create_Success(t *testing.T) {
	mockRepo := new(MockPublisherRepository)
	uc := usecase.NewPublisherUsecase(mockRepo)

	ctx := context.Background()
	req := usecase.CreatePublisherInput{Name: "Gramedia Pustaka Utama"}
	expectedSlug := "gramedia-pustaka-utama"

	mockRepo.On("ExistsBySlug", ctx, expectedSlug).Return(false, nil)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(p *entity.Publisher) bool {
		return p.Name == req.Name && p.Slug == expectedSlug
	})).Run(func(args mock.Arguments) {
		p := args.Get(1).(*entity.Publisher)
		p.ID = 1
		p.CreatedAt = time.Now()
		p.UpdatedAt = time.Now()
	}).Return(nil)

	res, err := uc.Create(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, uint(1), res.ID)
	assert.Equal(t, expectedSlug, res.Slug)
	mockRepo.AssertExpectations(t)
}

func TestPublisherUsecase_Create_SlugConflict(t *testing.T) {
	mockRepo := new(MockPublisherRepository)
	uc := usecase.NewPublisherUsecase(mockRepo)

	ctx := context.Background()
	req := usecase.CreatePublisherInput{Name: "Mizan"}

	mockRepo.On("ExistsBySlug", ctx, "mizan").Return(true, nil)

	res, err := uc.Create(ctx, req)
	assert.ErrorIs(t, err, entity.ErrPublisherSlugConflict)
	assert.Nil(t, res)
	mockRepo.AssertExpectations(t)
}

func TestPublisherUsecase_FindAll_Success(t *testing.T) {
	mockRepo := new(MockPublisherRepository)
	uc := usecase.NewPublisherUsecase(mockRepo)

	ctx := context.Background()
	publishers := []entity.Publisher{
		{ID: 1, Name: "Gramedia", Slug: "gramedia"},
		{ID: 2, Name: "Mizan", Slug: "mizan"},
	}

	mockRepo.On("FindAll", ctx).Return(publishers, nil)

	res, err := uc.FindAll(ctx)
	assert.NoError(t, err)
	assert.Len(t, res, 2)
	assert.Equal(t, "Gramedia", res[0].Name)
	mockRepo.AssertExpectations(t)
}

func TestPublisherUsecase_FindBySlug_NotFound(t *testing.T) {
	mockRepo := new(MockPublisherRepository)
	uc := usecase.NewPublisherUsecase(mockRepo)

	ctx := context.Background()
	mockRepo.On("FindBySlug", ctx, "unknown").Return(nil, nil)

	res, err := uc.FindBySlug(ctx, "unknown")
	assert.ErrorIs(t, err, entity.ErrPublisherNotFound)
	assert.Nil(t, res)
	mockRepo.AssertExpectations(t)
}

func TestPublisherUsecase_Update_Success(t *testing.T) {
	mockRepo := new(MockPublisherRepository)
	uc := usecase.NewPublisherUsecase(mockRepo)

	ctx := context.Background()
	existing := &entity.Publisher{ID: 1, Name: "Gramedia", Slug: "gramedia"}
	req := usecase.UpdatePublisherInput{Name: "Gramedia Digital"}
	expectedSlug := "gramedia-digital"

	mockRepo.On("FindByID", ctx, uint(1)).Return(existing, nil)
	mockRepo.On("ExistsBySlug", ctx, expectedSlug, uint(1)).Return(false, nil)
	mockRepo.On("Update", ctx, mock.MatchedBy(func(p *entity.Publisher) bool {
		return p.ID == 1 && p.Name == "Gramedia Digital" && p.Slug == expectedSlug
	})).Return(nil)

	res, err := uc.Update(ctx, 1, req)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, expectedSlug, res.Slug)
	mockRepo.AssertExpectations(t)
}

func TestPublisherUsecase_Delete_Success(t *testing.T) {
	mockRepo := new(MockPublisherRepository)
	uc := usecase.NewPublisherUsecase(mockRepo)

	ctx := context.Background()
	existing := &entity.Publisher{ID: 1, Name: "Mizan"}

	mockRepo.On("FindByID", ctx, uint(1)).Return(existing, nil)
	mockRepo.On("Delete", ctx, uint(1)).Return(nil)

	err := uc.Delete(ctx, 1)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
