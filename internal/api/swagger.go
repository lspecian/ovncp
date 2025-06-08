package api

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed swagger-ui/*
var swaggerUI embed.FS

// SwaggerUIHTML is the HTML template for Swagger UI
const SwaggerUIHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>OVN Control Platform API Documentation</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.10.0/swagger-ui.css">
    <style>
        html {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }
        *, *:before, *:after {
            box-sizing: inherit;
        }
        body {
            margin: 0;
            background: #fafafa;
        }
        .swagger-ui .topbar {
            display: none;
        }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.10.0/swagger-ui-bundle.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.10.0/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: "{{.SpecURL}}",
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
                validatorUrl: null,
                persistAuthorization: true,
                tryItOutEnabled: true,
                filter: true,
                docExpansion: "list",
                defaultModelsExpandDepth: 1,
                defaultModelExpandDepth: 1,
                displayRequestDuration: true,
                showExtensions: true,
                showCommonExtensions: true,
                onComplete: function() {
                    // Custom initialization if needed
                }
            });
            window.ui = ui;
        };
    </script>
</body>
</html>
`

// SetupSwaggerRoutes adds Swagger UI routes to the router
func (r *Router) SetupSwaggerRoutes() {
	// Serve OpenAPI spec
	r.engine.Static("/api/openapi", "./api")
	
	// Swagger UI endpoint
	r.engine.GET("/api/docs", r.swaggerUI)
	r.engine.GET("/api/docs/", r.swaggerUI)
	
	// Redirect /docs to /api/docs
	r.engine.GET("/docs", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/api/docs")
	})
}

// swaggerUI serves the Swagger UI interface
func (r *Router) swaggerUI(c *gin.Context) {
	tmpl, err := template.New("swagger").Parse(SwaggerUIHTML)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load documentation UI",
		})
		return
	}
	
	data := struct {
		SpecURL string
	}{
		SpecURL: "/api/openapi/openapi.yaml",
	}
	
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to render documentation UI",
		})
	}
}

// ReDocHTML is the HTML template for ReDoc
const ReDocHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>OVN Control Platform API Reference</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body {
            margin: 0;
            padding: 0;
        }
    </style>
</head>
<body>
    <redoc spec-url='{{.SpecURL}}'></redoc>
    <script src="https://cdn.jsdelivr.net/npm/redoc@2.0.0/bundles/redoc.standalone.js"></script>
</body>
</html>
`

// SetupReDocRoutes adds ReDoc routes to the router
func (r *Router) SetupReDocRoutes() {
	// ReDoc endpoint
	r.engine.GET("/api/redoc", r.reDoc)
	r.engine.GET("/api/reference", r.reDoc)
}

// reDoc serves the ReDoc interface
func (r *Router) reDoc(c *gin.Context) {
	tmpl, err := template.New("redoc").Parse(ReDocHTML)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load API reference",
		})
		return
	}
	
	data := struct {
		SpecURL string
	}{
		SpecURL: "/api/openapi/openapi.yaml",
	}
	
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to render API reference",
		})
	}
}