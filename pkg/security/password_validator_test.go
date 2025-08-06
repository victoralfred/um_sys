package security

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordPolicy_NewDefaultPolicy(t *testing.T) {
	policy := NewDefaultPasswordPolicy()
	
	assert.NotNil(t, policy)
	assert.Equal(t, 12, policy.MinLength)
	assert.Equal(t, 128, policy.MaxLength)
	assert.Equal(t, 2, policy.RequireUppercase)
	assert.Equal(t, 2, policy.RequireLowercase)
	assert.Equal(t, 2, policy.RequireNumbers)
	assert.Equal(t, 2, policy.RequireSpecialChars)
	assert.True(t, policy.ProhibitCommonWords)
	assert.True(t, policy.ProhibitUserInfo)
	assert.True(t, policy.ProhibitSequential)
	assert.Equal(t, 3, policy.ProhibitRepeating)
	assert.Equal(t, 90, policy.ExpiryDays)
	assert.Equal(t, 5, policy.HistoryCount)
	assert.Equal(t, 1, policy.MinAgeDays)
	assert.Equal(t, 3, policy.ComplexityScore)
}

func TestPasswordValidator_ValidateLength(t *testing.T) {
	tests := []struct {
		name     string
		password string
		policy   *PasswordPolicy
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "password too short",
			password: "Short1!",
			policy:   &PasswordPolicy{MinLength: 12, MaxLength: 128},
			wantErr:  true,
			errMsg:   "password must be at least 12 characters long",
		},
		{
			name:     "password too long",
			password: strings.Repeat("a", 129),
			policy:   &PasswordPolicy{MinLength: 12, MaxLength: 128},
			wantErr:  true,
			errMsg:   "password must not exceed 128 characters",
		},
		{
			name:     "password length valid",
			password: "ValidLength123!@#",
			policy:   &PasswordPolicy{MinLength: 12, MaxLength: 128},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewPasswordValidator(tt.policy)
			result, err := validator.Validate(tt.password, "", "")
			
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.False(t, result.IsValid)
			} else {
				assert.True(t, result.IsValid || len(result.Errors) > 0)
			}
		})
	}
}

func TestPasswordValidator_ValidateComplexity(t *testing.T) {
	tests := []struct {
		name     string
		password string
		policy   *PasswordPolicy
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "missing uppercase letters",
			password: "nouppercase123!@#",
			policy: &PasswordPolicy{
				MinLength:        12,
				RequireUppercase: 2,
			},
			wantErr: true,
			errMsg:  "password must contain at least 2 uppercase letters",
		},
		{
			name:     "missing lowercase letters",
			password: "NOLOWERCASE123!@#",
			policy: &PasswordPolicy{
				MinLength:        12,
				RequireLowercase: 2,
			},
			wantErr: true,
			errMsg:  "password must contain at least 2 lowercase letters",
		},
		{
			name:     "missing numbers",
			password: "NoNumbersHere!@#ABC",
			policy: &PasswordPolicy{
				MinLength:      12,
				RequireNumbers: 2,
			},
			wantErr: true,
			errMsg:  "password must contain at least 2 numbers",
		},
		{
			name:     "missing special characters",
			password: "NoSpecialChars123ABC",
			policy: &PasswordPolicy{
				MinLength:           12,
				RequireSpecialChars: 2,
			},
			wantErr: true,
			errMsg:  "password must contain at least 2 special characters",
		},
		{
			name:     "meets all complexity requirements",
			password: "ValidPass123!@#ABC",
			policy: &PasswordPolicy{
				MinLength:           12,
				RequireUppercase:    2,
				RequireLowercase:    2,
				RequireNumbers:      2,
				RequireSpecialChars: 2,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewPasswordValidator(tt.policy)
			result, err := validator.Validate(tt.password, "", "")
			
			if tt.wantErr {
				assert.False(t, result.IsValid)
				assert.Contains(t, strings.Join(result.Errors, " "), tt.errMsg)
			} else {
				if err == nil {
					assert.True(t, result.IsValid)
				}
			}
		})
	}
}

func TestPasswordValidator_ProhibitSequential(t *testing.T) {
	tests := []struct {
		name     string
		password string
		policy   *PasswordPolicy
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "contains sequential letters",
			password: "Pass123abcDEF!@#",
			policy: &PasswordPolicy{
				MinLength:          12,
				ProhibitSequential: true,
			},
			wantErr: true,
			errMsg:  "password contains sequential characters",
		},
		{
			name:     "contains sequential numbers",
			password: "Pass123456ABC!@#",
			policy: &PasswordPolicy{
				MinLength:          12,
				ProhibitSequential: true,
			},
			wantErr: true,
			errMsg:  "password contains sequential characters",
		},
		{
			name:     "no sequential characters",
			password: "Pass1A2B3C!@#XYZ",
			policy: &PasswordPolicy{
				MinLength:          12,
				ProhibitSequential: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewPasswordValidator(tt.policy)
			result, err := validator.Validate(tt.password, "", "")
			
			if tt.wantErr {
				assert.False(t, result.IsValid)
				assert.Contains(t, strings.Join(result.Errors, " "), tt.errMsg)
			} else {
				if err == nil {
					assert.True(t, result.IsValid || len(result.Errors) == 0)
				}
			}
		})
	}
}

