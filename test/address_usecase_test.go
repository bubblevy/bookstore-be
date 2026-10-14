package test

import (
	"context"
	"errors"
	"testing"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ============================================================================
// MOCKS
// ============================================================================

type mockAddressRepo struct {
	addresses     map[uint]*entity.UserAddress
	nextAddressID uint
	shouldFail    bool
}

func newMockAddressRepo() *mockAddressRepo {
	return &mockAddressRepo{
		addresses:     make(map[uint]*entity.UserAddress),
		nextAddressID: 1,
	}
}

func (m *mockAddressRepo) Create(ctx context.Context, address *entity.UserAddress) error {
	if m.shouldFail {
		return errors.New("db create error")
	}
	address.ID = m.nextAddressID
	m.nextAddressID++
	m.addresses[address.ID] = address
	return nil
}

func (m *mockAddressRepo) FindByID(ctx context.Context, id uint) (*entity.UserAddress, error) {
	if m.shouldFail {
		return nil, errors.New("db error")
	}
	if addr, ok := m.addresses[id]; ok {
		return addr, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockAddressRepo) FindByIDAndUserID(ctx context.Context, id uint, userID uint) (*entity.UserAddress, error) {
	if m.shouldFail {
		return nil, errors.New("db error")
	}
	if addr, ok := m.addresses[id]; ok && addr.UserID == userID {
		return addr, nil
	}
	return nil, nil
}

func (m *mockAddressRepo) FindByUserID(ctx context.Context, userID uint) ([]entity.UserAddress, error) {
	if m.shouldFail {
		return nil, errors.New("db error")
	}
	var res []entity.UserAddress
	for _, addr := range m.addresses {
		if addr.UserID == userID {
			res = append(res, *addr)
		}
	}
	return res, nil
}

func (m *mockAddressRepo) Update(ctx context.Context, address *entity.UserAddress) error {
	if m.shouldFail {
		return errors.New("db update error")
	}
	m.addresses[address.ID] = address
	return nil
}

func (m *mockAddressRepo) Delete(ctx context.Context, id uint, userID uint) error {
	if m.shouldFail {
		return errors.New("db delete error")
	}
	if addr, ok := m.addresses[id]; ok && addr.UserID == userID {
		delete(m.addresses, id)
		return nil
	}
	return gorm.ErrRecordNotFound
}

func (m *mockAddressRepo) ResetPrimaryStatus(ctx context.Context, userID uint) error {
	if m.shouldFail {
		return errors.New("db reset primary error")
	}
	for _, addr := range m.addresses {
		if addr.UserID == userID {
			addr.IsPrimary = false
		}
	}
	return nil
}

func (m *mockAddressRepo) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	if m.shouldFail {
		return 0, errors.New("db count error")
	}
	var count int64
	for _, addr := range m.addresses {
		if addr.UserID == userID {
			count++
		}
	}
	return count, nil
}

type mockAddressTxManager struct {
	shouldFail bool
}

func newMockAddressTxManager() *mockAddressTxManager {
	return &mockAddressTxManager{}
}

func (m *mockAddressTxManager) WithTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	if m.shouldFail {
		return errors.New("transaction failed")
	}
	return fn(ctx)
}

func setupAddressUsecase() (usecase.AddressUsecase, *mockAddressRepo, *mockAddressTxManager) {
	addressRepo := newMockAddressRepo()
	txManager := newMockAddressTxManager()
	uc := usecase.NewAddressUsecase(addressRepo, txManager)
	return uc, addressRepo, txManager
}

// ============================================================================
// CREATE ADDRESS TESTS
// ============================================================================

func TestCreateAddress_FirstAddress_AutoPrimary(t *testing.T) {
	uc, _, _ := setupAddressUsecase()

	input := usecase.CreateAddressInput{
		UserID:       1,
		ReceiverName: "Budi Santoso",
		Phone:        "081234567890",
		AddressLine:  "Jl. Merbabu No. 12",
		ProvinceID:   10,
		ProvinceName: "Jawa Tengah",
		CityID:       196,
		CityName:     "Klaten",
		PostalCode:   "57411",
		IsPrimary:    false,
	}

	result, err := uc.Create(context.Background(), input)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint(1), result.ID)
	assert.True(t, result.IsPrimary)
}

