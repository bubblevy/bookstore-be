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
	return r.db.WithContext(ctx).Create(publisher).Error
}

func (r *publisherRepository) FindByID(ctx context.Context, id uint) (*entity.Publisher, error) {
	var publisher entity.Publisher
	err := r.db.WithContext(ctx).First(&publisher, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &publisher, nil
}

func (r *publisherRepository) FindBySlug(ctx context.Context, slug string) (*entity.Publisher, error) {
	var publisher entity.Publisher
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&publisher).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &publisher, nil
}

func (r *publisherRepository) FindAll(ctx context.Context) ([]entity.Publisher, error) {
	var publishers []entity.Publisher
	err := r.db.WithContext(ctx).Order("name ASC").Find(&publishers).Error
	return publishers, err
}

func (r *publisherRepository) Update(ctx context.Context, publisher *entity.Publisher) error {
	return r.db.WithContext(ctx).Save(publisher).Error
}

func (r *publisherRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&entity.Publisher{}, id).Error
}

func (r *publisherRepository) ExistsBySlug(ctx context.Context, slug string, excludeID ...uint) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&entity.Publisher{}).Where("slug = ?", slug)
	if len(excludeID) > 0 {
		query = query.Where("id != ?", excludeID[0])
	}
	err := query.Count(&count).Error
	return count > 0, err
}
