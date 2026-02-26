package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"github.com/gin-gonic/gin"
	"whatsapp-groupe4/api-gateway/handlers"
	"whatsapp-groupe4/api-gateway/ws"
)

func proxyHandler(targetURL string) gin.HandlerFunc {
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("Invalid target URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	return func(c *gin.Context) {
		proxy.Director = func(req *http.Request) {
			req.Header = c.Request.Header
			req.Host = target.Host
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			// req.URL.Path = c.Request.URL.Path
			req.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/users")
			req.URL.RawQuery = c.Request.URL.RawQuery
		}

		// LOGGING IS HERE: 'c' and 'target' are valid here
		log.Printf("Proxying request: %s to %s%s", c.Request.URL.Path, target.Host, c.Request.URL.Path)

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func main() {
	hub := ws.NewHub()
	r := gin.Default()

	userServiceURL := os.Getenv("USER_SERVICE_URL")
	messageServiceURL := os.Getenv("MESSAGE_SERVICE_URL")

	if userServiceURL == "" { userServiceURL = "http://user-service:8080" }
	if messageServiceURL == "" { messageServiceURL = "http://message-service:8082" }

	r.Any("/users/*path", proxyHandler(userServiceURL))
	r.Any("/messages/*path", proxyHandler(messageServiceURL))

	r.GET("/ws", handlers.WsHandler(hub))

	r.Run(":8080")
}