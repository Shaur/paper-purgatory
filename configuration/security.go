package configuration

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"net/http"
	"paper/purgatory/model"
)

func AuthMiddleware(signingKey string, database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		value := c.GetHeader("Authorization")
		headerPrefixLen := len("Bearer ")
		if len(value) < headerPrefixLen {
			Unauthorized(c)
			return
		}

		token, err := jwt.Parse(value[headerPrefixLen:], func(token *jwt.Token) (interface{}, error) {
			return []byte(signingKey), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

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

func Unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
	c.Abort()
}
