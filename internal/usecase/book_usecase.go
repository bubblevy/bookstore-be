package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"github.com/gosimple/slug"
)

type CreateBookInput struct {
	CategoryID      uint
	AuthorID        uint
	PublisherID     uint
	Title           string
	ISBN            *string
	Synopsis        *string
	Price           float64
	WeightGrams     int
	Pages           *int
	PublicationYear *int
	Stock           int
	CoverImage      *string
	IsActive        *bool
}

type UpdateBookInput struct {
	CategoryID      uint
	AuthorID        uint
	PublisherID     uint
	Title           string
	ISBN            *string
	Synopsis        *string
	Price           float64
	WeightGrams     int
	Pages           *int
	PublicationYear *int
	Stock           int
	CoverImage      *string
	IsActive        *bool
}

type BookRelationItem struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type BookListItem struct {
	ID          uint             `json:"id"`
	Title       string           `json:"title"`
	Slug        string           `json:"slug"`
	Price       float64          `json:"price"`
	Stock       int              `json:"stock"`
	WeightGrams int              `json:"weight_grams"`
	CoverImage  *string          `json:"cover_image,omitempty"`
	IsActive    bool             `json:"is_active"`
	Category    BookRelationItem `json:"category"`
	Author      BookRelationItem `json:"author"`
	Publisher   BookRelationItem `json:"publisher"`
}

