package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// DocsHandler handles API documentation endpoints
type DocsHandler struct{}

// NewDocsHandler creates a new documentation handler
func NewDocsHandler() *DocsHandler {
	return &DocsHandler{}
}

// SwaggerSpec represents the OpenAPI specification structure
type SwaggerSpec struct {
	OpenAPI    string                 `json:"openapi"`
	Info       SwaggerInfo            `json:"info"`
	Paths      map[string]interface{} `json:"paths"`
	Components SwaggerComponents      `json:"components"`
}

// SwaggerInfo represents the API information
type SwaggerInfo struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// SwaggerComponents represents the reusable components
type SwaggerComponents struct {
	SecuritySchemes map[string]interface{} `json:"securitySchemes"`
	Schemas         map[string]interface{} `json:"schemas"`
}

// GetSwaggerJSON returns the OpenAPI specification in JSON format
func (h *DocsHandler) GetSwaggerJSON(c *gin.Context) {
	spec := h.generateSwaggerSpec()
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, spec)
}

// GetSwaggerUI returns the Swagger UI HTML page
func (h *DocsHandler) GetSwaggerUI(c *gin.Context) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>UManager API Documentation - Swagger UI</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui.css" />
    <style>
        .swagger-ui .topbar { display: none; }
        .swagger-ui .info { margin: 20px 0; }
        .swagger-ui .info .title { color: #1976d2; }
        body { margin: 0; padding: 20px; background: #fafafa; }
        .custom-header { 
            background: #1976d2; 
            color: white; 
            padding: 20px; 
            margin: -20px -20px 20px -20px;
            text-align: center;
        }
    </style>
</head>
<body>
    <div class="custom-header">
        <h1>UManager API Documentation</h1>
        <p>Complete API reference for the User Management System - Powered by Swagger UI</p>
    </div>
    <div id="swagger-ui"></div>
    
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: '/docs/swagger.json',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                tryItOutEnabled: true,
                supportedSubmitMethods: ['get', 'post', 'put', 'delete', 'patch'],
                onComplete: function() {
                    console.log('UManager API Documentation loaded successfully');
                }
            });
        };
    </script>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// GetRedocUI returns the Redoc UI HTML page
func (h *DocsHandler) GetRedocUI(c *gin.Context) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>UManager API Documentation</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
    <style>
        body { margin: 0; padding: 0; }
        .custom-header { 
            background: #1976d2; 
            color: white; 
            padding: 20px; 
            text-align: center;
        }
    </style>
    <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</head>
<body>
    <div class="custom-header">
        <h1>UManager API Documentation</h1>
        <p>Interactive API documentation powered by Redoc</p>
    </div>
    <redoc spec-url='/docs/swagger.json'></redoc>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// GetDocsIndex returns the documentation index page
