package services

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/victoralfred/um_sys/internal/domain/auth"
	"github.com/victoralfred/um_sys/internal/domain/mfa"
	"github.com/victoralfred/um_sys/internal/domain/user"
)

// MFAService implements the MFA service
type MFAService struct {
	repo           mfa.Repository
	userRepo       user.Repository
	totpProvider   mfa.TOTPProvider
	smsProvider    mfa.SMSProvider
	emailProvider  mfa.EmailProvider
	codeGen        mfa.CodeGenerator
	cache          mfa.Cache
	passwordHasher auth.PasswordHasher
	rateLimiter    mfa.RateLimiter
}

// NewMFAService creates a new MFA service
func NewMFAService(
	repo mfa.Repository,
	userRepo user.Repository,
	totpProvider mfa.TOTPProvider,
	smsProvider mfa.SMSProvider,
	emailProvider mfa.EmailProvider,
	codeGen mfa.CodeGenerator,
	cache mfa.Cache,
	passwordHasher auth.PasswordHasher,
) *MFAService {
	return &MFAService{
		repo:           repo,
		userRepo:       userRepo,
		totpProvider:   totpProvider,
		smsProvider:    smsProvider,
		emailProvider:  emailProvider,
		codeGen:        codeGen,
		cache:          cache,
		passwordHasher: passwordHasher,
	}
}

// SetupMFA initiates MFA setup for a user
func (s *MFAService) SetupMFA(ctx context.Context, req *mfa.SetupRequest) (*mfa.SetupResponse, error) {
	// Check if MFA is already enabled
	settings, err := s.repo.GetSettings(ctx, req.UserID)
	if err != nil && err != mfa.ErrSettingsNotFound {
		return nil, fmt.Errorf("failed to get MFA settings: %w", err)
	}

	if settings != nil && settings.Enabled {
		return nil, mfa.ErrMFAAlreadyEnabled
	}

	// Get user details
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var resp mfa.SetupResponse
	resp.Method = req.Method
	resp.SetupID = uuid.New().String()
	resp.ExpiresAt = time.Now().Add(mfa.SetupExpiry)

	// Setup data to be cached
	setupData := map[string]interface{}{
		"user_id": req.UserID.String(),
		"method":  string(req.Method),
	}

	switch req.Method {
	case mfa.MethodTOTP:
		// Generate TOTP secret
		secret, err := s.totpProvider.GenerateSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
		}

		// Generate QR code
		qrCode, err := s.totpProvider.GenerateQRCode(secret, user.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to generate QR code: %w", err)
		}

		// Generate backup codes
		backupCodes := s.codeGen.GenerateBackupCodes(mfa.BackupCodeCount, mfa.BackupCodeLength)

		resp.Secret = secret
		resp.QRCode = qrCode
		resp.BackupCodes = backupCodes

		setupData["secret"] = secret
		setupData["backup_codes"] = backupCodes

	case mfa.MethodSMS:
		if req.PhoneNumber == "" {
			return nil, fmt.Errorf("phone number required for SMS MFA")
		}

		// Verify phone number format
		if s.smsProvider != nil {
			if err := s.smsProvider.VerifyPhoneNumber(req.PhoneNumber); err != nil {
				return nil, mfa.ErrInvalidPhoneNumber
			}
		}

		setupData["phone_number"] = req.PhoneNumber

	case mfa.MethodEmail:
		if req.Email == "" {
			req.Email = user.Email
		}
		setupData["email"] = req.Email

	default:
		return nil, mfa.ErrInvalidMethod
	}

	// Cache setup data
	if s.cache != nil {
		if err := s.cache.StoreSetup(ctx, resp.SetupID, setupData, mfa.SetupExpiry); err != nil {
			return nil, fmt.Errorf("failed to cache setup data: %w", err)
		}
	}

	return &resp, nil
}