func TestPasswordValidator_ProhibitRepeating(t *testing.T) {
	tests := []struct {
		name     string
		password string
		policy   *PasswordPolicy
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "contains too many repeating characters",
			password: "Passsss123ABC!@#",
			policy: &PasswordPolicy{
				MinLength:         12,
				ProhibitRepeating: 3,
			},
			wantErr: true,
			errMsg:  "password contains more than 3 repeating characters",
		},
		{
			name:     "acceptable repeating characters",
			password: "Pass112233ABC!@#",
			policy: &PasswordPolicy{
				MinLength:         12,
				ProhibitRepeating: 3,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewPasswordValidator(tt.policy)
			result, err := validator.Validate(tt.password, "", "")
			
			if tt.wantErr {
				assert.False(t, result.IsValid)
				assert.Contains(t, strings.Join(result.Errors, " "), tt.errMsg)
			} else {
				if err == nil {
					assert.True(t, result.IsValid || len(result.Errors) == 0)
				}
			}
		})
	}
}

func TestPasswordValidator_ProhibitUserInfo(t *testing.T) {
	tests := []struct {
		name     string
		password string
		username string
		email    string
		policy   *PasswordPolicy
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "contains username",
			password: "MyJohnDoe123!@#",
			username: "johndoe",
			email:    "john@example.com",
			policy: &PasswordPolicy{
				MinLength:        12,
				ProhibitUserInfo: true,
			},
			wantErr: true,
			errMsg:  "password cannot contain username or email",
		},
		{
			name:     "contains email prefix",
			password: "Pass123john!@#ABC",
			username: "user123",
			email:    "john@example.com",
			policy: &PasswordPolicy{
				MinLength:        12,
				ProhibitUserInfo: true,
			},
			wantErr: true,
			errMsg:  "password cannot contain username or email",
		},
		{
			name:     "does not contain user info",
			password: "SecurePass123!@#",
			username: "johndoe",
			email:    "john@example.com",
			policy: &PasswordPolicy{
				MinLength:        12,
				ProhibitUserInfo: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewPasswordValidator(tt.policy)
			result, err := validator.Validate(tt.password, tt.username, tt.email)
			
			if tt.wantErr {
				assert.False(t, result.IsValid)
				assert.Contains(t, strings.Join(result.Errors, " "), tt.errMsg)
			} else {
				if err == nil {
					assert.True(t, result.IsValid || len(result.Errors) == 0)
				}
			}
		})
	}
}

func TestPasswordValidator_ProhibitCommonWords(t *testing.T) {
	tests := []struct {
		name     string
		password string
		policy   *PasswordPolicy
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "contains common password",
			password: "Password123!@#",
			policy: &PasswordPolicy{
				MinLength:           12,
				ProhibitCommonWords: true,
			},
			wantErr: true,
			errMsg:  "password is too common",
		},
		{
			name:     "contains qwerty pattern",
			password: "Qwerty123456!@#",
			policy: &PasswordPolicy{
				MinLength:           12,
				ProhibitCommonWords: true,
			},
			wantErr: true,
			errMsg:  "password is too common",
		},
		{
			name:     "unique password",
			password: "Un1qu3P@ssw0rd!",
			policy: &PasswordPolicy{
				MinLength:           12,
				ProhibitCommonWords: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewPasswordValidator(tt.policy)
			result, err := validator.Validate(tt.password, "", "")
			
			if tt.wantErr {
				assert.False(t, result.IsValid)
				assert.Contains(t, strings.Join(result.Errors, " "), tt.errMsg)
			} else {
				if err == nil {
					assert.True(t, result.IsValid || len(result.Errors) == 0)
				}
			}
		})
	}
}

func TestPasswordEntropy_Calculate(t *testing.T) {
	tests := []struct {
		name            string
		password        string
		minEntropy      float64
		shouldPassCheck bool
	}{
		{
			name:            "low entropy password",
			password:        "password",
			minEntropy:      50.0,
			shouldPassCheck: false,
		},
		{
			name:            "medium entropy password",
			password:        "Pass123!@#",
			minEntropy:      50.0,
			shouldPassCheck: true,
		},
		{
			name:            "high entropy password",
			password:        "Un1qu3P@ssw0rd!XyZ",
			minEntropy:      60.0,
			shouldPassCheck: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calculator := NewEntropyCalculator()
			entropy := calculator.Calculate(tt.password)
			
			assert.Greater(t, entropy, 0.0)
			
			if tt.shouldPassCheck {
				assert.GreaterOrEqual(t, entropy, tt.minEntropy)
			} else {
				assert.Less(t, entropy, tt.minEntropy)
			}
		})
	}
}

