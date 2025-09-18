package configuration

import (
	"encoding/base64"
	"net/http"
	"paper/purgatory/model"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

var whitelistPaths = []string{
	"/actuator/health",
}

func AuthMiddleware(signingKey string, database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if slices.Contains(whitelistPaths, c.FullPath()) {
			c.Next()
			return
		}

		value := c.GetHeader("Authorization")
		headerPrefixLen := len("Bearer ")
		if len(value) < headerPrefixLen {
			Unauthorized(c)
			return
		}

		token, err := jwt.Parse(value[headerPrefixLen:], func(token *jwt.Token) (interface{}, error) {
			data, err := base64.StdEncoding.DecodeString(signingKey)
			return data, err
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS384.Alg()}))

		if err != nil {
			Unauthorized(c)
			return
		}

		subject, err := token.Claims.GetSubject()
		if err != nil || subject == "" {
			Unauthorized(c)
			return
		}

		user := model.User{}
		database.Take(&user, "username = ?", subject)
		if user.Username != subject {
			Unauthorized(c)
			return
		}

		c.Next()
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func Unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
	c.Abort()
}
