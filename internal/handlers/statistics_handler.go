package handlers

import (
	"constructor-script-backend/internal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetStatistics(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var stats struct {
			TotalPosts      int64 `json:"total_posts"`
			PublishedPosts  int64 `json:"published_posts"`
			TotalUsers      int64 `json:"total_users"`
			TotalCategories int64 `json:"total_categories"`
			TotalComments   int64 `json:"total_comments"`
			TotalTags       int64 `json:"total_tags"`
			TotalViews      int64 `json:"total_views"`
		}

		db.Model(&models.Post{}).Count(&stats.TotalPosts)
		db.Model(&models.Post{}).Where("published = ?", true).Count(&stats.PublishedPosts)
		db.Model(&models.User{}).Count(&stats.TotalUsers)
		db.Model(&models.Category{}).Count(&stats.TotalCategories)
		db.Model(&models.Comment{}).Count(&stats.TotalComments)
		db.Model(&models.Tag{}).Count(&stats.TotalTags)
		db.Model(&models.Post{}).Select("COALESCE(SUM(views), 0)").Row().Scan(&stats.TotalViews)

		// Additional statistics
		var popularPosts []struct {
			ID    uint   `json:"id"`
			Title string `json:"title"`
			Views int    `json:"views"`
		}

		db.Model(&models.Post{}).
			Select("id, title, views").
			Where("published = ?", true).
			Order("views DESC").
			Limit(5).
			Scan(&popularPosts)

		type monthlyCount struct {
			Period time.Time
			Count  int64
		}

		now := time.Now().UTC()
		currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		windowStart := currentMonthStart.AddDate(0, -11, 0)

		var postMonthly []monthlyCount
		db.Model(&models.Post{}).
			Select("DATE_TRUNC('month', created_at) AS period, COUNT(*) AS count").
			Where("created_at >= ?", windowStart).
			Group("period").
			Order("period").
			Scan(&postMonthly)

		var commentMonthly []monthlyCount
		db.Model(&models.Comment{}).
			Select("DATE_TRUNC('month', created_at) AS period, COUNT(*) AS count").
			Where("created_at >= ?", windowStart).
			Group("period").
			Order("period").
			Scan(&commentMonthly)

		postCounts := make(map[string]int64, len(postMonthly))
		for _, bucket := range postMonthly {
			key := bucket.Period.UTC().Format("2006-01")
			postCounts[key] = bucket.Count
		}

		commentCounts := make(map[string]int64, len(commentMonthly))
		for _, bucket := range commentMonthly {
			key := bucket.Period.UTC().Format("2006-01")
			commentCounts[key] = bucket.Count
		}

		activityTrend := make([]gin.H, 0, 12)
		for month := 0; month < 12; month++ {
			point := windowStart.AddDate(0, month, 0)
			key := point.Format("2006-01")
			activityTrend = append(activityTrend, gin.H{
				"period":   point.Format(time.RFC3339),
				"posts":    postCounts[key],
				"comments": commentCounts[key],
			})
		}

		var recentUsers []struct {
			ID       uint   `json:"id"`
			Username string `json:"username"`
			Email    string `json:"email"`
			Role     string `json:"role"`
		}

		db.Model(&models.User{}).
			Select("id, username, email, role").
			Order("created_at DESC").
			Limit(5).
			Scan(&recentUsers)

		c.JSON(http.StatusOK, gin.H{
			"statistics":     stats,
			"popular_posts":  popularPosts,
			"recent_users":   recentUsers,
			"activity_trend": activityTrend,
		})
	}
}
