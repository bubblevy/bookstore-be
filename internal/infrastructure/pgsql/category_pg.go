package pgsql

import (
	"context"
	"errors"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"gorm.io/gorm"
)

type categoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) repository.CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) Create(ctx context.Context, category *entity.Category) error {
	db := extractDB(ctx, r.db)
	return db.Create(category).Error
}

func (r *categoryRepository) FindByID(ctx context.Context, id uint) (*entity.Category, error) {
	db := extractDB(ctx, r.db)
	var category entity.Category
	err := db.First(&category, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &category, nil
}

func (r *categoryRepository) FindBySlug(ctx context.Context, slug string) (*entity.Category, error) {
	db := extractDB(ctx, r.db)
	var category entity.Category
	err := db.Where("slug = ?", slug).First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &category, nil
}

func (r *categoryRepository) FindAll(ctx context.Context) ([]entity.Category, error) {
	db := extractDB(ctx, r.db)
	var categories []entity.Category
	err := db.Order("name ASC").Find(&categories).Error
	return categories, err
}

func (r *categoryRepository) Update(ctx context.Context, category *entity.Category) error {
	db := extractDB(ctx, r.db)
	return db.Save(category).Error
}

func (r *categoryRepository) Delete(ctx context.Context, id uint) error {
	db := extractDB(ctx, r.db)
	result := db.Delete(&entity.Category{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *categoryRepository) ExistsBySlug(ctx context.Context, slug string, excludeID ...uint) (bool, error) {
	db := extractDB(ctx, r.db)
	var count int64

	query := db.Model(&entity.Category{}).Where("slug = ?", slug)
	if len(excludeID) > 0 && excludeID[0] > 0 {
		query = query.Where("id != ?", excludeID[0])
	}

	err := query.Count(&count).Error
	return count > 0, err
}