func (h *DocsHandler) GetDocsIndex(c *gin.Context) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>UManager API Documentation</title>
    <style>
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0; 
            padding: 0; 
            background: #f5f5f5;
        }
        .container { max-width: 800px; margin: 0 auto; padding: 40px 20px; }
        .header { 
            background: #1976d2; 
            color: white; 
            padding: 40px 0; 
            text-align: center;
            margin: -40px -20px 40px -20px;
        }
        .card { 
            background: white; 
            border-radius: 8px; 
            padding: 30px; 
            margin: 20px 0; 
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .btn { 
            display: inline-block; 
            padding: 12px 24px; 
            background: #1976d2; 
            color: white; 
            text-decoration: none; 
            border-radius: 4px; 
            margin: 10px 10px 10px 0;
            font-weight: 500;
        }
        .btn:hover { background: #1565c0; }
        .btn.secondary { background: #424242; }
        .btn.secondary:hover { background: #212121; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>UManager API Documentation</h1>
            <p>Choose your preferred documentation interface</p>
        </div>
        
        <div class="card">
            <h2>Swagger UI</h2>
            <p>Interactive API documentation with try-it-out functionality. Perfect for testing endpoints directly from the browser.</p>
            <a href="/docs" class="btn">Open Swagger UI</a>
        </div>
        
        <div class="card">
            <h2>ReDoc</h2>
            <p>Clean, responsive documentation with a three-panel design. Great for reading and understanding the API structure.</p>
            <a href="/docs/redoc" class="btn">Open ReDoc</a>
        </div>
        
        <div class="card">
            <h2>OpenAPI Specification</h2>
            <p>Raw OpenAPI 3.0 specification in JSON format. Use this for generating client SDKs or importing into other tools.</p>
            <a href="/docs/swagger.json" class="btn secondary">Download JSON</a>
        </div>
    </div>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// generateSwaggerSpec creates the complete OpenAPI specification
func (h *DocsHandler) generateSwaggerSpec() SwaggerSpec {
	return SwaggerSpec{
		OpenAPI: "3.0.0",
		Info: SwaggerInfo{
			Title:       "UManager API",
			Version:     "1.0.0",
			Description: "User Management System API with comprehensive authentication, authorization, billing, and audit capabilities.",
		},
		Paths: map[string]interface{}{
			"/v1/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Health check",
					"description": "Check if the API server is running and healthy",
					"tags":        []string{"System"},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Server is healthy",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"status":    map[string]interface{}{"type": "string"},
											"timestamp": map[string]interface{}{"type": "string"},
											"version":   map[string]interface{}{"type": "string"},
											"uptime":    map[string]interface{}{"type": "number"},
										},
									},
								},
							},
						},
					},
				},
			},
			"/v1/auth/register": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Register a new user",
					"description": "Create a new user account with email, username, and password",
					"tags":        []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type":     "object",
									"required": []string{"email", "username", "password", "first_name", "last_name"},
									"properties": map[string]interface{}{
										"email":      map[string]interface{}{"type": "string", "format": "email"},
										"username":   map[string]interface{}{"type": "string", "minLength": 3},
										"password":   map[string]interface{}{"type": "string", "minLength": 8},
										"first_name": map[string]interface{}{"type": "string"},
										"last_name":  map[string]interface{}{"type": "string"},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "User registered successfully",
						},
						"400": map[string]interface{}{
							"description": "Validation error",
						},
						"409": map[string]interface{}{
							"description": "Email or username already exists",
						},
						"500": map[string]interface{}{
							"description": "Internal server error",
						},
					},
				},
			},
			"/v1/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Login user",
					"description": "Authenticate user with email/username and password",
					"tags":        []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"email":    map[string]interface{}{"type": "string"},
										"username": map[string]interface{}{"type": "string"},
										"password": map[string]interface{}{"type": "string"},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Login successful",
						},
						"401": map[string]interface{}{
							"description": "Invalid credentials",
						},
					},
				},
			},
			"/v1/auth/refresh": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Refresh access token",
					"description": "Get new access token using refresh token",
					"tags":        []string{"Authentication"},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Token refreshed successfully",
						},
						"401": map[string]interface{}{
							"description": "Invalid refresh token",
						},
					},
				},
			},
			"/v1/auth/logout": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Logout user",
					"description": "Invalidate user session and tokens",
					"tags":        []string{"Authentication"},
					"security":    []map[string]interface{}{{"bearerAuth": []string{}}},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Logout successful",
						},
						"401": map[string]interface{}{
							"description": "Unauthorized",
						},
					},
				},
			},
		},
		Components: SwaggerComponents{
			SecuritySchemes: map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
			Schemas: map[string]interface{}{
				"User": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id":         map[string]interface{}{"type": "string", "format": "uuid"},
						"email":      map[string]interface{}{"type": "string", "format": "email"},
						"username":   map[string]interface{}{"type": "string"},
						"first_name": map[string]interface{}{"type": "string"},
						"last_name":  map[string]interface{}{"type": "string"},
						"created_at": map[string]interface{}{"type": "string", "format": "date-time"},
						"updated_at": map[string]interface{}{"type": "string", "format": "date-time"},
					},
				},
				"Error": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{"type": "boolean"},
						"error": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"code":    map[string]interface{}{"type": "string"},
								"message": map[string]interface{}{"type": "string"},
							},
						},
					},
				},
			},
		},
	}
}
