package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/victoralfred/um_sys/internal/domain/auth"
	"github.com/victoralfred/um_sys/internal/domain/mfa"
	"github.com/victoralfred/um_sys/internal/domain/user"
	"github.com/victoralfred/um_sys/internal/services"
)

// Mock implementations
type MockMFARepository struct {
	mock.Mock
}

func (m *MockMFARepository) GetSettings(ctx context.Context, userID uuid.UUID) (*mfa.Settings, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mfa.Settings), args.Error(1)
}

func (m *MockMFARepository) SaveSettings(ctx context.Context, settings *mfa.Settings) error {
	args := m.Called(ctx, settings)
	return args.Error(0)
}

func (m *MockMFARepository) DeleteSettings(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockMFARepository) SaveChallenge(ctx context.Context, challenge *mfa.Challenge) error {
	args := m.Called(ctx, challenge)
	return args.Error(0)
}

func (m *MockMFARepository) GetChallenge(ctx context.Context, id uuid.UUID) (*mfa.Challenge, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mfa.Challenge), args.Error(1)
}

func (m *MockMFARepository) DeleteChallenge(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMFARepository) IncrementChallengeAttempts(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMFARepository) SaveBackupCode(ctx context.Context, userID uuid.UUID, code *mfa.BackupCode) error {
	args := m.Called(ctx, userID, code)
	return args.Error(0)
}

func (m *MockMFARepository) GetBackupCodes(ctx context.Context, userID uuid.UUID) ([]*mfa.BackupCode, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*mfa.BackupCode), args.Error(1)
}

func (m *MockMFARepository) MarkBackupCodeUsed(ctx context.Context, userID uuid.UUID, code string) error {
	args := m.Called(ctx, userID, code)
	return args.Error(0)
}

func (m *MockMFARepository) DeleteBackupCodes(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockMFARepository) LogAudit(ctx context.Context, log *mfa.AuditLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockMFARepository) GetAuditLogs(ctx context.Context, userID uuid.UUID, limit int) ([]*mfa.AuditLog, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*mfa.AuditLog), args.Error(1)
}

type MockTOTPProvider struct {
	mock.Mock
}

func (m *MockTOTPProvider) GenerateSecret() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockTOTPProvider) GenerateQRCode(secret, email string) (string, error) {
	args := m.Called(secret, email)
	return args.String(0), args.Error(1)
}

func (m *MockTOTPProvider) ValidateCode(secret, code string) (bool, error) {
	args := m.Called(secret, code)
	return args.Bool(0), args.Error(1)
}

func (m *MockTOTPProvider) GenerateCode(secret string) (string, error) {
	args := m.Called(secret)
	return args.String(0), args.Error(1)
}

type MockCodeGenerator struct {
	mock.Mock
}

func (m *MockCodeGenerator) GenerateNumericCode(length int) string {
	args := m.Called(length)
	return args.String(0)
}

func (m *MockCodeGenerator) GenerateAlphanumericCode(length int) string {
	args := m.Called(length)
	return args.String(0)
}

func (m *MockCodeGenerator) GenerateBackupCodes(count, length int) []string {
	args := m.Called(count, length)
	return args.Get(0).([]string)
}

type MockMFACache struct {
	mock.Mock
}

func (m *MockMFACache) StoreSetup(ctx context.Context, setupID string, data interface{}, expiry time.Duration) error {
	args := m.Called(ctx, setupID, data, expiry)
	return args.Error(0)
}

func (m *MockMFACache) GetSetup(ctx context.Context, setupID string) (interface{}, error) {
	args := m.Called(ctx, setupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0), args.Error(1)
}

func (m *MockMFACache) DeleteSetup(ctx context.Context, setupID string) error {
	args := m.Called(ctx, setupID)
	return args.Error(0)
}

func (m *MockMFACache) StoreChallenge(ctx context.Context, challenge *mfa.Challenge) error {
	args := m.Called(ctx, challenge)
	return args.Error(0)
}

func (m *MockMFACache) GetChallenge(ctx context.Context, id uuid.UUID) (*mfa.Challenge, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mfa.Challenge), args.Error(1)
}

