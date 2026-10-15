package pgsql

import (
	"context"
	"errors"
	"strings"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"gorm.io/gorm"
)

type bookRepository struct {
	db *gorm.DB
}

func NewBookRepository(db *gorm.DB) repository.BookRepository {
	return &bookRepository{db: db}
}

func (r *bookRepository) Create(ctx context.Context, book *entity.Book) error {
	db := extractDB(ctx, r.db)
	return db.Create(book).Error
}

func (r *bookRepository) FindByID(ctx context.Context, id uint) (*entity.Book, error) {
	db := extractDB(ctx, r.db)
	var book entity.Book
	err := db.Preload("Category").
		Preload("Author").
		Preload("Publisher").
		First(&book, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &book, nil
}

func (r *bookRepository) FindBySlug(ctx context.Context, slug string) (*entity.Book, error) {
	db := extractDB(ctx, r.db)
	var book entity.Book
	err := db.Preload("Category").
		Preload("Author").
		Preload("Publisher").
		Where("slug = ?", slug).
		First(&book).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &book, nil
}

func (r *bookRepository) FindAll(ctx context.Context, filter repository.BookFilter) ([]entity.Book, int64, error) {
	db := extractDB(ctx, r.db)
	var total int64
	var books []entity.Book

	query := db.Model(&entity.Book{})

	// Search filter (Title or ISBN)
	if strings.TrimSpace(filter.Search) != "" {
		searchTerm := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where("LOWER(books.title) LIKE ? OR LOWER(books.isbn) LIKE ?", searchTerm, searchTerm)
	}

	// Filter relations via Slug (JOIN with unique partial index)
	if strings.TrimSpace(filter.CategorySlug) != "" {
		query = query.Joins("JOIN categories ON categories.id = books.category_id").
			Where("categories.slug = ? AND categories.deleted_at IS NULL", filter.CategorySlug)
	}
	if strings.TrimSpace(filter.AuthorSlug) != "" {
		query = query.Joins("JOIN authors ON authors.id = books.author_id").
			Where("authors.slug = ? AND authors.deleted_at IS NULL", filter.AuthorSlug)
	}
	if strings.TrimSpace(filter.PublisherSlug) != "" {
		query = query.Joins("JOIN publishers ON publishers.id = books.publisher_id").
			Where("publishers.slug = ? AND publishers.deleted_at IS NULL", filter.PublisherSlug)
	}

	// Price range filter
	if filter.MinPrice != nil {
		query = query.Where("books.price >= ?", *filter.MinPrice)
	}
	if filter.MaxPrice != nil {
		query = query.Where("books.price <= ?", *filter.MaxPrice)
	}

	// Active status filter
	if filter.IsActive != nil {
		query = query.Where("books.is_active = ?", *filter.IsActive)
	}

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []entity.Book{}, 0, nil
	}

	// Sort criteria
	switch filter.SortBy {
	case "price_asc":
		query = query.Order("books.price ASC")
	case "price_desc":
		query = query.Order("books.price DESC")
	case "title_asc":
		query = query.Order("books.title ASC")
	default:
		query = query.Order("books.created_at DESC")
	}

	err := query.Preload("Category").
		Preload("Author").
		Preload("Publisher").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&books).Error

	return books, total, err
}

func (r *bookRepository) Update(ctx context.Context, book *entity.Book) error {
	db := extractDB(ctx, r.db)
	return db.Save(book).Error
}

func (r *bookRepository) Delete(ctx context.Context, id uint) error {
	db := extractDB(ctx, r.db)
	result := db.Delete(&entity.Book{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *bookRepository) ExistsBySlug(ctx context.Context, slug string, excludeID ...uint) (bool, error) {
	db := extractDB(ctx, r.db)
	var count int64
	query := db.Model(&entity.Book{}).Where("slug = ?", slug)
	if len(excludeID) > 0 && excludeID[0] > 0 {
		query = query.Where("id != ?", excludeID[0])
	}
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *bookRepository) ExistsByISBN(ctx context.Context, isbn string, excludeID ...uint) (bool, error) {
	if strings.TrimSpace(isbn) == "" {
		return false, nil
	}
	db := extractDB(ctx, r.db)
	var count int64
	query := db.Model(&entity.Book{}).Where("isbn = ?", isbn)
	if len(excludeID) > 0 && excludeID[0] > 0 {
		query = query.Where("id != ?", excludeID[0])
	}
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *bookRepository) UpdateStock(ctx context.Context, id uint, quantity int) error {
	db := extractDB(ctx, r.db)
	query := db.Model(&entity.Book{}).Where("id = ?", id)
	if quantity < 0 {
		query = query.Where("stock >= ?", -quantity)
	}

	result := query.Update("stock", gorm.Expr("stock + ?", quantity))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return entity.ErrBookInsufficientStock
	}
	return nil
}

func (r *bookRepository) CountExistingIDs(ctx context.Context, ids []uint) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	db := extractDB(ctx, r.db)
	var count int64
	err := db.Model(&entity.Book{}).Where("id IN ?", ids).Count(&count).Error
	return count, err
}

func (r *bookRepository) FindByIDs(ctx context.Context, ids []uint) ([]entity.Book, error) {
	if len(ids) == 0 {
		return []entity.Book{}, nil
	}

	db := extractDB(ctx, r.db)
	var books []entity.Book

	err := db.Where("id IN ? AND is_active = ?", ids, true).Find(&books).Error
	if err != nil {
		return nil, err
	}

	return books, nil
}
