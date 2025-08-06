package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"math"
	"strings"

	"golang.org/x/crypto/argon2"
)

type PasswordPolicy struct {
	MinLength           int
	MaxLength           int
	RequireUppercase    int
	RequireLowercase    int
	RequireNumbers      int
	RequireSpecialChars int
	ProhibitCommonWords bool
	ProhibitUserInfo    bool
	ProhibitSequential  bool
	ProhibitRepeating   int
	ExpiryDays          int
	HistoryCount        int
	MinAgeDays          int
	ComplexityScore     int
	MinEntropy          float64
}

func NewDefaultPasswordPolicy() *PasswordPolicy {
	return &PasswordPolicy{
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
		ExpiryDays:          90,
		HistoryCount:        5,
		MinAgeDays:          1,
		ComplexityScore:     3,
		MinEntropy:          50.0,
	}
}

type ValidationResult struct {
	IsValid     bool
	Errors      []string
	Suggestions []string
	Score       *PasswordScore
	Entropy     float64
}

type PasswordScore struct {
	Total      int
	Length     int
	Complexity int
	Uniqueness int
}

type PasswordValidator struct {
	policy           *PasswordPolicy
	entropyCalc      *EntropyCalculator
	strengthChecker  *PasswordStrengthChecker
	breachChecker    *BreachChecker
}

func NewPasswordValidator(policy *PasswordPolicy) *PasswordValidator {
	return &PasswordValidator{
		policy:          policy,
		entropyCalc:     NewEntropyCalculator(),
		strengthChecker: NewPasswordStrengthChecker(),
		breachChecker:   NewBreachChecker(),
	}
}

func (v *PasswordValidator) Validate(password, username, email string) (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid:     true,
		Errors:      []string{},
		Suggestions: []string{},
		Score:       &PasswordScore{},
	}

	if v.policy.MinLength > 0 && len(password) < v.policy.MinLength {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("password must be at least %d characters long", v.policy.MinLength))
		return result, fmt.Errorf("password must be at least %d characters long", v.policy.MinLength)
	}

	if v.policy.MaxLength > 0 && len(password) > v.policy.MaxLength {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("password must not exceed %d characters", v.policy.MaxLength))
		return result, fmt.Errorf("password must not exceed %d characters", v.policy.MaxLength)
	}

	if v.policy.RequireUppercase > 0 {
		count := countUppercase(password)
		if count < v.policy.RequireUppercase {
			result.IsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("password must contain at least %d uppercase letters", v.policy.RequireUppercase))
		}
	}

	if v.policy.RequireLowercase > 0 {
		count := countLowercase(password)
		if count < v.policy.RequireLowercase {
			result.IsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("password must contain at least %d lowercase letters", v.policy.RequireLowercase))
		}
	}

	if v.policy.RequireNumbers > 0 {
		count := countNumbers(password)
		if count < v.policy.RequireNumbers {
			result.IsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("password must contain at least %d numbers", v.policy.RequireNumbers))
		}
	}

	if v.policy.RequireSpecialChars > 0 {
		count := countSpecialChars(password)
		if count < v.policy.RequireSpecialChars {
			result.IsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("password must contain at least %d special characters", v.policy.RequireSpecialChars))
		}
	}

	if v.policy.ProhibitSequential && hasSequentialChars(password) {
		result.IsValid = false
		result.Errors = append(result.Errors, "password contains sequential characters")
	}

	if v.policy.ProhibitRepeating > 0 && hasRepeatingChars(password, v.policy.ProhibitRepeating) {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("password contains more than %d repeating characters", v.policy.ProhibitRepeating))
	}

	if v.policy.ProhibitUserInfo {
		lowerPassword := strings.ToLower(password)
		if username != "" && strings.Contains(lowerPassword, strings.ToLower(username)) {
			result.IsValid = false
			result.Errors = append(result.Errors, "password cannot contain username or email")
		}
		if email != "" {
			emailPrefix := strings.Split(email, "@")[0]
			if strings.Contains(lowerPassword, strings.ToLower(emailPrefix)) {
				result.IsValid = false
				result.Errors = append(result.Errors, "password cannot contain username or email")
			}
		}
	}

	if v.policy.ProhibitCommonWords && v.breachChecker.IsBreached(password) {
		result.IsValid = false
		result.Errors = append(result.Errors, "password is too common")
	}

	result.Entropy = v.entropyCalc.Calculate(password)
	if v.policy.MinEntropy > 0 && result.Entropy < v.policy.MinEntropy {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("password entropy too low (%.1f < %.1f)", result.Entropy, v.policy.MinEntropy))
	}

	result.Score.Total = v.strengthChecker.GetScore(password)
	if v.policy.ComplexityScore > 0 && result.Score.Total < v.policy.ComplexityScore {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("password complexity score too low (%d < %d)", result.Score.Total, v.policy.ComplexityScore))
	}

	if result.IsValid {
		result.Suggestions = append(result.Suggestions, "Password meets all requirements")
	} else {
		if len(password) < 16 {
			result.Suggestions = append(result.Suggestions, "Consider using a longer password for better security")
		}
		if result.Score.Total < 4 {
			result.Suggestions = append(result.Suggestions, "Add more variety in characters for stronger password")
		}
	}

	if len(result.Errors) > 0 {
		return result, nil
	}

	return result, nil
}

func countUppercase(s string) int {
	count := 0
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			count++
		}
	}
	return count
}

