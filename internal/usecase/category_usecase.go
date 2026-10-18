package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/gosimple/slug"
)

type CreateCategoryInput struct {
	Name     string
	ImageURL *string
}

type UpdateCategoryInput struct {
	ID       uint
	Name     string
	ImageURL *string
}

type CategoryList struct {
	ID         uint    `json:"id"`
	Name       string  `json:"name"`
	Slug       string  `json:"slug"`
	ImageURL   *string `json:"image_url,omitempty"`
	TotalBooks int64   `json:"total_books"`
}

type CategoryResult struct {
	ID         uint      `json:"id"`
	Name       string    `json:"name"`
	Slug       string    `json:"slug"`
	ImageURL   *string   `json:"image_url,omitempty"`
	TotalBooks int64     `json:"total_books"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CategoryUsecase interface {
	Create(ctx context.Context, req CreateCategoryInput) (*CategoryResult, error)
	FindAll(ctx context.Context) ([]CategoryList, error)
	FindBySlug(ctx context.Context, slug string) (*CategoryResult, error)
	Update(ctx context.Context, id uint, req UpdateCategoryInput) (*CategoryResult, error)
	Delete(ctx context.Context, id uint) error
}

type categoryUsecase struct {
	categoryRepo repository.CategoryRepository
}

func NewCategoryUsecase(categoryRepo repository.CategoryRepository) CategoryUsecase {
	return &categoryUsecase{categoryRepo: categoryRepo}
}

func (u *categoryUsecase) toResponse(cat *entity.Category) *CategoryResult {
	return &CategoryResult{
		ID:         cat.ID,
		Name:       cat.Name,
		Slug:       cat.Slug,
		ImageURL:   cat.ImageURL,
		TotalBooks: cat.TotalBooks,
		CreatedAt:  cat.CreatedAt,
		UpdatedAt:  cat.UpdatedAt,
	}
}

func (u *categoryUsecase) Create(ctx context.Context, req CreateCategoryInput) (*CategoryResult, error) {
	categorySlug := slug.Make(req.Name)

	exists, err := u.categoryRepo.ExistsBySlug(ctx, categorySlug)
	if err != nil {
		return nil, fmt.Errorf("failed to check category slug: %w", err)
	}
	if exists {
		return nil, entity.ErrCategorySlugConflict
	}

	category := &entity.Category{
		Name:     req.Name,
		Slug:     categorySlug,
		ImageURL: req.ImageURL,
	}

	if err := u.categoryRepo.Create(ctx, category); err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	return u.toResponse(category), nil
}

func (u *categoryUsecase) FindAll(ctx context.Context) ([]CategoryList, error) {
	categories, err := u.categoryRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	responses := make([]CategoryList, 0, len(categories))
	for _, cat := range categories {
		responses = append(responses, CategoryList{
			ID:         cat.ID,
			Name:       cat.Name,
			Slug:       cat.Slug,
			ImageURL:   cat.ImageURL,
			TotalBooks: cat.TotalBooks,
		})
	}
	return responses, nil
}

func (u *categoryUsecase) FindBySlug(ctx context.Context, slugStr string) (*CategoryResult, error) {
	category, err := u.categoryRepo.FindBySlug(ctx, slugStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get category: %w", err)
	}
	if category == nil {
		return nil, entity.ErrCategoryNotFound
	}
	return u.toResponse(category), nil
}

func (u *categoryUsecase) Update(ctx context.Context, id uint, req UpdateCategoryInput) (*CategoryResult, error) {
	category, err := u.categoryRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get category: %w", err)
	}
	if category == nil {
		return nil, entity.ErrCategoryNotFound
	}

	categorySlug := slug.Make(req.Name)
	exists, err := u.categoryRepo.ExistsBySlug(ctx, categorySlug, id)
	if err != nil {
		return nil, fmt.Errorf("failed to check category slug: %w", err)
	}
	if exists {
		return nil, entity.ErrCategorySlugConflict
	}

	category.Name = req.Name
	category.Slug = categorySlug
	category.ImageURL = req.ImageURL

	if err := u.categoryRepo.Update(ctx, category); err != nil {
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	return u.toResponse(category), nil
}

func (u *categoryUsecase) Delete(ctx context.Context, id uint) error {
	category, err := u.categoryRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get category: %w", err)
	}
	if category == nil {
		return entity.ErrCategoryNotFound
	}

	if err := u.categoryRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}
	return nil
}
