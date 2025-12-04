package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"constructor-script-backend/internal/authorization"
	"constructor-script-backend/internal/constants"
)

// extractTokenFromHeader extracts JWT token from Authorization header
// Returns empty string if header is missing or invalid
func extractTokenFromHeader(c *gin.Context) string {
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if authHeader == "" {
		return ""
	}

	bearerToken := strings.SplitN(authHeader, " ", 2)
	if len(bearerToken) != 2 || !strings.EqualFold(bearerToken[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(bearerToken[1])
}

// extractTokenFromCookie extracts JWT token from cookie
// Returns empty string if cookie is missing or empty
func extractTokenFromCookie(c *gin.Context) string {
	cookieToken, err := c.Cookie(constants.AuthTokenCookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookieToken)
}

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header first (priority)
		tokenString := extractTokenFromHeader(c)

		// If no valid token in header, try cookie
		if tokenString == "" {
			tokenString = extractTokenFromCookie(c)
		}

		// If still no token found, return unauthorized
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization credentials required"})
			c.Abort()
			return
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

		rawRole, ok := claims["role"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token role"})
			c.Abort()
			return
		}

		role := authorization.UserRole(strings.ToLower(strings.TrimSpace(rawRole)))
		if !role.IsValid() {
			c.JSON(http.StatusForbidden, gin.H{"error": "unknown user role"})
			c.Abort()
			return
		}

		c.Set("role", role)

		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		parsedRole, ok := authorization.ParseUserRole(role)
		if !exists || !ok || parsedRole != authorization.RoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequirePermissions(perms ...authorization.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(perms) == 0 {
			c.Next()
			return
		}

		roleValue, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "role not provided"})
			c.Abort()
			return
		}

		role, ok := authorization.ParseUserRole(roleValue)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid user role"})
			c.Abort()
			return
		}

		for _, perm := range perms {
			if authorization.RoleHasPermission(role, perm) {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		c.Abort()
	}
}

func RequireRoles(roles ...authorization.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(roles) == 0 {
			c.Next()
			return
		}

		roleValue, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "role not provided"})
			c.Abort()
			return
		}

		role, ok := authorization.ParseUserRole(roleValue)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid user role"})
			c.Abort()
			return
		}

		for _, allowed := range roles {
			if role == allowed {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "role not permitted"})
		c.Abort()
	}
}
