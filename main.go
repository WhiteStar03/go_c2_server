package main

import (
	"awesomeProject/config"
	"awesomeProject/database"
	"awesomeProject/routes"
	"fmt"
	"time"

	"github.com/gin-contrib/cors"
)

const (
	OfflineThreshold      = 20 * time.Second
	StatusMonitorInterval = 7 * time.Second
)

func main() {
	config.ConnectDatabase()
	go monitorImplantStatuses()

	// router with all routes
	r := routes.SetupRouter()

	// Apply CORS Middleware 
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	fmt.Println("Server running on port 8080")
	r.Run(":8080")
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
