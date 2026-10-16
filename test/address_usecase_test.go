package test

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type MockAddressRepository struct {
	mock.Mock
}

func (m *MockAddressRepository) WithTx(tx *gorm.DB) repository.AddressRepository {
	args := m.Called(tx)
	if res := args.Get(0); res != nil {
		return res.(repository.AddressRepository)
	}
	return m
}

func (m *MockAddressRepository) Create(ctx context.Context, address *entity.Address) error {
	args := m.Called(ctx, address)
	if args.Get(0) == nil {
		address.ID = 1
	}
	return args.Error(0)
}

func (m *MockAddressRepository) FindByID(ctx context.Context, id uint) (*entity.Address, error) {
	args := m.Called(ctx, id)
	if res := args.Get(0); res != nil {
		return res.(*entity.Address), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAddressRepository) FindByUserID(ctx context.Context, userID uint) ([]entity.Address, error) {
	args := m.Called(ctx, userID)
	if res := args.Get(0); res != nil {
		return res.([]entity.Address), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAddressRepository) FindPrimaryByUserID(ctx context.Context, userID uint) (*entity.Address, error) {
	args := m.Called(ctx, userID)
	if res := args.Get(0); res != nil {
		return res.(*entity.Address), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAddressRepository) Update(ctx context.Context, address *entity.Address) error {
	args := m.Called(ctx, address)
	return args.Error(0)
}

func (m *MockAddressRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAddressRepository) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAddressRepository) ResetPrimaryStatus(ctx context.Context, userID uint) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockAddressRepository) SetPrimary(ctx context.Context, userID, addressID uint) error {
	args := m.Called(ctx, userID, addressID)
	return args.Error(0)
}

func setupGormMock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 db,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	assert.NoError(t, err)

	return gormDB, mock
}

