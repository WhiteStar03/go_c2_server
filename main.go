package main

import (
	"awesomeProject/config"
	"awesomeProject/database"
	"awesomeProject/routes"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"time"
)

const (
	OfflineThreshold      = 20 * time.Second
	StatusMonitorInterval = 7 * time.Second
)

func main() {
	config.ConnectDatabase()
	go monitorImplantStatuses()

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

func monitorImplantStatuses() {
	ticker := time.NewTicker(StatusMonitorInterval)
	defer ticker.Stop()

	fmt.Println("Implant status monitor started.")
	for {
		select {
		case <-ticker.C:
			rowsAffected, err := database.UpdateStatusForInactiveImplants(OfflineThreshold)
			if err != nil {
				fmt.Printf("Error in status monitor - updating inactive implant statuses: %v\n", err)
			}
			if rowsAffected > 0 {
				fmt.Printf("Status Monitor: Marked %d implant(s) as offline.\n", rowsAffected)
			}
		}
	}
}