func TestCreateAddress_SecondAddress_SetAsNewPrimary(t *testing.T) {
	uc, repo, _ := setupAddressUsecase()

	addr1 := &entity.UserAddress{
		UserID:       1,
		ReceiverName: "Alamat Pertama",
		IsPrimary:    true,
	}
	_ = repo.Create(context.Background(), addr1)

	input := usecase.CreateAddressInput{
		UserID:       1,
		ReceiverName: "Alamat Kedua",
		Phone:        "081234567891",
		AddressLine:  "Jl. Pemuda No. 45",
		ProvinceID:   10,
		ProvinceName: "Jawa Tengah",
		CityID:       196,
		CityName:     "Klaten",
		PostalCode:   "57411",
		IsPrimary:    true,
	}

	result, err := uc.Create(context.Background(), input)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsPrimary)

	assert.False(t, repo.addresses[addr1.ID].IsPrimary)
}

func TestCreateAddress_LimitReached_Fail(t *testing.T) {
	uc, repo, _ := setupAddressUsecase()

	for i := 1; i <= 5; i++ {
		_ = repo.Create(context.Background(), &entity.UserAddress{
			UserID:       1,
			ReceiverName: "Alamat Test",
		})
	}

	input := usecase.CreateAddressInput{
		UserID:       1,
		ReceiverName: "Alamat Ke-6",
		Phone:        "081234567890",
		AddressLine:  "Jl. Merbabu No. 12",
		ProvinceID:   10,
		ProvinceName: "Jawa Tengah",
		CityID:       196,
		CityName:     "Klaten",
		PostalCode:   "57411",
	}

	result, err := uc.Create(context.Background(), input)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Equal(t, entity.ErrAddressLimitReached, err)
}

func TestCreateAddress_CountError_Fail(t *testing.T) {
	uc, repo, _ := setupAddressUsecase()
	repo.shouldFail = true

	input := usecase.CreateAddressInput{UserID: 1}
	result, err := uc.Create(context.Background(), input)
	assert.Nil(t, result)
	assert.Equal(t, entity.ErrFailedFetchAddress, err)
}

// ============================================================================
// GET ALL BY USER TESTS
// ============================================================================

func TestGetAllByUser_Success(t *testing.T) {
	uc, repo, _ := setupAddressUsecase()

	_ = repo.Create(context.Background(), &entity.UserAddress{UserID: 1, ReceiverName: "User 1 - Alamat 1"})
	_ = repo.Create(context.Background(), &entity.UserAddress{UserID: 1, ReceiverName: "User 1 - Alamat 2"})
	_ = repo.Create(context.Background(), &entity.UserAddress{UserID: 2, ReceiverName: "User 2 - Alamat 1"})

	res, err := uc.GetAllByUser(context.Background(), 1)
	assert.NoError(t, err)
	assert.Len(t, res, 2)
}

func TestGetAllByUser_Empty(t *testing.T) {
	uc, _, _ := setupAddressUsecase()

	res, err := uc.GetAllByUser(context.Background(), 99)
	assert.NoError(t, err)
	assert.Empty(t, res)
}

func TestGetAllByUser_DBError(t *testing.T) {
	uc, repo, _ := setupAddressUsecase()
	repo.shouldFail = true

	res, err := uc.GetAllByUser(context.Background(), 1)
	assert.Nil(t, res)
	assert.Equal(t, entity.ErrFailedFetchAddress, err)
}

// ============================================================================
// GET BY ID TESTS
// ============================================================================

func TestGetByID_Success(t *testing.T) {
	uc, repo, _ := setupAddressUsecase()

	addr := &entity.UserAddress{UserID: 1, ReceiverName: "Andi"}
	_ = repo.Create(context.Background(), addr)

	res, err := uc.GetByID(context.Background(), addr.ID, 1)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "Andi", res.ReceiverName)
}

func TestGetByID_NotFound_WrongUser(t *testing.T) {
	uc, repo, _ := setupAddressUsecase()

	addr := &entity.UserAddress{UserID: 1, ReceiverName: "Andi"}
	_ = repo.Create(context.Background(), addr)

	res, err := uc.GetByID(context.Background(), addr.ID, 2)
	assert.Nil(t, res)
	assert.Equal(t, entity.ErrAddressNotFound, err)
}

// ============================================================================
// UPDATE ADDRESS TESTS
// ============================================================================