func (m *MockMFACache) DeleteChallenge(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestMFAService_SetupMFA_TOTP(t *testing.T) {
	ctx := context.Background()

	t.Run("successful TOTP setup", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockMFARepository)
		mockUserRepo := new(MockUserRepository)
		mockTOTP := new(MockTOTPProvider)
		mockCodeGen := new(MockCodeGenerator)
		mockCache := new(MockMFACache)

		mfaService := services.NewMFAService(mockRepo, mockUserRepo, mockTOTP, nil, nil, mockCodeGen, mockCache, nil)

		userID := uuid.New()
		testUser := &user.User{
			ID:    userID,
			Email: "test@example.com",
		}

		req := &mfa.SetupRequest{
			UserID: userID,
			Method: mfa.MethodTOTP,
		}

		secret := "JBSWY3DPEHPK3PXP"
		qrCode := "data:image/png;base64,iVBORw0KGgoAAAANS..."
		backupCodes := []string{"ABCD1234", "EFGH5678", "IJKL9012"}

		mockUserRepo.On("GetByID", ctx, userID).Return(testUser, nil)
		mockRepo.On("GetSettings", ctx, userID).Return(nil, mfa.ErrSettingsNotFound)
		mockTOTP.On("GenerateSecret").Return(secret, nil)
		mockTOTP.On("GenerateQRCode", secret, testUser.Email).Return(qrCode, nil)
		mockCodeGen.On("GenerateBackupCodes", 10, 8).Return(backupCodes)
		mockCache.On("StoreSetup", ctx, mock.AnythingOfType("string"), mock.Anything, mfa.SetupExpiry).Return(nil)

		// Act
		resp, err := mfaService.SetupMFA(ctx, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, mfa.MethodTOTP, resp.Method)
		assert.Equal(t, secret, resp.Secret)
		assert.Equal(t, qrCode, resp.QRCode)
		assert.Equal(t, backupCodes, resp.BackupCodes)
		assert.NotEmpty(t, resp.SetupID)

		mockRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockTOTP.AssertExpectations(t)
		mockCodeGen.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("MFA already enabled", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockMFARepository)
		mockUserRepo := new(MockUserRepository)
		mockTOTP := new(MockTOTPProvider)
		mockCodeGen := new(MockCodeGenerator)
		mockCache := new(MockMFACache)

		mfaService := services.NewMFAService(mockRepo, mockUserRepo, mockTOTP, nil, nil, mockCodeGen, mockCache, nil)

		userID := uuid.New()

		req := &mfa.SetupRequest{
			UserID: userID,
			Method: mfa.MethodTOTP,
		}

		existingSettings := &mfa.Settings{
			UserID:  userID,
			Enabled: true,
		}

		mockRepo.On("GetSettings", ctx, userID).Return(existingSettings, nil)

		// Act
		resp, err := mfaService.SetupMFA(ctx, req)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, mfa.ErrMFAAlreadyEnabled, err)
		assert.Nil(t, resp)

		mockRepo.AssertExpectations(t)
	})
}