func TestAddressUsecase_CreateAddress(t *testing.T) {
	ctx := context.Background()
	userID := uint(10)

	sampleInput := usecase.CreateAddressInput{
		ReceiverName:    "Budi Santoso",
		Phone:           "08123456789",
		AddressLine:     "Jl. Pemuda No. 45",
		BiteshipAreaID:  "IDNP6IDNC148IDND843IDZ12250",
		ProvinceName:    "DKI Jakarta",
		CityName:        "Jakarta Selatan",
		DistrictName:    "Kebayoran Baru",
		SubdistrictName: "Kramat Pela",
		PostalCode:      "12250",
		IsPrimary:       false,
	}

	t.Run("success create first address (automatically promoted to primary)", func(t *testing.T) {
		gormDB, sqlMock := setupGormMock(t)
		mockRepo := new(MockAddressRepository)
		uc := usecase.NewAddressUsecase(gormDB, mockRepo)

		sqlMock.ExpectBegin()
		sqlMock.ExpectCommit()

		mockRepo.On("CountByUserID", ctx, userID).Return(int64(0), nil).Once()
		mockRepo.On("WithTx", mock.Anything).Return(mockRepo).Once()
		mockRepo.On("Create", ctx, mock.MatchedBy(func(addr *entity.Address) bool {
			return addr.UserID == userID && addr.IsPrimary == true && addr.BiteshipAreaID == sampleInput.BiteshipAreaID
		})).Return(nil).Once()

		result, err := uc.CreateAddress(ctx, userID, sampleInput)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsPrimary)
		assert.Equal(t, sampleInput.ReceiverName, result.ReceiverName)
		mockRepo.AssertExpectations(t)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("success create additional address with is_primary true (resets old primary)", func(t *testing.T) {
		gormDB, sqlMock := setupGormMock(t)
		mockRepo := new(MockAddressRepository)
		uc := usecase.NewAddressUsecase(gormDB, mockRepo)

		sqlMock.ExpectBegin()
		sqlMock.ExpectCommit()

		inputPrimary := sampleInput
		inputPrimary.IsPrimary = true

		mockRepo.On("CountByUserID", ctx, userID).Return(int64(2), nil).Once()
		mockRepo.On("WithTx", mock.Anything).Return(mockRepo).Once()
		mockRepo.On("ResetPrimaryStatus", ctx, userID).Return(nil).Once()
		mockRepo.On("Create", ctx, mock.MatchedBy(func(addr *entity.Address) bool {
			return addr.UserID == userID && addr.IsPrimary == true
		})).Return(nil).Once()

		result, err := uc.CreateAddress(ctx, userID, inputPrimary)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsPrimary)
		mockRepo.AssertExpectations(t)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("failed when maximum address quota reached (limit 5)", func(t *testing.T) {
		gormDB, _ := setupGormMock(t)
		mockRepo := new(MockAddressRepository)
		uc := usecase.NewAddressUsecase(gormDB, mockRepo)

		mockRepo.On("CountByUserID", ctx, userID).Return(int64(5), nil).Once()

		result, err := uc.CreateAddress(ctx, userID, sampleInput)
		assert.ErrorIs(t, err, entity.ErrAddressLimitReached)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("failed on counting error", func(t *testing.T) {
		gormDB, _ := setupGormMock(t)
		mockRepo := new(MockAddressRepository)
		uc := usecase.NewAddressUsecase(gormDB, mockRepo)

		mockRepo.On("CountByUserID", ctx, userID).Return(int64(0), errors.New("db connection failure")).Once()

		result, err := uc.CreateAddress(ctx, userID, sampleInput)
		assert.Error(t, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestAddressUsecase_GetAddressByID(t *testing.T) {
	ctx := context.Background()
	gormDB, _ := setupGormMock(t)
	mockRepo := new(MockAddressRepository)
	uc := usecase.NewAddressUsecase(gormDB, mockRepo)
	userID := uint(10)
	addressID := uint(1)

	t.Run("success get address by id", func(t *testing.T) {
		existing := &entity.Address{
			UserID:          userID,
			ReceiverName:    "Budi Santoso",
			AddressLine:     "Jl. Pemuda No. 45",
			BiteshipAreaID:  "IDNP6IDNC148IDND843IDZ12250",
			CityName:        "Jakarta Selatan",
			DistrictName:    "Kebayoran Baru",
			SubdistrictName: "Kramat Pela",
			PostalCode:      "12250",
		}
		existing.ID = addressID

		mockRepo.On("FindByID", ctx, addressID).Return(existing, nil).Once()

		res, err := uc.GetAddressByID(ctx, userID, addressID)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, addressID, res.ID)
		assert.Equal(t, "Jl. Pemuda No. 45", res.AddressLine)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error address not found", func(t *testing.T) {
		mockRepo.On("FindByID", ctx, addressID).Return(nil, nil).Once()

		res, err := uc.GetAddressByID(ctx, userID, addressID)
		assert.ErrorIs(t, err, entity.ErrAddressNotFound)
		assert.Nil(t, res)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error unauthorized different user", func(t *testing.T) {
		existing := &entity.Address{
			UserID: uint(999),
		}
		existing.ID = addressID

		mockRepo.On("FindByID", ctx, addressID).Return(existing, nil).Once()

		res, err := uc.GetAddressByID(ctx, userID, addressID)
		assert.ErrorIs(t, err, entity.ErrUnauthorizedAddress)
		assert.Nil(t, res)
		mockRepo.AssertExpectations(t)
	})
}

func TestAddressUsecase_GetUserAddresses(t *testing.T) {
	ctx := context.Background()
	gormDB, _ := setupGormMock(t)
	mockRepo := new(MockAddressRepository)
	uc := usecase.NewAddressUsecase(gormDB, mockRepo)
	userID := uint(10)

	t.Run("success get list of addresses in compact format", func(t *testing.T) {
		addr1 := entity.Address{
			UserID:       userID,
			ReceiverName: "Rumah",
			Phone:        "08123456789",
			CityName:     "Jakarta Selatan",
			DistrictName: "Kebayoran Baru",
			IsPrimary:    true,
		}
		addr1.ID = 1

		addr2 := entity.Address{
			UserID:       userID,
			ReceiverName: "Kantor",
			Phone:        "08987654321",
			CityName:     "Jakarta Pusat",
			DistrictName: "Menteng",
			IsPrimary:    false,
		}
		addr2.ID = 2

		addresses := []entity.Address{addr1, addr2}
		mockRepo.On("FindByUserID", ctx, userID).Return(addresses, nil).Once()

		results, err := uc.GetUserAddresses(ctx, userID)
		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, uint(1), results[0].ID)
		assert.Equal(t, "Rumah", results[0].ReceiverName)
		assert.Equal(t, "Jakarta Selatan", results[0].CityName)
		assert.True(t, results[0].IsPrimary)
		assert.Equal(t, uint(2), results[1].ID)
		assert.Equal(t, "Kantor", results[1].ReceiverName)
		assert.False(t, results[1].IsPrimary)
		mockRepo.AssertExpectations(t)
	})
}

func TestAddressUsecase_GetPrimaryAddress(t *testing.T) {
	ctx := context.Background()
	gormDB, _ := setupGormMock(t)
	mockRepo := new(MockAddressRepository)
	uc := usecase.NewAddressUsecase(gormDB, mockRepo)
	userID := uint(10)

	t.Run("success get primary address", func(t *testing.T) {
		primaryAddr := &entity.Address{
			UserID:       userID,
			ReceiverName: "Rumah",
			IsPrimary:    true,
		}
		primaryAddr.ID = 1

		mockRepo.On("FindPrimaryByUserID", ctx, userID).Return(primaryAddr, nil).Once()

		res, err := uc.GetPrimaryAddress(ctx, userID)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.True(t, res.IsPrimary)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error primary address not found", func(t *testing.T) {
		mockRepo.On("FindPrimaryByUserID", ctx, userID).Return(nil, nil).Once()

		res, err := uc.GetPrimaryAddress(ctx, userID)
		assert.ErrorIs(t, err, entity.ErrAddressNotFound)
		assert.Nil(t, res)
		mockRepo.AssertExpectations(t)
	})
}

func TestAddressUsecase_UpdateAddress(t *testing.T) {
	ctx := context.Background()
	userID := uint(10)
	addressID := uint(1)

	updateInput := usecase.UpdateAddressInput{
		ReceiverName:    "Budi Baru",
		Phone:           "089988776655",
		AddressLine:     "Jl. Sudirman No. 1",
		BiteshipAreaID:  "IDNP6IDNC148IDND843IDZ12250",
		ProvinceName:    "DKI Jakarta",
		CityName:        "Jakarta Pusat",
		DistrictName:    "Tanah Abang",
		SubdistrictName: "Kebon Kacang",
		PostalCode:      "10250",
		IsPrimary:       true,
	}

	t.Run("success update address and promote to primary", func(t *testing.T) {
		gormDB, sqlMock := setupGormMock(t)
		mockRepo := new(MockAddressRepository)
		uc := usecase.NewAddressUsecase(gormDB, mockRepo)

		sqlMock.ExpectBegin()
		sqlMock.ExpectCommit()

		existing := &entity.Address{
			UserID:    userID,
			IsPrimary: false,
		}
		existing.ID = addressID

		mockRepo.On("FindByID", ctx, addressID).Return(existing, nil).Once()
		mockRepo.On("WithTx", mock.Anything).Return(mockRepo).Once()
		mockRepo.On("ResetPrimaryStatus", ctx, userID).Return(nil).Once()
		mockRepo.On("Update", ctx, mock.MatchedBy(func(addr *entity.Address) bool {
			return addr.ID == addressID && addr.ReceiverName == updateInput.ReceiverName && addr.IsPrimary == true
		})).Return(nil).Once()

		res, err := uc.UpdateAddress(ctx, userID, addressID, updateInput)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, updateInput.ReceiverName, res.ReceiverName)
		assert.True(t, res.IsPrimary)
		mockRepo.AssertExpectations(t)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("error address not found on update", func(t *testing.T) {
		gormDB, _ := setupGormMock(t)
		mockRepo := new(MockAddressRepository)
		uc := usecase.NewAddressUsecase(gormDB, mockRepo)

		mockRepo.On("FindByID", ctx, addressID).Return(nil, nil).Once()

		res, err := uc.UpdateAddress(ctx, userID, addressID, updateInput)
		assert.ErrorIs(t, err, entity.ErrAddressNotFound)
		assert.Nil(t, res)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error unauthorized on update", func(t *testing.T) {
		gormDB, _ := setupGormMock(t)
		mockRepo := new(MockAddressRepository)
		uc := usecase.NewAddressUsecase(gormDB, mockRepo)

		existing := &entity.Address{
			UserID: uint(999),
		}
		existing.ID = addressID

		mockRepo.On("FindByID", ctx, addressID).Return(existing, nil).Once()

		res, err := uc.UpdateAddress(ctx, userID, addressID, updateInput)
		assert.ErrorIs(t, err, entity.ErrUnauthorizedAddress)
		assert.Nil(t, res)
		mockRepo.AssertExpectations(t)
	})
}

func TestAddressUsecase_DeleteAddress(t *testing.T) {
	ctx := context.Background()
	gormDB, _ := setupGormMock(t)
	mockRepo := new(MockAddressRepository)
	uc := usecase.NewAddressUsecase(gormDB, mockRepo)
	userID := uint(10)
	addressID := uint(2)

	t.Run("success delete non-primary address", func(t *testing.T) {
		existing := &entity.Address{
			UserID:    userID,
			IsPrimary: false,
		}
		existing.ID = addressID

		mockRepo.On("FindByID", ctx, addressID).Return(existing, nil).Once()
		mockRepo.On("Delete", ctx, addressID).Return(nil).Once()

		err := uc.DeleteAddress(ctx, userID, addressID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error cannot delete primary address", func(t *testing.T) {
		existing := &entity.Address{
			UserID:    userID,
			IsPrimary: true,
		}
		existing.ID = addressID

		mockRepo.On("FindByID", ctx, addressID).Return(existing, nil).Once()

		err := uc.DeleteAddress(ctx, userID, addressID)
		assert.ErrorIs(t, err, entity.ErrPrimaryAddressDelete)
		mockRepo.AssertExpectations(t)
	})
}

func TestAddressUsecase_SetPrimaryAddress(t *testing.T) {
	ctx := context.Background()
	gormDB, _ := setupGormMock(t)
	mockRepo := new(MockAddressRepository)
	uc := usecase.NewAddressUsecase(gormDB, mockRepo)
	userID := uint(10)
	addressID := uint(3)

	t.Run("success set primary address", func(t *testing.T) {
		mockRepo.On("SetPrimary", ctx, userID, addressID).Return(nil).Once()

		err := uc.SetPrimaryAddress(ctx, userID, addressID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}
