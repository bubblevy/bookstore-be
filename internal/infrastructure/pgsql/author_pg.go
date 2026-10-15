package pgsql

import (
	"context"
	"errors"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"gorm.io/gorm"
)

type authorRepository struct {
	db *gorm.DB
}

func NewAuthorRepository(db *gorm.DB) repository.AuthorRepository {
	return &authorRepository{db: db}
}

func (r *authorRepository) Create(ctx context.Context, author *entity.Author) error {
	db := extractDB(ctx, r.db)
	return db.Create(author).Error
}

func (r *authorRepository) FindByID(ctx context.Context, id uint) (*entity.Author, error) {
	db := extractDB(ctx, r.db)
	var author entity.Author
	err := db.First(&author, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &author, nil
}

func (r *authorRepository) FindBySlug(ctx context.Context, slug string) (*entity.Author, error) {
	db := extractDB(ctx, r.db)
	var author entity.Author
	err := db.Where("slug = ?", slug).First(&author).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &author, nil
}

func (r *authorRepository) FindAll(ctx context.Context, limit, offset int) ([]entity.Author, int64, error) {
	db := extractDB(ctx, r.db)
	var total int64

	if err := db.Model(&entity.Author{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []entity.Author{}, 0, nil
	}

	var authors []entity.Author
	err := db.Order("name ASC").
		Limit(limit).
		Offset(offset).
		Find(&authors).Error

	return authors, total, err
}

func (r *authorRepository) Update(ctx context.Context, author *entity.Author) error {
	db := extractDB(ctx, r.db)
	return db.Save(author).Error
}

func (r *authorRepository) Delete(ctx context.Context, id uint) error {
	db := extractDB(ctx, r.db)
	result := db.Delete(&entity.Author{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *authorRepository) ExistsBySlug(ctx context.Context, slug string, excludeID ...uint) (bool, error) {
	db := extractDB(ctx, r.db)
	var count int64

	query := db.Model(&entity.Author{}).Where("slug = ?", slug)
	if len(excludeID) > 0 {
		query = query.Where("id != ?", excludeID[0])
	}

	err := query.Count(&count).Error
	return count > 0, err
}

func (r *authorRepository) FindBooksByAuthorID(ctx context.Context, authorID uint, limit, offset int) ([]entity.Book, int64, error) {
	db := extractDB(ctx, r.db)
	var total int64

	baseQuery := db.Model(&entity.Book{}).
		Where("author_id = ? AND is_active = ?", authorID, true)

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []entity.Book{}, 0, nil
	}

	var books []entity.Book
	err := baseQuery.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&books).Error

	return books, total, err
}