func TestMFAService_VerifySetup(t *testing.T) {
	ctx := context.Background()

	t.Run("successful setup verification", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockMFARepository)
		mockUserRepo := new(MockUserRepository)
		mockTOTP := new(MockTOTPProvider)
		mockCodeGen := new(MockCodeGenerator)
		mockCache := new(MockMFACache)

		mfaService := services.NewMFAService(mockRepo, mockUserRepo, mockTOTP, nil, nil, mockCodeGen, mockCache, nil)

		userID := uuid.New()
		setupID := uuid.New().String()
		code := "123456"
		secret := "JBSWY3DPEHPK3PXP"

		req := &mfa.VerifySetupRequest{
			UserID:  userID,
			SetupID: setupID,
			Code:    code,
		}

		setupData := map[string]interface{}{
			"user_id":      userID.String(),
			"method":       string(mfa.MethodTOTP),
			"secret":       secret,
			"backup_codes": []string{"ABCD1234", "EFGH5678"},
		}

		mockCache.On("GetSetup", ctx, setupID).Return(setupData, nil)
		mockTOTP.On("ValidateCode", secret, code).Return(true, nil)
		mockRepo.On("SaveSettings", ctx, mock.AnythingOfType("*mfa.Settings")).Return(nil)
		mockRepo.On("DeleteBackupCodes", ctx, userID).Return(nil)
		mockRepo.On("SaveBackupCode", ctx, userID, mock.AnythingOfType("*mfa.BackupCode")).Return(nil).Times(2)
		mockCache.On("DeleteSetup", ctx, setupID).Return(nil)
		mockRepo.On("LogAudit", ctx, mock.AnythingOfType("*mfa.AuditLog")).Return(nil)

		// Act
		err := mfaService.VerifySetup(ctx, req)

		// Assert
		assert.NoError(t, err)

		mockCache.AssertExpectations(t)
		mockTOTP.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid code", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockMFARepository)
		mockUserRepo := new(MockUserRepository)
		mockTOTP := new(MockTOTPProvider)
		mockCodeGen := new(MockCodeGenerator)
		mockCache := new(MockMFACache)

		mfaService := services.NewMFAService(mockRepo, mockUserRepo, mockTOTP, nil, nil, mockCodeGen, mockCache, nil)

		userID := uuid.New()
		setupID := uuid.New().String()
		code := "000000"
		secret := "JBSWY3DPEHPK3PXP"

		req := &mfa.VerifySetupRequest{
			UserID:  userID,
			SetupID: setupID,
			Code:    code,
		}

		setupData := map[string]interface{}{
			"user_id": userID.String(),
			"method":  string(mfa.MethodTOTP),
			"secret":  secret,
		}

		mockCache.On("GetSetup", ctx, setupID).Return(setupData, nil)
		mockTOTP.On("ValidateCode", secret, code).Return(false, nil)

		// Act
		err := mfaService.VerifySetup(ctx, req)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, mfa.ErrInvalidCode, err)

		mockCache.AssertExpectations(t)
		mockTOTP.AssertExpectations(t)
	})
}

func TestMFAService_VerifyCode(t *testing.T) {
	ctx := context.Background()

	t.Run("successful TOTP verification", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockMFARepository)
		mockUserRepo := new(MockUserRepository)
		mockTOTP := new(MockTOTPProvider)
		mockCodeGen := new(MockCodeGenerator)
		mockCache := new(MockMFACache)

		mfaService := services.NewMFAService(mockRepo, mockUserRepo, mockTOTP, nil, nil, mockCodeGen, mockCache, nil)

		userID := uuid.New()
		code := "123456"
		secret := "JBSWY3DPEHPK3PXP"

		req := &mfa.VerifyRequest{
			UserID: userID,
			Method: mfa.MethodTOTP,
			Code:   code,
		}

		settings := &mfa.Settings{
			UserID:        userID,
			Enabled:       true,
			TOTPSecret:    secret,
			Methods:       []mfa.Method{mfa.MethodTOTP},
			PrimaryMethod: mfa.MethodTOTP,
		}

		mockRepo.On("GetSettings", ctx, userID).Return(settings, nil)
		mockTOTP.On("ValidateCode", secret, code).Return(true, nil)
		mockRepo.On("SaveSettings", ctx, mock.AnythingOfType("*mfa.Settings")).Return(nil)
		mockRepo.On("LogAudit", ctx, mock.AnythingOfType("*mfa.AuditLog")).Return(nil)

		// Act
		resp, err := mfaService.VerifyCode(ctx, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Valid)
		assert.Equal(t, mfa.MethodTOTP, resp.Method)

		mockRepo.AssertExpectations(t)
		mockTOTP.AssertExpectations(t)
	})

	t.Run("MFA not enabled", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockMFARepository)
		mockUserRepo := new(MockUserRepository)
		mockTOTP := new(MockTOTPProvider)
		mockCodeGen := new(MockCodeGenerator)
		mockCache := new(MockMFACache)

		mfaService := services.NewMFAService(mockRepo, mockUserRepo, mockTOTP, nil, nil, mockCodeGen, mockCache, nil)

		userID := uuid.New()

		req := &mfa.VerifyRequest{
			UserID: userID,
			Method: mfa.MethodTOTP,
			Code:   "123456",
		}

		mockRepo.On("GetSettings", ctx, userID).Return(nil, mfa.ErrSettingsNotFound)

		// Act
		resp, err := mfaService.VerifyCode(ctx, req)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, mfa.ErrMFANotEnabled, err)
		assert.Nil(t, resp)

		mockRepo.AssertExpectations(t)
	})
}

