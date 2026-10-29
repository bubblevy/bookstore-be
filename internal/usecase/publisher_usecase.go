package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/cache"
	"github.com/gosimple/slug"
)

type CreatePublisherInput struct {
	Name string
}

type UpdatePublisherInput struct {
	Name string
}

type PublisherList struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type PublisherResult struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PublisherUsecase interface {
	Create(ctx context.Context, req CreatePublisherInput) (*PublisherResult, error)
	FindAll(ctx context.Context) ([]PublisherList, error)
	FindBySlug(ctx context.Context, slug string) (*PublisherResult, error)
	Update(ctx context.Context, id uint, req UpdatePublisherInput) (*PublisherResult, error)
	Delete(ctx context.Context, id uint) error
}

type publisherUsecase struct {
	publisherRepo repository.PublisherRepository
	cacheSvc      cache.Cache
}

func NewPublisherUsecase(publisherRepo repository.PublisherRepository, cacheSvc ...cache.Cache) PublisherUsecase {
	var c cache.Cache
	if len(cacheSvc) > 0 {
		c = cacheSvc[0]
	}
	return &publisherUsecase{publisherRepo: publisherRepo, cacheSvc: c}
}

func (u *publisherUsecase) toResponse(pub *entity.Publisher) *PublisherResult {
	return &PublisherResult{
		ID:        pub.ID,
		Name:      pub.Name,
		Slug:      pub.Slug,
		CreatedAt: pub.CreatedAt,
		UpdatedAt: pub.UpdatedAt,
	}
}

func (u *publisherUsecase) Create(ctx context.Context, req CreatePublisherInput) (*PublisherResult, error) {
	publisherSlug := slug.Make(req.Name)
	exists, err := u.publisherRepo.ExistsBySlug(ctx, publisherSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to check publisher slug: %w", err)
	}
	if exists {
		return nil, entity.ErrPublisherSlugConflict
	}

	publisher := &entity.Publisher{
		Name: req.Name,
		Slug: publisherSlug,
	}

	if err := u.publisherRepo.Create(ctx, publisher); err != nil {
		return nil, fmt.Errorf("failed to create publisher: %w", err)
	}

	if u.cacheSvc != nil {
		_ = u.cacheSvc.DeletePrefix(ctx, "publishers:")
	}

	return u.toResponse(publisher), nil
}

func (u *publisherUsecase) FindAll(ctx context.Context) ([]PublisherList, error) {
	cacheKey := "publishers:all"
	if u.cacheSvc != nil {
		var cached []PublisherList
		if err := u.cacheSvc.Get(ctx, cacheKey, &cached); err == nil && len(cached) > 0 {
			return cached, nil
		}
	}

	publishers, err := u.publisherRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get publishers: %w", err)
	}

	responses := make([]PublisherList, 0, len(publishers))
	for _, pub := range publishers {
		responses = append(responses, PublisherList{
			ID:   pub.ID,
			Name: pub.Name,
			Slug: pub.Slug,
		})
	}

	if u.cacheSvc != nil && len(responses) > 0 {
		_ = u.cacheSvc.Set(ctx, cacheKey, responses, 30*time.Minute)
	}

	return responses, nil
}

func (u *publisherUsecase) FindBySlug(ctx context.Context, slugStr string) (*PublisherResult, error) {
	publisher, err := u.publisherRepo.FindBySlug(ctx, slugStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get publisher: %w", err)
	}
	if publisher == nil {
		return nil, entity.ErrPublisherNotFound
	}
	return u.toResponse(publisher), nil
}

func (u *publisherUsecase) Update(ctx context.Context, id uint, req UpdatePublisherInput) (*PublisherResult, error) {
	publisher, err := u.publisherRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get publisher: %w", err)
	}
	if publisher == nil {
		return nil, entity.ErrPublisherNotFound
	}

	publisherSlug := slug.Make(req.Name)
	exists, err := u.publisherRepo.ExistsBySlug(ctx, publisherSlug, id)
	if err != nil {
		return nil, fmt.Errorf("failed to check publisher slug: %w", err)
	}
	if exists {
		return nil, entity.ErrPublisherSlugConflict
	}

	publisher.Name = req.Name
	publisher.Slug = publisherSlug

	if err := u.publisherRepo.Update(ctx, publisher); err != nil {
		return nil, fmt.Errorf("failed to update publisher: %w", err)
	}

	if u.cacheSvc != nil {
		_ = u.cacheSvc.DeletePrefix(ctx, "publishers:")
	}

	return u.toResponse(publisher), nil
}

func (u *publisherUsecase) Delete(ctx context.Context, id uint) error {
	publisher, err := u.publisherRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get publisher: %w", err)
	}
	if publisher == nil {
		return entity.ErrPublisherNotFound
	}

	if err := u.publisherRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete publisher: %w", err)
	}

	if u.cacheSvc != nil {
		_ = u.cacheSvc.DeletePrefix(ctx, "publishers:")
	}

	return nil
}
