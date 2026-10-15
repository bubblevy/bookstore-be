package pgsql

import (
	"context"
	"errors"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"gorm.io/gorm"
)

type publisherRepository struct {
	db *gorm.DB
}

func NewPublisherRepository(db *gorm.DB) repository.PublisherRepository {
	return &publisherRepository{db: db}
}

func (r *publisherRepository) Create(ctx context.Context, publisher *entity.Publisher) error {
	db := extractDB(ctx, r.db)
	return db.Create(publisher).Error
}

func (r *publisherRepository) FindByID(ctx context.Context, id uint) (*entity.Publisher, error) {
	db := extractDB(ctx, r.db)
	var publisher entity.Publisher
	err := db.First(&publisher, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &publisher, nil
}

func (r *publisherRepository) FindBySlug(ctx context.Context, slug string) (*entity.Publisher, error) {
	db := extractDB(ctx, r.db)
	var publisher entity.Publisher
	err := db.Where("slug = ?", slug).First(&publisher).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &publisher, nil
}

func (r *publisherRepository) FindAll(ctx context.Context) ([]entity.Publisher, error) {
	db := extractDB(ctx, r.db)
	var publishers []entity.Publisher
	err := db.Order("name ASC").Find(&publishers).Error
	return publishers, err
}

func (r *publisherRepository) Update(ctx context.Context, publisher *entity.Publisher) error {
	db := extractDB(ctx, r.db)
	return db.Save(publisher).Error
}

func (r *publisherRepository) Delete(ctx context.Context, id uint) error {
	db := extractDB(ctx, r.db)
	result := db.Delete(&entity.Publisher{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *publisherRepository) ExistsBySlug(ctx context.Context, slug string, excludeID ...uint) (bool, error) {
	db := extractDB(ctx, r.db)
	var count int64

	query := db.Model(&entity.Publisher{}).Where("slug = ?", slug)
	if len(excludeID) > 0 && excludeID[0] > 0 {
		query = query.Where("id != ?", excludeID[0])
	}

	err := query.Count(&count).Error
	return count > 0, err
}
