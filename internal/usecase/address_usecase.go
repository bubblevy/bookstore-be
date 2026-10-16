package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"gorm.io/gorm"
)

type CreateAddressInput struct {
	ReceiverName    string
	Phone           string
	AddressLine     string
	BiteshipAreaID  string
	ProvinceName    string
	CityName        string
	DistrictName    string
	SubdistrictName string
	PostalCode      string
	IsPrimary       bool
}

type UpdateAddressInput struct {
	ReceiverName    string
	Phone           string
	AddressLine     string
	BiteshipAreaID  string
	ProvinceName    string
	CityName        string
	DistrictName    string
	SubdistrictName string
	PostalCode      string
	IsPrimary       bool
}

type AddressListItemResult struct {
	ID           uint   `json:"id"`
	ReceiverName string `json:"receiver_name"`
	Phone        string `json:"phone"`
	CityName     string `json:"city_name"`
	DistrictName string `json:"district_name"`
	IsPrimary    bool   `json:"is_primary"`
}

type AddressResult struct {
	ID              uint      `json:"id"`
	ReceiverName    string    `json:"receiver_name"`
	Phone           string    `json:"phone"`
	AddressLine     string    `json:"address_line"`
	BiteshipAreaID  string    `json:"biteship_area_id"`
	ProvinceName    string    `json:"province_name"`
	CityName        string    `json:"city_name"`
	DistrictName    string    `json:"district_name"`
	SubdistrictName string    `json:"subdistrict_name"`
	PostalCode      string    `json:"postal_code"`
	IsPrimary       bool      `json:"is_primary"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type AddressUsecase interface {
	CreateAddress(ctx context.Context, userID uint, input CreateAddressInput) (*AddressResult, error)
	GetAddressByID(ctx context.Context, userID, addressID uint) (*AddressResult, error)
	GetUserAddresses(ctx context.Context, userID uint) ([]AddressListItemResult, error)
	GetPrimaryAddress(ctx context.Context, userID uint) (*AddressResult, error)
	UpdateAddress(ctx context.Context, userID, addressID uint, input UpdateAddressInput) (*AddressResult, error)
	DeleteAddress(ctx context.Context, userID, addressID uint) error
	SetPrimaryAddress(ctx context.Context, userID, addressID uint) error
}

type addressUsecase struct {
	db          *gorm.DB
	addressRepo repository.AddressRepository
}

func NewAddressUsecase(db *gorm.DB, addressRepo repository.AddressRepository) AddressUsecase {
	return &addressUsecase{
		db:          db,
		addressRepo: addressRepo,
	}
}

func (u *addressUsecase) CreateAddress(ctx context.Context, userID uint, input CreateAddressInput) (*AddressResult, error) {
	count, err := u.addressRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed counting user addresses: %w", err)
	}

	const maxAddresses = 5
	if count >= maxAddresses {
		return nil, entity.ErrAddressLimitReached
	}

	if count == 0 {
		input.IsPrimary = true
	}

	addr := &entity.Address{
		UserID:          userID,
		ReceiverName:    input.ReceiverName,
		Phone:           input.Phone,
		AddressLine:     input.AddressLine,
		BiteshipAreaID:  input.BiteshipAreaID,
		ProvinceName:    input.ProvinceName,
		CityName:        input.CityName,
		DistrictName:    input.DistrictName,
		SubdistrictName: input.SubdistrictName,
		PostalCode:      input.PostalCode,
		IsPrimary:       input.IsPrimary,
	}

	err = u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repoTx := u.addressRepo.WithTx(tx)
		if input.IsPrimary && count > 0 {
			if err := repoTx.ResetPrimaryStatus(ctx, userID); err != nil {
				return err
			}
		}
		return repoTx.Create(ctx, addr)
	})
	if err != nil {
		return nil, err
	}

	return toAddressResult(addr), nil
}

func (u *addressUsecase) GetAddressByID(ctx context.Context, userID, addressID uint) (*AddressResult, error) {
	addr, err := u.addressRepo.FindByID(ctx, addressID)
	if err != nil {
		return nil, err
	}
	if addr == nil {
		return nil, entity.ErrAddressNotFound
	}
	if addr.UserID != userID {
		return nil, entity.ErrUnauthorizedAddress
	}

	return toAddressResult(addr), nil
}

func (u *addressUsecase) GetUserAddresses(ctx context.Context, userID uint) ([]AddressListItemResult, error) {
	addresses, err := u.addressRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	results := make([]AddressListItemResult, len(addresses))
	for i := range addresses {
		results[i] = AddressListItemResult{
			ID:           addresses[i].ID,
			ReceiverName: addresses[i].ReceiverName,
			Phone:        addresses[i].Phone,
			CityName:     addresses[i].CityName,
			DistrictName: addresses[i].DistrictName,
			IsPrimary:    addresses[i].IsPrimary,
		}
	}

	return results, nil
}

func (u *addressUsecase) GetPrimaryAddress(ctx context.Context, userID uint) (*AddressResult, error) {
	addr, err := u.addressRepo.FindPrimaryByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if addr == nil {
		return nil, entity.ErrAddressNotFound
	}

	return toAddressResult(addr), nil
}

func (u *addressUsecase) UpdateAddress(ctx context.Context, userID, addressID uint, input UpdateAddressInput) (*AddressResult, error) {
	addr, err := u.addressRepo.FindByID(ctx, addressID)
	if err != nil {
		return nil, err
	}
	if addr == nil {
		return nil, entity.ErrAddressNotFound
	}
	if addr.UserID != userID {
		return nil, entity.ErrUnauthorizedAddress
	}

	addr.ReceiverName = input.ReceiverName
	addr.Phone = input.Phone
	addr.AddressLine = input.AddressLine
	addr.BiteshipAreaID = input.BiteshipAreaID
	addr.ProvinceName = input.ProvinceName
	addr.CityName = input.CityName
	addr.DistrictName = input.DistrictName
	addr.SubdistrictName = input.SubdistrictName
	addr.PostalCode = input.PostalCode

	err = u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repoTx := u.addressRepo.WithTx(tx)
		if input.IsPrimary && !addr.IsPrimary {
			if err := repoTx.ResetPrimaryStatus(ctx, userID); err != nil {
				return err
			}
			addr.IsPrimary = true
		}
		return repoTx.Update(ctx, addr)
	})
	if err != nil {
		return nil, err
	}

	return toAddressResult(addr), nil
}

func (u *addressUsecase) DeleteAddress(ctx context.Context, userID, addressID uint) error {
	addr, err := u.addressRepo.FindByID(ctx, addressID)
	if err != nil {
		return err
	}
	if addr == nil {
		return entity.ErrAddressNotFound
	}
	if addr.UserID != userID {
		return entity.ErrUnauthorizedAddress
	}
	if addr.IsPrimary {
		return entity.ErrPrimaryAddressDelete
	}

	return u.addressRepo.Delete(ctx, addressID)
}

func (u *addressUsecase) SetPrimaryAddress(ctx context.Context, userID, addressID uint) error {
	return u.addressRepo.SetPrimary(ctx, userID, addressID)
}

func toAddressResult(a *entity.Address) *AddressResult {
	return &AddressResult{
		ID:              a.ID,
		ReceiverName:    a.ReceiverName,
		Phone:           a.Phone,
		AddressLine:     a.AddressLine,
		BiteshipAreaID:  a.BiteshipAreaID,
		ProvinceName:    a.ProvinceName,
		CityName:        a.CityName,
		DistrictName:    a.DistrictName,
		SubdistrictName: a.SubdistrictName,
		PostalCode:      a.PostalCode,
		IsPrimary:       a.IsPrimary,
		CreatedAt:       a.CreatedAt,
		UpdatedAt:       a.UpdatedAt,
	}
}
