package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"github.com/gosimple/slug"
)

type CreateAuthorInput struct {
	Name string
	Bio  *string
}

type UpdateAuthorInput struct {
	Name string
	Bio  *string
}

type AuthorList struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type AuthorResult struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Bio       *string   `json:"bio,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AuthorBookItem struct {
	ID    uint    `json:"id"`
	Title string  `json:"title"`
	Slug  string  `json:"slug"`
	Price float64 `json:"price"`
	Cover *string `json:"cover_image"`
}

type AuthorDetailResult struct {
	ID        uint             `json:"id"`
	Name      string           `json:"name"`
	Slug      string           `json:"slug"`
	Bio       *string          `json:"bio,omitempty"`
	Books     []AuthorBookItem `json:"books"`
	BooksMeta pagination.Meta  `json:"books_meta"`
}

type AuthorUsecase interface {
	Create(ctx context.Context, req CreateAuthorInput) (*AuthorResult, error)
	FindAll(ctx context.Context, pq pagination.Query) ([]AuthorList, pagination.Meta, error)
	FindBySlug(ctx context.Context, slug string, bookPq pagination.Query) (*AuthorDetailResult, error)
	Update(ctx context.Context, id uint, req UpdateAuthorInput) (*AuthorResult, error)
	Delete(ctx context.Context, id uint) error
}

type authorUsecase struct {
	authorRepo repository.AuthorRepository
}

func NewAuthorUsecase(authorRepo repository.AuthorRepository) AuthorUsecase {
	return &authorUsecase{authorRepo: authorRepo}
}

func (u *authorUsecase) toResponse(author *entity.Author) *AuthorResult {
	return &AuthorResult{
		ID:        author.ID,
		Name:      author.Name,
		Slug:      author.Slug,
		Bio:       author.Bio,
		CreatedAt: author.CreatedAt,
		UpdatedAt: author.UpdatedAt,
	}
}

func (u *authorUsecase) Create(ctx context.Context, req CreateAuthorInput) (*AuthorResult, error) {
	authorSlug := slug.Make(req.Name)
	exists, err := u.authorRepo.ExistsBySlug(ctx, authorSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to check author slug: %w", err)
	}
	if exists {
		return nil, entity.ErrAuthorSlugConflict
	}

	author := &entity.Author{
		Name: req.Name,
		Slug: authorSlug,
		Bio:  req.Bio,
	}

	if err := u.authorRepo.Create(ctx, author); err != nil {
		return nil, fmt.Errorf("failed to create author: %w", err)
	}

	return u.toResponse(author), nil
}

func (u *authorUsecase) FindAll(ctx context.Context, pq pagination.Query) ([]AuthorList, pagination.Meta, error) {
	authors, total, err := u.authorRepo.FindAll(ctx, pq.PerPage, pq.Offset())
	if err != nil {
		return nil, pagination.Meta{}, fmt.Errorf("failed to get authors: %w", err)
	}

	responses := make([]AuthorList, 0, len(authors))
	for _, a := range authors {
		responses = append(responses, AuthorList{
			ID:   a.ID,
			Name: a.Name,
			Slug: a.Slug,
		})
	}

	meta := pagination.BuildMeta(total, pq.Page, pq.PerPage)
	return responses, meta, nil
}

func (u *authorUsecase) FindBySlug(ctx context.Context, slugStr string, bookPq pagination.Query) (*AuthorDetailResult, error) {
	author, err := u.authorRepo.FindBySlug(ctx, slugStr)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return nil, entity.ErrAuthorNotFound
	}

	books, totalBooks, err := u.authorRepo.FindBooksByAuthorID(ctx, author.ID, bookPq.PerPage, bookPq.Offset())
	if err != nil {
		return nil, err
	}

	bookItems := make([]AuthorBookItem, 0, len(books))
	for _, b := range books {
		bookItems = append(bookItems, AuthorBookItem{
			ID:    b.ID,
			Title: b.Title,
			Slug:  b.Slug,
			Price: b.Price,
			Cover: b.CoverImage,
		})
	}

	return &AuthorDetailResult{
		ID:        author.ID,
		Name:      author.Name,
		Slug:      author.Slug,
		Bio:       author.Bio,
		Books:     bookItems,
		BooksMeta: pagination.BuildMeta(totalBooks, bookPq.Page, bookPq.PerPage),
	}, nil
}

func (u *authorUsecase) Update(ctx context.Context, id uint, req UpdateAuthorInput) (*AuthorResult, error) {
	author, err := u.authorRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get author: %w", err)
	}
	if author == nil {
		return nil, entity.ErrAuthorNotFound
	}

	authorSlug := slug.Make(req.Name)
	exists, err := u.authorRepo.ExistsBySlug(ctx, authorSlug, id)
	if err != nil {
		return nil, fmt.Errorf("failed to check author slug: %w", err)
	}
	if exists {
		return nil, entity.ErrAuthorSlugConflict
	}

	author.Name = req.Name
	author.Slug = authorSlug
	author.Bio = req.Bio

	if err := u.authorRepo.Update(ctx, author); err != nil {
		return nil, fmt.Errorf("failed to update author: %w", err)
	}

	return u.toResponse(author), nil
}

func (u *authorUsecase) Delete(ctx context.Context, id uint) error {
	author, err := u.authorRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get author: %w", err)
	}
	if author == nil {
		return entity.ErrAuthorNotFound
	}

	if err := u.authorRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete author: %w", err)
	}
	return nil
}
