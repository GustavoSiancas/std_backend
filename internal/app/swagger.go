package app

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files/v2"
)

//go:embed openapi.json
var openAPISpec []byte

const swaggerHTML = `<!doctype html>
<html lang="es"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>STD — Swagger</title><link rel="stylesheet" href="./swagger-ui.css"></head>
<body><div id="swagger-ui"></div><script src="./swagger-ui-bundle.js"></script>
<script>window.onload = function () { SwaggerUIBundle({url:"./openapi.json",dom_id:"#swagger-ui",deepLinking:true,validatorUrl:null}); };</script>
</body></html>`

func registerSwagger(r *gin.Engine) {
	r.GET("/swagger", func(c *gin.Context) { c.Redirect(http.StatusTemporaryRedirect, "/swagger/") })
	assets := http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerFiles.FS)))
	r.GET("/swagger/*path", func(c *gin.Context) {
		switch c.Param("path") {
		case "/", "/index.html":
			c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
		case "/openapi.json":
			c.Data(http.StatusOK, "application/json; charset=utf-8", openAPISpec)
		default:
			assets.ServeHTTP(c.Writer, c.Request)
		}
	})
}