func countLowercase(s string) int {
	count := 0
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			count++
		}
	}
	return count
}

func countNumbers(s string) int {
	count := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			count++
		}
	}
	return count
}

func countSpecialChars(s string) int {
	count := 0
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			count++
		}
	}
	return count
}

func hasSequentialChars(s string) bool {
	s = strings.ToLower(s)
	sequences := []string{
		"abc", "bcd", "cde", "def", "efg", "fgh", "ghi", "hij", "ijk", "jkl",
		"klm", "lmn", "mno", "nop", "opq", "pqr", "qrs", "rst", "stu", "tuv",
		"uvw", "vwx", "wxy", "xyz", "012", "123", "234", "345", "456", "567",
		"678", "789", "890", "1234", "2345", "3456", "4567", "5678", "6789",
		"abcd", "bcde", "cdef", "defg", "efgh", "fghi", "ghij", "hijk", "ijkl",
		"jklm", "klmn", "lmno", "mnop", "nopq", "opqr", "pqrs", "qrst", "rstu",
		"stuv", "tuvw", "uvwx", "vwxy", "wxyz", "12345", "23456", "34567",
		"45678", "56789", "123456", "234567", "345678", "456789",
	}
	
	for _, seq := range sequences {
		if strings.Contains(s, seq) {
			return true
		}
	}
	return false
}

func hasRepeatingChars(s string, maxRepeat int) bool {
	if maxRepeat <= 0 {
		return false
	}
	
	for i := 0; i < len(s); i++ {
		count := 1
		for j := i + 1; j < len(s) && s[j] == s[i]; j++ {
			count++
			if count > maxRepeat {
				return true
			}
		}
	}
	return false
}

type EntropyCalculator struct{}

func NewEntropyCalculator() *EntropyCalculator {
	return &EntropyCalculator{}
}

func (e *EntropyCalculator) Calculate(password string) float64 {
	if len(password) == 0 {
		return 0
	}
	
	charSet := 0
	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSpecial := false
	
	for _, r := range password {
		if r >= 'a' && r <= 'z' && !hasLower {
			charSet += 26
			hasLower = true
		}
		if r >= 'A' && r <= 'Z' && !hasUpper {
			charSet += 26
			hasUpper = true
		}
		if r >= '0' && r <= '9' && !hasDigit {
			charSet += 10
			hasDigit = true
		}
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) && !hasSpecial {
			charSet += 32
			hasSpecial = true
		}
	}
	
	if charSet == 0 {
		return 0
	}
	
	return float64(len(password)) * math.Log2(float64(charSet))
}

type PasswordStrengthChecker struct{}

func NewPasswordStrengthChecker() *PasswordStrengthChecker {
	return &PasswordStrengthChecker{}
}

func (p *PasswordStrengthChecker) GetScore(password string) int {
	score := 0
	
	if len(password) >= 8 {
		score++
	}
	if len(password) >= 12 {
		score++
	}
	
	if countLowercase(password) > 0 && countUppercase(password) > 0 {
		score++
	}
	
	if countNumbers(password) > 0 && countSpecialChars(password) > 0 {
		score++
	}
	
	if score > 4 {
		score = 4
	}
	
	return score
}

type BreachChecker struct {
	commonPasswords map[string]bool
}

func NewBreachChecker() *BreachChecker {
	commonPasswords := map[string]bool{
		"password":     true,
		"password1":    true,
		"password123":  true,
		"123456":       true,
		"12345678":     true,
		"qwerty":       true,
		"qwerty123":    true,
		"admin":        true,
		"letmein":      true,
		"welcome":      true,
		"monkey":       true,
		"dragon":       true,
		"master":       true,
		"1234567890":   true,
	}
	
	return &BreachChecker{
		commonPasswords: commonPasswords,
	}
}

func (b *BreachChecker) IsBreached(password string) bool {
	lowerPassword := strings.ToLower(password)
	
	if b.commonPasswords[lowerPassword] {
		return true
	}
	
	for common := range b.commonPasswords {
		if strings.Contains(lowerPassword, common) {
			return true
		}
	}
	
	return false
}

type Argon2Hasher struct {
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

func NewArgon2Hasher() *Argon2Hasher {
	return &Argon2Hasher{
		time:    3,
		memory:  64 * 1024,
		threads: 4,
		keyLen:  32,
	}
}

func (h *Argon2Hasher) Hash(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	
	hash := argon2.IDKey([]byte(password), salt, h.time, h.memory, h.threads, h.keyLen)
	
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.memory, h.time, h.threads, b64Salt, b64Hash)
	
	return encoded, nil
}

func (h *Argon2Hasher) Verify(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("invalid hash format")
	}
	
	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false, err
	}
	
	var memory, time uint32
	var threads uint8
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false, err
	}
	
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	
	hash2 := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(hash)))
	
	return subtle.ConstantTimeCompare(hash, hash2) == 1, nil
}

type PasswordHistory struct {
	maxCount int
}

func NewPasswordHistory(maxCount int) *PasswordHistory {
	return &PasswordHistory{
		maxCount: maxCount,
	}
}

func (ph *PasswordHistory) IsPasswordUsed(password string, previousHashes []string) (bool, error) {
	hasher := NewArgon2Hasher()
	
	for _, hash := range previousHashes {
		match, err := hasher.Verify(password, hash)
		if err != nil {
			continue
		}
		if match {
			return true, nil
		}
	}
	
	return false, nil
}