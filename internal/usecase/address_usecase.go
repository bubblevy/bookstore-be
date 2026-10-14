package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/logger"
)

type CreateAddressInput struct {
	UserID          uint
	ReceiverName    string
	Phone           string
	AddressLine     string
	ProvinceID      int
	ProvinceName    string
	CityID          int
	CityName        string
	SubdistrictID   *int
	SubdistrictName *string
	PostalCode      string
	IsPrimary       bool
}

type UpdateAddressInput struct {
	ID              uint
	UserID          uint
	ReceiverName    string
	Phone           string
	AddressLine     string
	ProvinceID      int
	ProvinceName    string
	CityID          int
	CityName        string
	SubdistrictID   *int
	SubdistrictName *string
	PostalCode      string
	IsPrimary       bool
}

type AddressListItem struct {
	ID           uint   `json:"id"`
	ReceiverName string `json:"receiver_name"`
	Phone        string `json:"phone"`
	AddressLine  string `json:"address_line"`
	CityName     string `json:"city_name"`
	ProvinceName string `json:"province_name"`
	PostalCode   string `json:"postal_code"`
	IsPrimary    bool   `json:"is_primary"`
}

type AddressResult struct {
	ID              uint       `json:"id"`
	ReceiverName    string     `json:"receiver_name"`
	Phone           string     `json:"phone"`
	AddressLine     string     `json:"address_line"`
	ProvinceID      int        `json:"province_id"`
	ProvinceName    string     `json:"province_name"`
	CityID          int        `json:"city_id"`
	CityName        string     `json:"city_name"`
	SubdistrictID   *int       `json:"subdistrict_id,omitempty"`
	SubdistrictName *string    `json:"subdistrict_name,omitempty"`
	PostalCode      string     `json:"postal_code"`
	IsPrimary       bool       `json:"is_primary"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}

type AddressUsecase interface {
	Create(ctx context.Context, i CreateAddressInput) (*AddressResult, error)
	GetByID(ctx context.Context, id uint, userID uint) (*AddressResult, error)
	GetAllByUser(ctx context.Context, userID uint) ([]AddressListItem, error)
	Update(ctx context.Context, i UpdateAddressInput) (*AddressResult, error)
	Delete(ctx context.Context, id uint, userID uint) error
	SetPrimary(ctx context.Context, id uint, userID uint) error
}

type addressUsecase struct {
	addressRepo repository.AddressRepository
	txManager   repository.TransactionManager
}

func NewAddressUsecase(addressRepo repository.AddressRepository, txManager repository.TransactionManager) AddressUsecase {
	return &addressUsecase{
		addressRepo: addressRepo,
		txManager:   txManager,
	}
}

func (u *addressUsecase) Create(ctx context.Context, i CreateAddressInput) (*AddressResult, error) {
	count, err := u.addressRepo.CountByUserID(ctx, i.UserID)
	if err != nil {
		logger.Error("create address failed - count error", "user_id", i.UserID, "error", err)
		return nil, entity.ErrFailedFetchAddress
	}

	if count >= 5 {
		logger.Warn("create address failed - limit reached", "user_id", i.UserID, "current_count", count)
		return nil, entity.ErrAddressLimitReached
	}

	isPrimary := i.IsPrimary
	if count == 0 {
		isPrimary = true
	}

	address := &entity.UserAddress{
		UserID:          i.UserID,
		ReceiverName:    i.ReceiverName,
		Phone:           i.Phone,
		AddressLine:     i.AddressLine,
		ProvinceID:      i.ProvinceID,
		ProvinceName:    i.ProvinceName,
		CityID:          i.CityID,
		CityName:        i.CityName,
		SubdistrictID:   i.SubdistrictID,
		SubdistrictName: i.SubdistrictName,
		PostalCode:      i.PostalCode,
		IsPrimary:       isPrimary,
	}

	err = u.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if isPrimary && count > 0 {
			if err := u.addressRepo.ResetPrimaryStatus(txCtx, i.UserID); err != nil {
				logger.Error("create address failed - reset primary status error", "user_id", i.UserID, "error", err)
				return err
			}
		}

		if err := u.addressRepo.Create(txCtx, address); err != nil {
			logger.Error("create address failed - insert error", "user_id", i.UserID, "error", err)
			return err
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create address: %w", err)
	}

	logger.Info("address created", "address_id", address.ID, "user_id", i.UserID)
	res := toAddressResult(address)
	return &res, nil
}

func (u *addressUsecase) GetByID(ctx context.Context, id uint, userID uint) (*AddressResult, error) {
	addr, err := u.addressRepo.FindByIDAndUserID(ctx, id, userID)
	if err != nil {
		logger.Error("get address failed - db error", "address_id", id, "user_id", userID, "error", err)
		return nil, entity.ErrFailedFetchAddress
	}
	if addr == nil {
		logger.Warn("get address failed - not found", "address_id", id, "user_id", userID)
		return nil, entity.ErrAddressNotFound
	}

	res := toAddressResult(addr)
	return &res, nil
}

func (u *addressUsecase) GetAllByUser(ctx context.Context, userID uint) ([]AddressListItem, error) {
	addresses, err := u.addressRepo.FindByUserID(ctx, userID)
	if err != nil {
		logger.Error("get all addresses failed - db error", "user_id", userID, "error", err)
		return nil, entity.ErrFailedFetchAddress
	}

	res := make([]AddressListItem, 0, len(addresses))
	for _, a := range addresses {
		res = append(res, AddressListItem{
			ID:           a.ID,
			ReceiverName: a.ReceiverName,
			Phone:        a.Phone,
			AddressLine:  a.AddressLine,
			CityName:     a.CityName,
			ProvinceName: a.ProvinceName,
			PostalCode:   a.PostalCode,
			IsPrimary:    a.IsPrimary,
		})
	}
	return res, nil
}

func (u *addressUsecase) Update(ctx context.Context, i UpdateAddressInput) (*AddressResult, error) {
	addr, err := u.addressRepo.FindByIDAndUserID(ctx, i.ID, i.UserID)
	if err != nil {
		logger.Error("update address failed - db error", "address_id", i.ID, "user_id", i.UserID, "error", err)
		return nil, entity.ErrFailedFetchAddress
	}
	if addr == nil {
		logger.Warn("update address failed - not found", "address_id", i.ID, "user_id", i.UserID)
		return nil, entity.ErrAddressNotFound
	}

	addr.ReceiverName = i.ReceiverName
	addr.Phone = i.Phone
	addr.AddressLine = i.AddressLine
	addr.ProvinceID = i.ProvinceID
	addr.ProvinceName = i.ProvinceName
	addr.CityID = i.CityID
	addr.CityName = i.CityName
	addr.SubdistrictID = i.SubdistrictID
	addr.SubdistrictName = i.SubdistrictName
	addr.PostalCode = i.PostalCode

	err = u.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if i.IsPrimary && !addr.IsPrimary {
			if err := u.addressRepo.ResetPrimaryStatus(txCtx, i.UserID); err != nil {
				logger.Error("update address failed - reset primary status error", "user_id", i.UserID, "error", err)
				return err
			}
			addr.IsPrimary = true
		}

		if err := u.addressRepo.Update(txCtx, addr); err != nil {
			logger.Error("update address failed - save error", "address_id", i.ID, "error", err)
			return err
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to update address: %w", err)
	}

	logger.Info("address updated", "address_id", i.ID, "user_id", i.UserID)
	res := toAddressResult(addr)
	return &res, nil
}

func (u *addressUsecase) Delete(ctx context.Context, id uint, userID uint) error {
	addr, err := u.addressRepo.FindByIDAndUserID(ctx, id, userID)
	if err != nil {
		logger.Error("delete address failed - db error", "address_id", id, "user_id", userID, "error", err)
		return entity.ErrFailedFetchAddress
	}
	if addr == nil {
		logger.Warn("delete address failed - not found", "address_id", id, "user_id", userID)
		return entity.ErrAddressNotFound
	}

	count, err := u.addressRepo.CountByUserID(ctx, userID)
	if err != nil {
		logger.Error("delete address failed - count error", "user_id", userID, "error", err)
		return entity.ErrFailedFetchAddress
	}

	if addr.IsPrimary && count > 1 {
		logger.Warn("delete address failed - cannot delete primary address", "address_id", id, "user_id", userID)
		return entity.ErrCannotDeletePrimaryAddress
	}

	if err := u.addressRepo.Delete(ctx, id, userID); err != nil {
		logger.Error("delete address failed - delete error", "address_id", id, "error", err)
		return entity.ErrFailedDeleteAddress
	}

	logger.Info("address deleted", "address_id", id, "user_id", userID)
	return nil
}

func (u *addressUsecase) SetPrimary(ctx context.Context, id uint, userID uint) error {
	addr, err := u.addressRepo.FindByIDAndUserID(ctx, id, userID)
	if err != nil {
		logger.Error("set primary address failed - db error", "address_id", id, "user_id", userID, "error", err)
		return entity.ErrFailedFetchAddress
	}
	if addr == nil {
		logger.Warn("set primary address failed - not found", "address_id", id, "user_id", userID)
		return entity.ErrAddressNotFound
	}

	if addr.IsPrimary {
		return nil
	}

	err = u.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := u.addressRepo.ResetPrimaryStatus(txCtx, userID); err != nil {
			logger.Error("set primary address failed - reset primary error", "user_id", userID, "error", err)
			return err
		}
		addr.IsPrimary = true
		if err := u.addressRepo.Update(txCtx, addr); err != nil {
			logger.Error("set primary address failed - save error", "address_id", id, "error", err)
			return err
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to set primary address: %w", err)
	}

	logger.Info("primary address updated", "address_id", id, "user_id", userID)
	return nil
}

func toAddressResult(a *entity.UserAddress) AddressResult {
	var deletedAt *time.Time
	if a.DeletedAt.Valid {
		deletedAt = &a.DeletedAt.Time
	}

	return AddressResult{
		ID:              a.ID,
		ReceiverName:    a.ReceiverName,
		Phone:           a.Phone,
		AddressLine:     a.AddressLine,
		ProvinceID:      a.ProvinceID,
		ProvinceName:    a.ProvinceName,
		CityID:          a.CityID,
		CityName:        a.CityName,
		SubdistrictID:   a.SubdistrictID,
		SubdistrictName: a.SubdistrictName,
		PostalCode:      a.PostalCode,
		IsPrimary:       a.IsPrimary,
		CreatedAt:       a.CreatedAt,
		UpdatedAt:       a.UpdatedAt,
		DeletedAt:       deletedAt,
	}
}
