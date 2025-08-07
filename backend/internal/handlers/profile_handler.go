package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/victoralfred/um_sys/internal/services"
)

type ProfileHandler struct {
	userService *services.UserService
}

type ProfileUpdateRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Bio       *string `json:"bio,omitempty"`
	Locale    *string `json:"locale,omitempty"`
	Timezone  *string `json:"timezone,omitempty"`
}

type ProfileResponse struct {
	ID                uuid.UUID `json:"id"`
	Email             string    `json:"email"`
	Username          string    `json:"username"`
	FirstName         string    `json:"first_name,omitempty"`
	LastName          string    `json:"last_name,omitempty"`
	Bio               string    `json:"bio,omitempty"`
	Locale            string    `json:"locale"`
	Timezone          string    `json:"timezone"`
	ProfilePictureURL string    `json:"profile_picture_url,omitempty"`
	EmailVerified     bool      `json:"email_verified"`
	PhoneVerified     bool      `json:"phone_verified"`
	MFAEnabled        bool      `json:"mfa_enabled"`
	CreatedAt         string    `json:"created_at"`
	UpdatedAt         string    `json:"updated_at"`
}

func NewProfileHandler(userService *services.UserService) *ProfileHandler {
	return &ProfileHandler{
		userService: userService,
	}
}

// GetProfile retrieves the user's profile information
// @Summary Get user profile
// @Description Get the profile information of a specific user
// @Tags Profile
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} ProfileResponse
// @Failure 400 {object} ErrorResponse "Invalid user ID"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /profile/{id} [get]
func (h *ProfileHandler) GetProfile(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_USER_ID",
			Message: "User ID must be a valid UUID",
		})
		return
	}

	user, err := h.userService.GetByID(context.Background(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Code:    "USER_NOT_FOUND",
			Message: "The requested user does not exist",
		})
		return
	}

	response := ProfileResponse{
		ID:                user.ID,
		Email:             user.Email,
		Username:          user.Username,
		FirstName:         user.FirstName,
		LastName:          user.LastName,
		Bio:               user.Bio,
		Locale:            user.Locale,
		Timezone:          user.Timezone,
		ProfilePictureURL: user.ProfilePictureURL,
		EmailVerified:     user.EmailVerified,
		PhoneVerified:     user.PhoneVerified,
		MFAEnabled:        user.MFAEnabled,
		CreatedAt:         user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:         user.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	c.JSON(http.StatusOK, response)
}

// UpdateProfile updates the user's profile information
// @Summary Update user profile
// @Description Update the profile information of a specific user
// @Tags Profile
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param profile body ProfileUpdateRequest true "Profile update data"
// @Success 200 {object} ProfileResponse
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /profile/{id} [put]
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_USER_ID",
			Message: "User ID must be a valid UUID",
		})
		return
	}

	var req ProfileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Failed to parse request data: " + err.Error(),
		})
		return
	}

	// Get existing user
	user, err := h.userService.GetByID(context.Background(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Code:    "USER_NOT_FOUND",
			Message: "The requested user does not exist",
		})
		return
	}

	// Update fields if provided
	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.Bio != nil {
		user.Bio = *req.Bio
	}
	if req.Locale != nil {
		user.Locale = *req.Locale
	}
	if req.Timezone != nil {
		user.Timezone = *req.Timezone
	}

	// Update user
	if err := h.userService.Update(context.Background(), user); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "UPDATE_FAILED",
			Message: "An error occurred while updating the user profile",
		})
		return
	}

	// Return updated profile
	response := ProfileResponse{
		ID:                user.ID,
		Email:             user.Email,
		Username:          user.Username,
		FirstName:         user.FirstName,
		LastName:          user.LastName,
		Bio:               user.Bio,
		Locale:            user.Locale,
		Timezone:          user.Timezone,
		ProfilePictureURL: user.ProfilePictureURL,
		EmailVerified:     user.EmailVerified,
		PhoneVerified:     user.PhoneVerified,
		MFAEnabled:        user.MFAEnabled,
		CreatedAt:         user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:         user.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	c.JSON(http.StatusOK, response)
}

