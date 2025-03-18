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

	// Protected routes
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/implants", controllers.GetUserImplants)  // Get all implants for the current user
		protected.POST("/send-command", controllers.SendCommand) // Send a command to a specific implant
		protected.GET("/commands", controllers.GetCommandsForImplant)
	}

	return r
}
