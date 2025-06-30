package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// mudar para vir de variavel de ambiente
var secretKey = []byte("minha-chave-supersecreta")

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token ausente"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return secretKey, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
			return
		}

		// Extrair claims do token
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if userID, exists := claims["sub"]; exists {
				// Converter para string e depois para int
				userIDStr := fmt.Sprintf("%v", userID)
				userIDInt, err := strconv.Atoi(userIDStr)
				if err == nil {
					c.Set("userID", userIDInt)
				}
			}
		}

		c.Next()
	}
}