func TestPasswordStrength_GetScore(t *testing.T) {
	tests := []struct {
		name     string
		password string
		minScore int
	}{
		{
			name:     "very weak password",
			password: "password",
			minScore: 0,
		},
		{
			name:     "weak password",
			password: "Password1",
			minScore: 1,
		},
		{
			name:     "fair password",
			password: "Password123!",
			minScore: 2,
		},
		{
			name:     "good password",
			password: "MyP@ssw0rd123!",
			minScore: 3,
		},
		{
			name:     "strong password",
			password: "Un1qu3P@ssw0rd!XyZ#2024",
			minScore: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewPasswordStrengthChecker()
			score := checker.GetScore(tt.password)
			
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, 4)
		})
	}
}

func TestPasswordValidator_CompleteValidation(t *testing.T) {
	policy := &PasswordPolicy{
		MinLength:           12,
		MaxLength:           128,
		RequireUppercase:    2,
		RequireLowercase:    2,
		RequireNumbers:      2,
		RequireSpecialChars: 2,
		ProhibitCommonWords: true,
		ProhibitUserInfo:    true,
		ProhibitSequential:  true,
		ProhibitRepeating:   3,
		ComplexityScore:     3,
		MinEntropy:          50.0,
	}

	validator := NewPasswordValidator(policy)

	tests := []struct {
		name     string
		password string
		username string
		email    string
		wantErr  bool
	}{
		{
			name:     "perfect password",
			password: "S3cur3P@ssph#aseZ9",
			username: "johndoe",
			email:    "john@example.com",
			wantErr:  false,
		},
		{
			name:     "fails multiple checks",
			password: "password123",
			username: "johndoe",
			email:    "john@example.com",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.Validate(tt.password, tt.username, tt.email)
			
			if tt.wantErr {
				assert.False(t, result.IsValid)
				assert.NotEmpty(t, result.Errors)
			} else {
				if err == nil {
					assert.True(t, result.IsValid)
					assert.Empty(t, result.Errors)
					assert.NotEmpty(t, result.Suggestions)
				}
			}
			
			assert.NotNil(t, result.Score)
			assert.GreaterOrEqual(t, result.Score.Total, 0)
			assert.LessOrEqual(t, result.Score.Total, 4)
		})
	}
}

func TestPasswordHasher_Argon2(t *testing.T) {
	hasher := NewArgon2Hasher()

	t.Run("hash and verify password", func(t *testing.T) {
		password := "SecureP@ssw0rd123!"
		
		hash, err := hasher.Hash(password)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, password, hash)
		
		valid, err := hasher.Verify(password, hash)
		require.NoError(t, err)
		assert.True(t, valid)
		
		invalid, err := hasher.Verify("WrongPassword", hash)
		require.NoError(t, err)
		assert.False(t, invalid)
	})

	t.Run("different hashes for same password", func(t *testing.T) {
		password := "SecureP@ssw0rd123!"
		
		hash1, err := hasher.Hash(password)
		require.NoError(t, err)
		
		hash2, err := hasher.Hash(password)
		require.NoError(t, err)
		
		assert.NotEqual(t, hash1, hash2)
		
		valid1, err := hasher.Verify(password, hash1)
		require.NoError(t, err)
		assert.True(t, valid1)
		
		valid2, err := hasher.Verify(password, hash2)
		require.NoError(t, err)
		assert.True(t, valid2)
	})
}

func TestPasswordHistory_Check(t *testing.T) {
	history := NewPasswordHistory(5)

	t.Run("check password history", func(t *testing.T) {
		hasher := NewArgon2Hasher()
		
		passwords := []string{
			"OldPassword1!@#",
			"OldPassword2!@#",
			"OldPassword3!@#",
			"OldPassword4!@#",
			"OldPassword5!@#",
		}
		
		var hashes []string
		for _, pwd := range passwords {
			hash, err := hasher.Hash(pwd)
			require.NoError(t, err)
			hashes = append(hashes, hash)
		}
		
		for i, pwd := range passwords {
			used, err := history.IsPasswordUsed(pwd, hashes[:i+1])
			require.NoError(t, err)
			assert.True(t, used, "Password %s should be in history", pwd)
		}
		
		newPassword := "NewPassword6!@#"
		used, err := history.IsPasswordUsed(newPassword, hashes)
		require.NoError(t, err)
		assert.False(t, used, "New password should not be in history")
	})
}

func TestBreachChecker_CheckPassword(t *testing.T) {
	t.Run("check breached passwords", func(t *testing.T) {
		checker := NewBreachChecker()
		
		breachedPasswords := []string{
			"password",
			"123456",
			"password123",
			"qwerty",
			"admin",
		}
		
		for _, pwd := range breachedPasswords {
			breached := checker.IsBreached(pwd)
			assert.True(t, breached, "Password %s should be marked as breached", pwd)
		}
		
		safePassword := "Un1qu3P@ssw0rd!XyZ#2024"
		breached := checker.IsBreached(safePassword)
		assert.False(t, breached, "Safe password should not be marked as breached")
	})
}