package middleware

import (
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

var jwtSecret = []byte("70085b38f0924091929ac5b3b273d1809158dce4ff0fee07b2afab427f92d888fbbc05334f1ad33c70bf0d44f112f601f3730311106fccd0bf80cde1b1decf4b498adcc21f7e2e3abbc960054d9dc0992f0a70db552cbba96864d2d0ccc3d208e5d48950782e79556f2c7f6165c24ab0a5293b8af7f2d5ac8afbdf0cd02a9f8ca4e5809457c54a710858a757afd496da692a2985506b5bc772fef13200fa9f14178442dff9b46509a0d9f80d36fde400ebc2e69d4de4adb6ed0bc37c1cb580e3c3baff810b07f1ed9a5e3ef7258c83258362ff1a1c3faf9a994946708d20104b4f8588a459a69fd4158658c86f4aa0e59f41eb1bc0ded2be693767f50364d9e9")

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString = tokenString[7:]
		fmt.Println(tokenString)

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		userID := claims["user_id"].(float64)

		c.Set("user_id", int(userID))
		c.Next()
	}
}
