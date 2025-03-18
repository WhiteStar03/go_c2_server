package controllers

import (
	"awesomeProject/config"
	"awesomeProject/models"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

var jwtSecret = []byte("70085b38f0924091929ac5b3b273d1809158dce4ff0fee07b2afab427f92d888fbbc05334f1ad33c70bf0d44f112f601f3730311106fccd0bf80cde1b1decf4b498adcc21f7e2e3abbc960054d9dc0992f0a70db552cbba96864d2d0ccc3d208e5d48950782e79556f2c7f6165c24ab0a5293b8af7f2d5ac8afbdf0cd02a9f8ca4e5809457c54a710858a757afd496da692a2985506b5bc772fef13200fa9f14178442dff9b46509a0d9f80d36fde400ebc2e69d4de4adb6ed0bc37c1cb580e3c3baff810b07f1ed9a5e3ef7258c83258362ff1a1c3faf9a994946708d20104b4f8588a459a69fd4158658c86f4aa0e59f41eb1bc0ded2be693767f50364d9e9")

func Register(c *gin.Context) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	user := models.User{Username: input.Username}
	if err := user.SetPassword(input.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	result := config.DB.Create(&user)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}

func Login(c *gin.Context) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	var user models.User
	result := config.DB.Where("username = ?", input.Username).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !user.CheckPassword(input.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}
