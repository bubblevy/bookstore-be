package pgsql

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"gorm.io/gorm"
)

type discountRepository struct {
	db *gorm.DB
}

func NewDiscountRepository(db *gorm.DB) repository.DiscountRepository {
	return &discountRepository{db: db}
}

func (r *discountRepository) Create(ctx context.Context, discount *entity.Discount, bookIDs []uint, tiers []entity.DiscountTier) error {
	db := extractDB(ctx, r.db)

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(discount).Error; err != nil {
			return err
		}

		// synchronize associated books
		if !discount.GetIsAllBooks() && len(bookIDs) > 0 {
			type DiscountBookPivot struct {
				DiscountID uint `gorm:"column:discount_id"`
				BookID     uint `gorm:"column:book_id"`
			}

			pivots := make([]DiscountBookPivot, 0, len(bookIDs))
			for _, bookID := range bookIDs {
				pivots = append(pivots, DiscountBookPivot{
					DiscountID: discount.ID,
					BookID:     bookID,
				})
			}

			if err := tx.Table("discount_books").Create(&pivots).Error; err != nil {
				return err
			}
		}

		// Save tiers if type is TIERED
		if discount.DiscountType == entity.DiscountTypeTiered && len(tiers) > 0 {
			for i := range tiers {
				tiers[i].DiscountID = discount.ID
			}
			if err := tx.Create(&tiers).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *discountRepository) FindByID(ctx context.Context, id uint) (*entity.Discount, error) {
	db := extractDB(ctx, r.db)
	var discount entity.Discount
	err := db.Preload("Books").
		Preload("Tiers", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("discount_tiers.min_spend ASC")
		}).
		First(&discount, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &discount, nil
}

func (r *discountRepository) FindByCode(ctx context.Context, code string) (*entity.Discount, error) {
	db := extractDB(ctx, r.db)
	var discount entity.Discount
	err := db.Preload("Books").
		Preload("Tiers", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("discount_tiers.min_spend ASC")
		}).
		Where("UPPER(code) = UPPER(?)", strings.TrimSpace(code)).
		First(&discount).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &discount, nil
}

func (r *discountRepository) FindAll(ctx context.Context, filter repository.DiscountFilter) ([]entity.Discount, int64, error) {
	db := extractDB(ctx, r.db)
	var total int64
	var discounts []entity.Discount

	query := db.Model(&entity.Discount{})

	if strings.TrimSpace(filter.Search) != "" {
		searchTerm := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(code) LIKE ?", searchTerm, searchTerm)
	}

	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	if filter.OnlyValid {
		now := time.Now()
		query = query.Where("is_active = true AND start_date <= ? AND end_date >= ? AND (quota = 0 OR used_count < quota)", now, now)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []entity.Discount{}, 0, nil
	}

	err := query.Order("created_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&discounts).Error

	return discounts, total, err
}

func (r *discountRepository) Update(ctx context.Context, discount *entity.Discount, bookIDs []uint, tiers []entity.DiscountTier) error {
	db := extractDB(ctx, r.db)

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(discount).Error; err != nil {
			return err
		}

		// synchronize associated books
		if err := tx.Exec("DELETE FROM discount_books WHERE discount_id = ?", discount.ID).Error; err != nil {
			return err
		}

		if !discount.GetIsAllBooks() && len(bookIDs) > 0 {
			type DiscountBookPivot struct {
				DiscountID uint `gorm:"column:discount_id"`
				BookID     uint `gorm:"column:book_id"`
			}

			pivots := make([]DiscountBookPivot, 0, len(bookIDs))
			for _, bookID := range bookIDs {
				pivots = append(pivots, DiscountBookPivot{
					DiscountID: discount.ID,
					BookID:     bookID,
				})
			}

			if err := tx.Table("discount_books").Create(&pivots).Error; err != nil {
				return err
			}
		}

		// synchronize associated tiers
		if err := tx.Where("discount_id = ?", discount.ID).Delete(&entity.DiscountTier{}).Error; err != nil {
			return err
		}

		if discount.DiscountType == entity.DiscountTypeTiered && len(tiers) > 0 {
			for i := range tiers {
				tiers[i].DiscountID = discount.ID
			}
			if err := tx.Create(&tiers).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *discountRepository) Delete(ctx context.Context, id uint) error {
	db := extractDB(ctx, r.db)
	result := db.Delete(&entity.Discount{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *discountRepository) ExistsByCode(ctx context.Context, code string, excludeID ...uint) (bool, error) {
	db := extractDB(ctx, r.db)
	var count int64
	query := db.Model(&entity.Discount{}).Where("UPPER(code) = UPPER(?)", strings.TrimSpace(code))
	if len(excludeID) > 0 && excludeID[0] > 0 {
		query = query.Where("id != ?", excludeID[0])
	}
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *discountRepository) IncrementUsedCount(ctx context.Context, id uint) error {
	db := extractDB(ctx, r.db)
	query := db.Model(&entity.Discount{}).Where("id = ?", id)
	query = query.Where("quota = 0 OR used_count < quota")

	result := query.Update("used_count", gorm.Expr("used_count + 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return entity.ErrDiscountQuotaExhaust
	}
	return nil
}