func TestMFAService_DisableMFA(t *testing.T) {
	ctx := context.Background()

	t.Run("successful disable", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockMFARepository)
		mockUserRepo := new(MockUserRepository)
		mockTOTP := new(MockTOTPProvider)
		mockCodeGen := new(MockCodeGenerator)
		mockCache := new(MockMFACache)
		mockHasher := new(MockPasswordHasher)

		mfaService := services.NewMFAService(mockRepo, mockUserRepo, mockTOTP, nil, nil, mockCodeGen, mockCache, mockHasher)

		userID := uuid.New()
		password := "SecurePass123!"
		code := "123456"
		secret := "JBSWY3DPEHPK3PXP"

		req := &mfa.DisableRequest{
			UserID:   userID,
			Password: password,
			Code:     code,
		}

		testUser := &user.User{
			ID:           userID,
			PasswordHash: "$2a$10$hashedpassword",
		}

		settings := &mfa.Settings{
			UserID:        userID,
			Enabled:       true,
			TOTPSecret:    secret,
			Methods:       []mfa.Method{mfa.MethodTOTP},
			PrimaryMethod: mfa.MethodTOTP,
		}

		mockUserRepo.On("GetByID", ctx, userID).Return(testUser, nil)
		mockHasher.On("VerifyPassword", password, testUser.PasswordHash).Return(nil)
		mockRepo.On("GetSettings", ctx, userID).Return(settings, nil).Times(2) // Called twice: once in DisableMFA, once in VerifyCode
		mockTOTP.On("ValidateCode", secret, code).Return(true, nil)
		mockRepo.On("SaveSettings", ctx, mock.AnythingOfType("*mfa.Settings")).Return(nil)      // For updating last used
		mockRepo.On("LogAudit", ctx, mock.AnythingOfType("*mfa.AuditLog")).Return(nil).Times(2) // For verify and disable
		mockRepo.On("DeleteSettings", ctx, userID).Return(nil)
		mockRepo.On("DeleteBackupCodes", ctx, userID).Return(nil)

		// Act
		err := mfaService.DisableMFA(ctx, req)

		// Assert
		assert.NoError(t, err)

		mockUserRepo.AssertExpectations(t)
		mockHasher.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
		mockTOTP.AssertExpectations(t)
	})

	t.Run("invalid password", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockMFARepository)
		mockUserRepo := new(MockUserRepository)
		mockTOTP := new(MockTOTPProvider)
		mockCodeGen := new(MockCodeGenerator)
		mockCache := new(MockMFACache)
		mockHasher := new(MockPasswordHasher)

		mfaService := services.NewMFAService(mockRepo, mockUserRepo, mockTOTP, nil, nil, mockCodeGen, mockCache, mockHasher)

		userID := uuid.New()
		password := "WrongPassword"

		req := &mfa.DisableRequest{
			UserID:   userID,
			Password: password,
		}

		testUser := &user.User{
			ID:           userID,
			PasswordHash: "$2a$10$hashedpassword",
		}

		mockUserRepo.On("GetByID", ctx, userID).Return(testUser, nil)
		mockHasher.On("VerifyPassword", password, testUser.PasswordHash).Return(auth.ErrInvalidCredentials)

		// Act
		err := mfaService.DisableMFA(ctx, req)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, auth.ErrInvalidCredentials, err)

		mockUserRepo.AssertExpectations(t)
		mockHasher.AssertExpectations(t)
	})
}