// UploadProfilePicture uploads a profile picture for the user
// @Summary Upload profile picture
// @Description Upload a profile picture for a specific user
// @Tags Profile
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "User ID"
// @Param file formData file true "Profile picture file"
// @Success 200 {object} map[string]string "Upload successful"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /profile/{id}/picture [post]
func (h *ProfileHandler) UploadProfilePicture(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_USER_ID",
			Message: "User ID must be a valid UUID",
		})
		return
	}

	// Check if user exists
	user, err := h.userService.GetByID(context.Background(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Code:    "USER_NOT_FOUND",
			Message: "The requested user does not exist",
		})
		return
	}

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "NO_FILE_UPLOADED",
			Message: "Please select a file to upload",
		})
		return
	}

	// Validate file size (max 5MB)
	if file.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "FILE_TOO_LARGE",
			Message: "Profile picture must be smaller than 5MB",
		})
		return
	}

	// For now, we'll just return a mock URL
	// In a real implementation, you would save the file and return the actual URL
	profilePictureURL := "https://example.com/profiles/" + userID.String() + ".jpg"

	// Update user's profile picture URL
	user.ProfilePictureURL = profilePictureURL
	if err := h.userService.Update(context.Background(), user); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "UPDATE_FAILED",
			Message: "An error occurred while saving the profile picture",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile picture uploaded successfully",
		"url":     profilePictureURL,
	})
}

// GetUserPreferences retrieves user preferences
// @Summary Get user preferences
// @Description Get the preferences and settings for a specific user
// @Tags Profile
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} map[string]interface{} "User preferences"
// @Failure 400 {object} ErrorResponse "Invalid user ID"
// @Failure 404 {object} ErrorResponse "User not found"
// @Router /profile/{id}/preferences [get]
func (h *ProfileHandler) GetUserPreferences(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_USER_ID",
			Message: "User ID must be a valid UUID",
		})
		return
	}

	user, err := h.userService.GetByID(context.Background(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Code:    "USER_NOT_FOUND",
			Message: "The requested user does not exist",
		})
		return
	}

	preferences := map[string]interface{}{
		"locale":         user.Locale,
		"timezone":       user.Timezone,
		"mfa_enabled":    user.MFAEnabled,
		"email_verified": user.EmailVerified,
		"phone_verified": user.PhoneVerified,
	}

	c.JSON(http.StatusOK, preferences)
}

// UpdateUserPreferences updates user preferences
// @Summary Update user preferences
// @Description Update preferences and settings for a specific user
// @Tags Profile
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param preferences body map[string]interface{} true "User preferences"
// @Success 200 {object} map[string]interface{} "Updated preferences"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 404 {object} ErrorResponse "User not found"
// @Router /profile/{id}/preferences [put]
func (h *ProfileHandler) UpdateUserPreferences(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_USER_ID",
			Message: "User ID must be a valid UUID",
		})
		return
	}

	user, err := h.userService.GetByID(context.Background(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Code:    "USER_NOT_FOUND",
			Message: "The requested user does not exist",
		})
		return
	}

	var preferences map[string]interface{}
	if err := c.ShouldBindJSON(&preferences); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Failed to parse preferences data",
		})
		return
	}

	// Update preferences
	if locale, ok := preferences["locale"].(string); ok {
		user.Locale = locale
	}
	if timezone, ok := preferences["timezone"].(string); ok {
		user.Timezone = timezone
	}

	if err := h.userService.Update(context.Background(), user); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "UPDATE_FAILED",
			Message: "An error occurred while updating user preferences",
		})
		return
	}

	updatedPreferences := map[string]interface{}{
		"locale":         user.Locale,
		"timezone":       user.Timezone,
		"mfa_enabled":    user.MFAEnabled,
		"email_verified": user.EmailVerified,
		"phone_verified": user.PhoneVerified,
	}

	c.JSON(http.StatusOK, updatedPreferences)
}