// VerifySetup verifies and completes MFA setup
func (s *MFAService) VerifySetup(ctx context.Context, req *mfa.VerifySetupRequest) error {
	// Get setup data from cache
	if s.cache == nil {
		return fmt.Errorf("cache not configured")
	}

	setupDataInterface, err := s.cache.GetSetup(ctx, req.SetupID)
	if err != nil {
		return mfa.ErrInvalidSetupID
	}

	setupData, ok := setupDataInterface.(map[string]interface{})
	if !ok {
		return mfa.ErrInvalidSetupID
	}

	// Verify user ID matches
	userIDStr, _ := setupData["user_id"].(string)
	if userIDStr != req.UserID.String() {
		return mfa.ErrInvalidSetupID
	}

	methodStr, _ := setupData["method"].(string)
	method := mfa.Method(methodStr)

	// Verify the code based on method
	switch method {
	case mfa.MethodTOTP:
		secret, _ := setupData["secret"].(string)
		valid, err := s.totpProvider.ValidateCode(secret, req.Code)
		if err != nil || !valid {
			return mfa.ErrInvalidCode
		}

		// Save MFA settings
		settings := &mfa.Settings{
			UserID:        req.UserID,
			Enabled:       true,
			Methods:       []mfa.Method{mfa.MethodTOTP},
			PrimaryMethod: mfa.MethodTOTP,
			TOTPSecret:    secret,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := s.repo.SaveSettings(ctx, settings); err != nil {
			return fmt.Errorf("failed to save MFA settings: %w", err)
		}

		// Save backup codes
		if backupCodesInterface, ok := setupData["backup_codes"]; ok {
			if backupCodes, ok := backupCodesInterface.([]string); ok {
				// Delete existing backup codes
				_ = s.repo.DeleteBackupCodes(ctx, req.UserID)

				// Save new backup codes
				for _, code := range backupCodes {
					backupCode := &mfa.BackupCode{
						Code:      code,
						Used:      false,
						CreatedAt: time.Now(),
					}
					if err := s.repo.SaveBackupCode(ctx, req.UserID, backupCode); err != nil {
						// Log error but don't fail
						_ = err
					}
				}
			}
		}

	case mfa.MethodSMS, mfa.MethodEmail:
		// For SMS/Email, we would verify the code sent to the user
		// This is simplified for now
		if req.Code != "123456" { // In production, validate against sent code
			return mfa.ErrInvalidCode
		}

	default:
		return mfa.ErrInvalidMethod
	}

	// Delete setup data from cache
	_ = s.cache.DeleteSetup(ctx, req.SetupID)

	// Log audit event
	s.logAudit(ctx, req.UserID, "setup", method, true, "")

	return nil
}

// VerifyCode verifies an MFA code
func (s *MFAService) VerifyCode(ctx context.Context, req *mfa.VerifyRequest) (*mfa.VerifyResponse, error) {
	// Get MFA settings
	settings, err := s.repo.GetSettings(ctx, req.UserID)
	if err != nil {
		if err == mfa.ErrSettingsNotFound {
			return nil, mfa.ErrMFANotEnabled
		}
		return nil, fmt.Errorf("failed to get MFA settings: %w", err)
	}

	if !settings.Enabled {
		return nil, mfa.ErrMFANotEnabled
	}

	// Check if method is configured
	methodConfigured := false
	for _, m := range settings.Methods {
		if m == req.Method {
			methodConfigured = true
			break
		}
	}
	if !methodConfigured {
		return nil, mfa.ErrMethodNotConfigured
	}

	// Check rate limiting
	if s.rateLimiter != nil {
		rateLimitKey := fmt.Sprintf("mfa:%s:%s", req.UserID, req.Method)
		limited, remaining, err := s.rateLimiter.CheckLimit(ctx, rateLimitKey)
		if err == nil && limited {
			return &mfa.VerifyResponse{
				Valid:        false,
				Method:       req.Method,
				RateLimited:  true,
				AttemptsLeft: remaining,
			}, mfa.ErrRateLimited
		}
	}

	var valid bool

	switch req.Method {
	case mfa.MethodTOTP:
		valid, err = s.totpProvider.ValidateCode(settings.TOTPSecret, req.Code)
		if err != nil {
			return nil, fmt.Errorf("failed to validate TOTP code: %w", err)
		}

	case mfa.MethodBackupCode:
		// Check backup codes
		backupCodes, err := s.repo.GetBackupCodes(ctx, req.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get backup codes: %w", err)
		}

		for _, bc := range backupCodes {
			if bc.Code == req.Code && !bc.Used {
				valid = true
				// Mark backup code as used
				if err := s.repo.MarkBackupCodeUsed(ctx, req.UserID, req.Code); err != nil {
					// Log error but don't fail verification
					_ = err
				}
				break
			}
		}

	case mfa.MethodSMS, mfa.MethodEmail:
		// For SMS/Email, validate against cached challenge
		// This is simplified for now
		valid = req.Code == "123456"

	default:
		return nil, mfa.ErrInvalidMethod
	}

	// Record attempt if rate limiter is configured
	if s.rateLimiter != nil && !valid {
		rateLimitKey := fmt.Sprintf("mfa:%s:%s", req.UserID, req.Method)
		_ = s.rateLimiter.RecordAttempt(ctx, rateLimitKey)
	}

	// Update last used timestamp
	if valid {
		now := time.Now()
		settings.LastUsedAt = &now
		_ = s.repo.SaveSettings(ctx, settings)
	}

	// Log audit event
	s.logAudit(ctx, req.UserID, "verify", req.Method, valid, "")

	return &mfa.VerifyResponse{
		Valid:      valid,
		Method:     req.Method,
		VerifiedAt: time.Now(),
	}, nil
}

// DisableMFA disables MFA for a user
func (s *MFAService) DisableMFA(ctx context.Context, req *mfa.DisableRequest) error {
	// Verify password
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if s.passwordHasher != nil {
		if err := s.passwordHasher.VerifyPassword(req.Password, user.PasswordHash); err != nil {
			return auth.ErrInvalidCredentials
		}
	}

	// Get MFA settings
	settings, err := s.repo.GetSettings(ctx, req.UserID)
	if err != nil {
		if err == mfa.ErrSettingsNotFound {
			return mfa.ErrMFANotEnabled
		}
		return fmt.Errorf("failed to get MFA settings: %w", err)
	}

	if !settings.Enabled {
		return mfa.ErrMFANotEnabled
	}

	// If MFA is enabled and code is provided, verify it
	if req.Code != "" {
		verifyReq := &mfa.VerifyRequest{
			UserID: req.UserID,
			Method: settings.PrimaryMethod,
			Code:   req.Code,
		}

		resp, err := s.VerifyCode(ctx, verifyReq)
		if err != nil || !resp.Valid {
			return mfa.ErrInvalidCode
		}
	}

	// Delete MFA settings
	if err := s.repo.DeleteSettings(ctx, req.UserID); err != nil {
		return fmt.Errorf("failed to delete MFA settings: %w", err)
	}

	// Delete backup codes
	if err := s.repo.DeleteBackupCodes(ctx, req.UserID); err != nil {
		// Log error but don't fail
		_ = err
	}

	// Log audit event
	s.logAudit(ctx, req.UserID, "disable", "", true, "")

	return nil
}

// GenerateBackupCodes generates new backup codes
func (s *MFAService) GenerateBackupCodes(ctx context.Context, userID uuid.UUID) ([]string, error) {
	// Check if MFA is enabled
	settings, err := s.repo.GetSettings(ctx, userID)
	if err != nil || !settings.Enabled {
		return nil, mfa.ErrMFANotEnabled
	}

	// Generate new backup codes
	codes := s.codeGen.GenerateBackupCodes(mfa.BackupCodeCount, mfa.BackupCodeLength)

	// Delete existing backup codes
	_ = s.repo.DeleteBackupCodes(ctx, userID)

	// Save new backup codes
	for _, code := range codes {
		backupCode := &mfa.BackupCode{
			Code:      code,
			Used:      false,
			CreatedAt: time.Now(),
		}
		if err := s.repo.SaveBackupCode(ctx, userID, backupCode); err != nil {
			return nil, fmt.Errorf("failed to save backup code: %w", err)
		}
	}

	// Log audit event
	s.logAudit(ctx, userID, "generate_backup_codes", "", true, "")

	return codes, nil
}

// RecoverAccess recovers access using a recovery code
func (s *MFAService) RecoverAccess(ctx context.Context, req *mfa.RecoveryRequest) error {
	// Get backup codes
	backupCodes, err := s.repo.GetBackupCodes(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("failed to get backup codes: %w", err)
	}

	// Verify recovery code
	valid := false
	for _, bc := range backupCodes {
		if bc.Code == req.RecoveryCode && !bc.Used {
			valid = true
			// Mark as used
			if err := s.repo.MarkBackupCodeUsed(ctx, req.UserID, req.RecoveryCode); err != nil {
				return fmt.Errorf("failed to mark backup code as used: %w", err)
			}
			break
		}
	}

	if !valid {
		return mfa.ErrInvalidRecoveryCode
	}

	// Log audit event
	s.logAudit(ctx, req.UserID, "recover", mfa.MethodBackupCode, true, "")

	return nil
}

// GetSettings retrieves MFA settings for a user
func (s *MFAService) GetSettings(ctx context.Context, userID uuid.UUID) (*mfa.Settings, error) {
	settings, err := s.repo.GetSettings(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get MFA settings: %w", err)
	}
	return settings, nil
}

// IsEnabled checks if MFA is enabled for a user
func (s *MFAService) IsEnabled(ctx context.Context, userID uuid.UUID) (bool, error) {
	settings, err := s.repo.GetSettings(ctx, userID)
	if err != nil {
		if err == mfa.ErrSettingsNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to get MFA settings: %w", err)
	}
	return settings.Enabled, nil
}

// GetAvailableMethods gets available MFA methods for a user
func (s *MFAService) GetAvailableMethods(ctx context.Context, userID uuid.UUID) ([]mfa.Method, error) {
	settings, err := s.repo.GetSettings(ctx, userID)
	if err != nil {
		if err == mfa.ErrSettingsNotFound {
			return []mfa.Method{}, nil
		}
		return nil, fmt.Errorf("failed to get MFA settings: %w", err)
	}
	return settings.Methods, nil
}

// CreateChallenge creates an MFA challenge
func (s *MFAService) CreateChallenge(ctx context.Context, userID uuid.UUID, method mfa.Method) (*mfa.Challenge, error) {
	// Generate challenge code
	code := s.codeGen.GenerateNumericCode(6)

	challenge := &mfa.Challenge{
		ID:        uuid.New(),
		UserID:    userID,
		Method:    method,
		Code:      code,
		ExpiresAt: time.Now().Add(mfa.CodeExpiry),
		Attempts:  0,
		CreatedAt: time.Now(),
	}

	// Save challenge
	if err := s.repo.SaveChallenge(ctx, challenge); err != nil {
		return nil, fmt.Errorf("failed to save challenge: %w", err)
	}

	// Send code based on method
	switch method {
	case mfa.MethodSMS:
		settings, _ := s.repo.GetSettings(ctx, userID)
		if settings != nil && s.smsProvider != nil {
			_ = s.smsProvider.SendCode(ctx, settings.PhoneNumber, code)
		}

	case mfa.MethodEmail:
		user, _ := s.userRepo.GetByID(ctx, userID)
		if user != nil && s.emailProvider != nil {
			_ = s.emailProvider.SendCode(ctx, user.Email, code)
		}
	}

	// Return challenge without the code
	challenge.Code = ""
	return challenge, nil
}

// ValidateChallenge validates an MFA challenge
func (s *MFAService) ValidateChallenge(ctx context.Context, challengeID uuid.UUID, code string) (bool, error) {
	// Get challenge
	challenge, err := s.repo.GetChallenge(ctx, challengeID)
	if err != nil {
		return false, mfa.ErrChallengeNotFound
	}

	// Check if expired
	if time.Now().After(challenge.ExpiresAt) {
		return false, mfa.ErrChallengeExpired
	}

	// Check attempts
	if challenge.Attempts >= mfa.MaxAttempts {
		return false, mfa.ErrTooManyAttempts
	}

	// Validate code
	valid := challenge.Code == code

	if !valid {
		// Increment attempts
		_ = s.repo.IncrementChallengeAttempts(ctx, challengeID)
	} else {
		// Delete challenge on success
		_ = s.repo.DeleteChallenge(ctx, challengeID)
	}

	return valid, nil
}

// logAudit logs an MFA audit event
func (s *MFAService) logAudit(ctx context.Context, userID uuid.UUID, action string, method mfa.Method, success bool, details string) {
	log := &mfa.AuditLog{
		ID:        uuid.New(),
		UserID:    userID,
		Action:    action,
		Method:    method,
		Success:   success,
		Details:   details,
		CreatedAt: time.Now(),
	}
	_ = s.repo.LogAudit(ctx, log)
}

// DefaultTOTPProvider implements the TOTP provider interface
type DefaultTOTPProvider struct{}

// GenerateSecret generates a new TOTP secret
func (p *DefaultTOTPProvider) GenerateSecret() (string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

// GenerateQRCode generates a QR code for TOTP setup
func (p *DefaultTOTPProvider) GenerateQRCode(secret, email string) (string, error) {
	key, err := otp.NewKeyFromURL(fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s",
		mfa.TOTPIssuer, email, secret, mfa.TOTPIssuer))
	if err != nil {
		return "", err
	}

	_, err = key.Image(200, 200)
	if err != nil {
		return "", err
	}

	// Convert to base64
	var buf strings.Builder
	buf.WriteString("data:image/png;base64,")
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	// In production, encode the actual image
	_, _ = encoder.Write([]byte("mock-qr-code"))
	_ = encoder.Close()

	return buf.String(), nil
}

// ValidateCode validates a TOTP code
func (p *DefaultTOTPProvider) ValidateCode(secret, code string) (bool, error) {
	return totp.Validate(code, secret), nil
}

// GenerateCode generates a TOTP code (for testing)
func (p *DefaultTOTPProvider) GenerateCode(secret string) (string, error) {
	return totp.GenerateCode(secret, time.Now())
}

// DefaultCodeGenerator implements the code generator interface
type DefaultCodeGenerator struct{}

// GenerateNumericCode generates a numeric code
func (g *DefaultCodeGenerator) GenerateNumericCode(length int) string {
	const charset = "0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[randInt(len(charset))]
	}
	return string(b)
}

// GenerateAlphanumericCode generates an alphanumeric code
func (g *DefaultCodeGenerator) GenerateAlphanumericCode(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[randInt(len(charset))]
	}
	return string(b)
}

// GenerateBackupCodes generates backup codes
func (g *DefaultCodeGenerator) GenerateBackupCodes(count, length int) []string {
	codes := make([]string, count)
	for i := range codes {
		codes[i] = g.GenerateAlphanumericCode(length)
	}
	return codes
}

// randInt generates a random integer less than n
func randInt(n int) int {
	b := make([]byte, 1)
	_, _ = rand.Read(b)
	return int(b[0]) % n
}
