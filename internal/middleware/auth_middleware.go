package middleware

import (
	"constructor-script-backend/internal/constants"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const authTokenCookieName = "auth_token"

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader != "" {
			bearerToken := strings.SplitN(authHeader, " ", 2)
			if len(bearerToken) == 2 && strings.EqualFold(bearerToken[0], "Bearer") {
				tokenString = strings.TrimSpace(bearerToken[1])
			} else {
				if cookieToken, err := c.Cookie(constants.AuthTokenCookieName); err == nil && strings.TrimSpace(cookieToken) != "" {
					tokenString = cookieToken
				} else {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
					c.Abort()
					return
				}
			}
		}

		if tokenString == "" {
			if cookieToken, err := c.Cookie(constants.AuthTokenCookieName); err == nil && strings.TrimSpace(cookieToken) != "" {
				tokenString = cookieToken
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization credentials required"})
				c.Abort()
				return
			}
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		if exp, ok := claims["exp"].(float64); ok {
			if time.Now().Unix() > int64(exp) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
				c.Abort()
				return
			}
		}

		c.Set("user_id", uint(claims["user_id"].(float64)))
		c.Set("email", claims["email"].(string))
		c.Set("username", claims["username"].(string))
		c.Set("role", claims["role"].(string))

		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}