type BookDetailResult struct {
	ID              uint             `json:"id"`
	Title           string           `json:"title"`
	Slug            string           `json:"slug"`
	ISBN            *string          `json:"isbn,omitempty"`
	Synopsis        *string          `json:"synopsis,omitempty"`
	Price           float64          `json:"price"`
	WeightGrams     int              `json:"weight_grams"`
	Pages           *int             `json:"pages,omitempty"`
	PublicationYear *int             `json:"publication_year,omitempty"`
	Stock           int              `json:"stock"`
	CoverImage      *string          `json:"cover_image,omitempty"`
	IsActive        bool             `json:"is_active"`
	Category        BookRelationItem `json:"category"`
	Author          BookRelationItem `json:"author"`
	Publisher       BookRelationItem `json:"publisher"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

type BookUsecase interface {
	Create(ctx context.Context, input CreateBookInput) (*BookDetailResult, error)
	FindAll(ctx context.Context, filter repository.BookFilter, pq pagination.Query) ([]BookListItem, pagination.Meta, error)
	FindBySlug(ctx context.Context, slug string) (*BookDetailResult, error)
	Update(ctx context.Context, id uint, input UpdateBookInput) (*BookDetailResult, error)
	Delete(ctx context.Context, id uint) error
}

type bookUsecase struct {
	bookRepo      repository.BookRepository
	categoryRepo  repository.CategoryRepository
	authorRepo    repository.AuthorRepository
	publisherRepo repository.PublisherRepository
}

func NewBookUsecase(
	bookRepo repository.BookRepository,
	categoryRepo repository.CategoryRepository,
	authorRepo repository.AuthorRepository,
	publisherRepo repository.PublisherRepository,
) BookUsecase {
	return &bookUsecase{
		bookRepo:      bookRepo,
		categoryRepo:  categoryRepo,
		authorRepo:    authorRepo,
		publisherRepo: publisherRepo,
	}
}

func (u *bookUsecase) toDetailResult(b *entity.Book) *BookDetailResult {
	return &BookDetailResult{
		ID:              b.ID,
		Title:           b.Title,
		Slug:            b.Slug,
		ISBN:            b.ISBN,
		Synopsis:        b.Synopsis,
		Price:           b.Price,
		WeightGrams:     b.WeightGrams,
		Pages:           b.Pages,
		PublicationYear: b.PublicationYear,
		Stock:           b.Stock,
		CoverImage:      b.CoverImage,
		IsActive:        b.IsActive,
		Category: BookRelationItem{
			ID:   b.Category.ID,
			Name: b.Category.Name,
			Slug: b.Category.Slug,
		},
		Author: BookRelationItem{
			ID:   b.Author.ID,
			Name: b.Author.Name,
			Slug: b.Author.Slug,
		},
		Publisher: BookRelationItem{
			ID:   b.Publisher.ID,
			Name: b.Publisher.Name,
			Slug: b.Publisher.Slug,
		},
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
	}
}

func (u *bookUsecase) Create(ctx context.Context, input CreateBookInput) (*BookDetailResult, error) {
	category, err := u.categoryRepo.FindByID(ctx, input.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("failed checking category: %w", err)
	}
	if category == nil {
		return nil, entity.ErrBookCategoryNotFound
	}

	author, err := u.authorRepo.FindByID(ctx, input.AuthorID)
	if err != nil {
		return nil, fmt.Errorf("failed checking author: %w", err)
	}
	if author == nil {
		return nil, entity.ErrBookAuthorNotFound
	}

	publisher, err := u.publisherRepo.FindByID(ctx, input.PublisherID)
	if err != nil {
		return nil, fmt.Errorf("failed checking publisher: %w", err)
	}
	if publisher == nil {
		return nil, entity.ErrBookPublisherNotFound
	}

	bookSlug := slug.Make(input.Title)
	exists, err := u.bookRepo.ExistsBySlug(ctx, bookSlug)
	if err != nil {
		return nil, fmt.Errorf("failed checking slug: %w", err)
	}
	if exists {
		return nil, entity.ErrBookSlugConflict
	}

	if input.ISBN != nil && strings.TrimSpace(*input.ISBN) != "" {
		exists, err := u.bookRepo.ExistsByISBN(ctx, *input.ISBN)
		if err != nil {
			return nil, fmt.Errorf("failed checking isbn: %w", err)
		}
		if exists {
			return nil, entity.ErrBookISBNConflict
		}
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	book := &entity.Book{
		CategoryID:      input.CategoryID,
		AuthorID:        input.AuthorID,
		PublisherID:     input.PublisherID,
		Title:           input.Title,
		Slug:            bookSlug,
		ISBN:            input.ISBN,
		Synopsis:        input.Synopsis,
		Price:           input.Price,
		WeightGrams:     input.WeightGrams,
		Pages:           input.Pages,
		PublicationYear: input.PublicationYear,
		Stock:           input.Stock,
		CoverImage:      input.CoverImage,
		IsActive:        isActive,
	}

	if err := u.bookRepo.Create(ctx, book); err != nil {
		return nil, fmt.Errorf("failed to create book: %w", err)
	}

	book.Category = *category
	book.Author = *author
	book.Publisher = *publisher

	return u.toDetailResult(book), nil
}

func (u *bookUsecase) FindAll(ctx context.Context, filter repository.BookFilter, pq pagination.Query) ([]BookListItem, pagination.Meta, error) {
	filter.Limit = pq.PerPage
	filter.Offset = pq.Offset()

	books, total, err := u.bookRepo.FindAll(ctx, filter)
	if err != nil {
		return nil, pagination.Meta{}, fmt.Errorf("failed to fetch books: %w", err)
	}

	items := make([]BookListItem, 0, len(books))
	for _, b := range books {
		items = append(items, BookListItem{
			ID:          b.ID,
			Title:       b.Title,
			Slug:        b.Slug,
			Price:       b.Price,
			Stock:       b.Stock,
			WeightGrams: b.WeightGrams,
			CoverImage:  b.CoverImage,
			IsActive:    b.IsActive,
			Category: BookRelationItem{
				ID:   b.Category.ID,
				Name: b.Category.Name,
				Slug: b.Category.Slug,
			},
			Author: BookRelationItem{
				ID:   b.Author.ID,
				Name: b.Author.Name,
				Slug: b.Author.Slug,
			},
			Publisher: BookRelationItem{
				ID:   b.Publisher.ID,
				Name: b.Publisher.Name,
				Slug: b.Publisher.Slug,
			},
		})
	}

	meta := pagination.BuildMeta(total, pq.Page, pq.PerPage)
	return items, meta, nil
}

func (u *bookUsecase) FindBySlug(ctx context.Context, slugStr string) (*BookDetailResult, error) {
	book, err := u.bookRepo.FindBySlug(ctx, slugStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get book: %w", err)
	}
	if book == nil {
		return nil, entity.ErrBookNotFound
	}
	return u.toDetailResult(book), nil
}

func (u *bookUsecase) Update(ctx context.Context, id uint, input UpdateBookInput) (*BookDetailResult, error) {
	book, err := u.bookRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed getting book: %w", err)
	}
	if book == nil {
		return nil, entity.ErrBookNotFound
	}

	category, err := u.categoryRepo.FindByID(ctx, input.CategoryID)
	if err != nil || category == nil {
		return nil, entity.ErrBookCategoryNotFound
	}

	author, err := u.authorRepo.FindByID(ctx, input.AuthorID)
	if err != nil || author == nil {
		return nil, entity.ErrBookAuthorNotFound
	}

	publisher, err := u.publisherRepo.FindByID(ctx, input.PublisherID)
	if err != nil || publisher == nil {
		return nil, entity.ErrBookPublisherNotFound
	}

	bookSlug := slug.Make(input.Title)
	exists, err := u.bookRepo.ExistsBySlug(ctx, bookSlug, id)
	if err != nil {
		return nil, fmt.Errorf("failed checking slug: %w", err)
	}
	if exists {
		return nil, entity.ErrBookSlugConflict
	}

	if input.ISBN != nil && strings.TrimSpace(*input.ISBN) != "" {
		exists, err := u.bookRepo.ExistsByISBN(ctx, *input.ISBN, id)
		if err != nil {
			return nil, fmt.Errorf("failed checking isbn: %w", err)
		}
		if exists {
			return nil, entity.ErrBookISBNConflict
		}
	}

	book.CategoryID = input.CategoryID
	book.AuthorID = input.AuthorID
	book.PublisherID = input.PublisherID
	book.Title = input.Title
	book.Slug = bookSlug
	book.ISBN = input.ISBN
	book.Synopsis = input.Synopsis
	book.Price = input.Price
	book.WeightGrams = input.WeightGrams
	book.Pages = input.Pages
	book.PublicationYear = input.PublicationYear
	book.Stock = input.Stock
	book.CoverImage = input.CoverImage
	if input.IsActive != nil {
		book.IsActive = *input.IsActive
	}

	if err := u.bookRepo.Update(ctx, book); err != nil {
		return nil, fmt.Errorf("failed to update book: %w", err)
	}

	book.Category = *category
	book.Author = *author
	book.Publisher = *publisher

	return u.toDetailResult(book), nil
}

func (u *bookUsecase) Delete(ctx context.Context, id uint) error {
	book, err := u.bookRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get book: %w", err)
	}
	if book == nil {
		return entity.ErrBookNotFound
	}

	if err := u.bookRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete book: %w", err)
	}
	return nil
}