func TestUpdateAddress_Success(t *testing.T) {
	uc, repo, _ := setupAddressUsecase()

	addr := &entity.UserAddress{
		UserID:       1,
		ReceiverName: "Nama Lama",
		IsPrimary:    false,
	}
	_ = repo.Create(context.Background(), addr)

	input := usecase.UpdateAddressInput{
		ID:           addr.ID,
		UserID:       1,
		ReceiverName: "Nama Baru",
		Phone:        "089999999",
		AddressLine:  "Jalan Baru",
		ProvinceID:   11,
		ProvinceName: "Jawa Timur",
		CityID:       200,
		CityName:     "Surabaya",
		PostalCode:   "60111",
		IsPrimary:    false,
	}

	res, err := uc.Update(context.Background(), input)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "Nama Baru", res.ReceiverName)
	assert.Equal(t, "Surabaya", res.CityName)
}

func TestUpdateAddress_NotFound(t *testing.T) {
	uc, _, _ := setupAddressUsecase()

	input := usecase.UpdateAddressInput{
		ID:     999,
		UserID: 1,
	}

	res, err := uc.Update(context.Background(), input)
	assert.Nil(t, res)
	assert.Equal(t, entity.ErrAddressNotFound, err)
}

// ============================================================================
// SET PRIMARY TESTS
// ============================================================================

func TestSetPrimary_Success(t *testing.T) {
	uc, repo, _ := setupAddressUsecase()

	addr1 := &entity.UserAddress{UserID: 1, IsPrimary: true}
	addr2 := &entity.UserAddress{UserID: 1, IsPrimary: false}
	_ = repo.Create(context.Background(), addr1)
	_ = repo.Create(context.Background(), addr2)

	err := uc.SetPrimary(context.Background(), addr2.ID, 1)
	assert.NoError(t, err)

	assert.False(t, repo.addresses[addr1.ID].IsPrimary)
	assert.True(t, repo.addresses[addr2.ID].IsPrimary)
}

func TestSetPrimary_AlreadyPrimary_NoAction(t *testing.T) {
	uc, repo, _ := setupAddressUsecase()

	addr := &entity.UserAddress{UserID: 1, IsPrimary: true}
	_ = repo.Create(context.Background(), addr)

	err := uc.SetPrimary(context.Background(), addr.ID, 1)
	assert.NoError(t, err)
	assert.True(t, repo.addresses[addr.ID].IsPrimary)
}

// ============================================================================
// DELETE ADDRESS TESTS
// ============================================================================

func TestDeleteAddress_Success_NonPrimary(t *testing.T) {
	uc, repo, _ := setupAddressUsecase()

	addrPrimary := &entity.UserAddress{UserID: 1, IsPrimary: true}
	addrSecondary := &entity.UserAddress{UserID: 1, IsPrimary: false}
	_ = repo.Create(context.Background(), addrPrimary)
	_ = repo.Create(context.Background(), addrSecondary)

	err := uc.Delete(context.Background(), addrSecondary.ID, 1)
	assert.NoError(t, err)
	assert.Nil(t, repo.addresses[addrSecondary.ID])
	assert.NotNil(t, repo.addresses[addrPrimary.ID])
}

func TestDeleteAddress_Block_PrimaryWhenOthersExist(t *testing.T) {
	uc, repo, _ := setupAddressUsecase()

	addr1 := &entity.UserAddress{UserID: 1, IsPrimary: true}
	addr2 := &entity.UserAddress{UserID: 1, IsPrimary: false}
	_ = repo.Create(context.Background(), addr1)
	_ = repo.Create(context.Background(), addr2)

	err := uc.Delete(context.Background(), addr1.ID, 1)
	assert.Error(t, err)
	assert.Equal(t, entity.ErrCannotDeletePrimaryAddress, err)
	assert.NotNil(t, repo.addresses[addr1.ID])
}

func TestDeleteAddress_Allow_PrimaryWhenOnlyOneLeft(t *testing.T) {
	uc, repo, _ := setupAddressUsecase()

	addrOnly := &entity.UserAddress{UserID: 1, IsPrimary: true}
	_ = repo.Create(context.Background(), addrOnly)

	err := uc.Delete(context.Background(), addrOnly.ID, 1)
	assert.NoError(t, err)
	assert.Nil(t, repo.addresses[addrOnly.ID])
}

func TestDeleteAddress_NotFound(t *testing.T) {
	uc, _, _ := setupAddressUsecase()

	err := uc.Delete(context.Background(), 999, 1)
	assert.Equal(t, entity.ErrAddressNotFound, err)
}
