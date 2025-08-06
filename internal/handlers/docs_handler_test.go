package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("GetSwaggerJSON", func(t *testing.T) {
		t.Run("should return OpenAPI 3.0 specification", func(t *testing.T) {
			// Arrange
			handler := NewDocsHandler()
			router := gin.New()
			router.GET("/docs/swagger.json", handler.GetSwaggerJSON)

			req, _ := http.NewRequest("GET", "/docs/swagger.json", nil)
			resp := httptest.NewRecorder()

			// Act
			router.ServeHTTP(resp, req)

			// Assert
			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))

			var swaggerDoc map[string]interface{}
			err := json.Unmarshal(resp.Body.Bytes(), &swaggerDoc)
			require.NoError(t, err)

			// Validate OpenAPI structure
			assert.Equal(t, "3.0.0", swaggerDoc["openapi"])
			assert.Contains(t, swaggerDoc, "info")
			assert.Contains(t, swaggerDoc, "paths")
			assert.Contains(t, swaggerDoc, "components")

			// Validate info section
			info, ok := swaggerDoc["info"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, "UManager API", info["title"])
			assert.Contains(t, info, "version")
			assert.Contains(t, info, "description")

			// Validate paths exist
			paths, ok := swaggerDoc["paths"].(map[string]interface{})
			require.True(t, ok)
			assert.Contains(t, paths, "/v1/health")
			assert.Contains(t, paths, "/v1/auth/register")
			assert.Contains(t, paths, "/v1/auth/login")
		})

		t.Run("should include security schemas", func(t *testing.T) {
			// Arrange
			handler := NewDocsHandler()
			router := gin.New()
			router.GET("/docs/swagger.json", handler.GetSwaggerJSON)

			req, _ := http.NewRequest("GET", "/docs/swagger.json", nil)
			resp := httptest.NewRecorder()

			// Act
			router.ServeHTTP(resp, req)

			// Assert
			var swaggerDoc map[string]interface{}
			err := json.Unmarshal(resp.Body.Bytes(), &swaggerDoc)
			require.NoError(t, err)

			components, ok := swaggerDoc["components"].(map[string]interface{})
			require.True(t, ok)

			securitySchemes, ok := components["securitySchemes"].(map[string]interface{})
			require.True(t, ok)
			assert.Contains(t, securitySchemes, "bearerAuth")
		})
	})

	t.Run("GetSwaggerUI", func(t *testing.T) {
		t.Run("should return HTML page with Swagger UI", func(t *testing.T) {
			// Arrange
			handler := NewDocsHandler()
			router := gin.New()
			router.GET("/docs", handler.GetSwaggerUI)

			req, _ := http.NewRequest("GET", "/docs", nil)
			resp := httptest.NewRecorder()

			// Act
			router.ServeHTTP(resp, req)

			// Assert
			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Equal(t, "text/html; charset=utf-8", resp.Header().Get("Content-Type"))

			body := resp.Body.String()
			assert.Contains(t, body, "<!DOCTYPE html>")
			assert.Contains(t, body, "Swagger UI")
			assert.Contains(t, body, "UManager API Documentation")
			assert.Contains(t, body, "swagger-ui-bundle")
			assert.Contains(t, body, "/docs/swagger.json")
		})

		t.Run("should include custom CSS for branding", func(t *testing.T) {
			// Arrange
			handler := NewDocsHandler()
			router := gin.New()
			router.GET("/docs", handler.GetSwaggerUI)

			req, _ := http.NewRequest("GET", "/docs", nil)
			resp := httptest.NewRecorder()

			// Act
			router.ServeHTTP(resp, req)

			// Assert
			body := resp.Body.String()
			assert.Contains(t, body, ".swagger-ui .topbar")
			assert.Contains(t, body, "UManager")
		})
	})

	t.Run("GetRedocUI", func(t *testing.T) {
		t.Run("should return HTML page with Redoc UI", func(t *testing.T) {
			// Arrange
			handler := NewDocsHandler()
			router := gin.New()
			router.GET("/docs/redoc", handler.GetRedocUI)

			req, _ := http.NewRequest("GET", "/docs/redoc", nil)
			resp := httptest.NewRecorder()

			// Act
			router.ServeHTTP(resp, req)

			// Assert
			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Equal(t, "text/html; charset=utf-8", resp.Header().Get("Content-Type"))

			body := resp.Body.String()
			assert.Contains(t, body, "<!DOCTYPE html>")
			assert.Contains(t, body, "Redoc")
			assert.Contains(t, body, "UManager API Documentation")
			assert.Contains(t, body, "redoc.standalone.js")
			assert.Contains(t, body, "/docs/swagger.json")
		})
	})

	t.Run("GetDocsIndex", func(t *testing.T) {
		t.Run("should return index page with documentation options", func(t *testing.T) {
			// Arrange
			handler := NewDocsHandler()
			router := gin.New()
			router.GET("/docs/", handler.GetDocsIndex)

			req, _ := http.NewRequest("GET", "/docs/", nil)
			resp := httptest.NewRecorder()

			// Act
			router.ServeHTTP(resp, req)

			// Assert
			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Equal(t, "text/html; charset=utf-8", resp.Header().Get("Content-Type"))

			body := resp.Body.String()
			assert.Contains(t, body, "UManager API Documentation")
			assert.Contains(t, body, "/docs")
			assert.Contains(t, body, "/docs/redoc")
			assert.Contains(t, body, "Swagger UI")
			assert.Contains(t, body, "ReDoc")
		})
	})
}

func TestSwaggerSpecGeneration(t *testing.T) {
	t.Run("should generate valid OpenAPI spec for auth endpoints", func(t *testing.T) {
		// Arrange
		handler := NewDocsHandler()

		// Act
		spec := handler.generateSwaggerSpec()

		// Assert
		assert.Equal(t, "3.0.0", spec.OpenAPI)
		assert.Equal(t, "UManager API", spec.Info.Title)

		// Check auth endpoints exist
		assert.Contains(t, spec.Paths, "/v1/auth/register")
		assert.Contains(t, spec.Paths, "/v1/auth/login")
		assert.Contains(t, spec.Paths, "/v1/auth/refresh")
		assert.Contains(t, spec.Paths, "/v1/auth/logout")

		// Validate register endpoint
		registerPath, ok := spec.Paths["/v1/auth/register"].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, registerPath, "post")

		postOp := registerPath["post"].(map[string]interface{})
		assert.Equal(t, "Register a new user", postOp["summary"])
		assert.Contains(t, postOp, "requestBody")
		assert.Contains(t, postOp, "responses")
	})

	t.Run("should include proper error responses", func(t *testing.T) {
		// Arrange
		handler := NewDocsHandler()

		// Act
		spec := handler.generateSwaggerSpec()

		// Assert
		registerPath, ok := spec.Paths["/v1/auth/register"].(map[string]interface{})
		require.True(t, ok)
		postOp := registerPath["post"].(map[string]interface{})
		responses := postOp["responses"].(map[string]interface{})

		// Check standard responses
		assert.Contains(t, responses, "201")
		assert.Contains(t, responses, "400")
		assert.Contains(t, responses, "409")
		assert.Contains(t, responses, "500")
	})
}
