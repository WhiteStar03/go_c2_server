package config

import (
	"awesomeProject/models"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// JWT secret for signing and verifying tokens - keeping it secure
var JWTSecret = []byte("70085b38f0924091929ac5b3b273d1809158dce4ff0fee07b2afab427f92d888fbbc05334f1ad33c70bf0d44f112f601f3730311106fccd0bf80cde1b1decf4b498adcc21f7e2e3abbc960054d9dc0992f0a70db552cbba96864d2d0ccc3d208e5d48950782e79556f2c7f6165c24ab0a5293b8af7f2d5ac8afbdf0cd02a9f8ca4e5809457c54a710858a757afd496da692a2985506b5bc772fef13200fa9f14178442dff9b46509a0d9f80d36fde400ebc2e69d4de4adb6ed0bc37c1cb580e3c3baff810b07f1ed9a5e3ef7258c83258362ff1a1c3faf9a994946708d20104b4f8588a459a69fd4158658c86f4aa0e59f41eb1bc0ded2be693767f50364d9e9")

func ConnectDatabase() {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", host, user, pass, name, port)
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	DB = database
	if err := DB.AutoMigrate(
		&models.User{},
		&models.Implant{},
		&models.Command{},
		&models.ScreenshotInfo{},
	); err != nil {
		panic("failed to migrate database: " + err.Error())
	}
	fmt.Println("Database connected")
}
