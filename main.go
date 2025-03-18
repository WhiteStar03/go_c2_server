package main

import (
	"awesomeProject/config"
	"awesomeProject/routes"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"time"
)

func main() {
	config.ConnectDatabase()

	// ✅ Initialize Gin Router (Fix: Don't overwrite the router)
	r := gin.Default()

	// ✅ Apply CORS Middleware BEFORE setting up routes
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true, // ✅ Allow all origins
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// ✅ Register routes after CORS middleware
	apiRoutes := routes.SetupRouter()
	r.Any("/*path", func(c *gin.Context) {
		apiRoutes.ServeHTTP(c.Writer, c.Request) // ✅ Ensure routes are correctly handled
	})

	// ✅ Start the server
	fmt.Println("Server running on port 8080")
	r.Run(":8080") // Listen on port 8080
}
