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
		now := time.Now().UTC()

		var stats struct {
			TotalPosts          int64 `json:"total_posts"`
			PublishedPosts      int64 `json:"published_posts"`
			TotalUsers          int64 `json:"total_users"`
			TotalCategories     int64 `json:"total_categories"`
			TotalComments       int64 `json:"total_comments"`
			TotalTags           int64 `json:"total_tags"`
			TotalViews          int64 `json:"total_views"`
			PostsLast24Hours    int64 `json:"posts_last_24_hours"`
			PostsLast7Days      int64 `json:"posts_last_7_days"`
			CommentsLast24Hours int64 `json:"comments_last_24_hours"`
			CommentsLast7Days   int64 `json:"comments_last_7_days"`
			UsersLast7Days      int64 `json:"users_last_7_days"`
		}

		db.Model(&models.Post{}).Count(&stats.TotalPosts)
		db.Model(&models.Post{}).Where("published = ?", true).Count(&stats.PublishedPosts)
		db.Model(&models.User{}).Count(&stats.TotalUsers)
		db.Model(&models.Category{}).Count(&stats.TotalCategories)
		db.Model(&models.Comment{}).Count(&stats.TotalComments)
		db.Model(&models.Tag{}).Count(&stats.TotalTags)
		db.Model(&models.Post{}).Select("COALESCE(SUM(views), 0)").Row().Scan(&stats.TotalViews)

		twentyFourHoursAgo := now.Add(-24 * time.Hour)
		sevenDaysAgo := now.AddDate(0, 0, -7)

		db.Model(&models.Post{}).
			Where("created_at >= ?", twentyFourHoursAgo).
			Count(&stats.PostsLast24Hours)
		db.Model(&models.Post{}).
			Where("created_at >= ?", sevenDaysAgo).
			Count(&stats.PostsLast7Days)
		db.Model(&models.Comment{}).
			Where("created_at >= ?", twentyFourHoursAgo).
			Count(&stats.CommentsLast24Hours)
		db.Model(&models.Comment{}).
			Where("created_at >= ?", sevenDaysAgo).
			Count(&stats.CommentsLast7Days)
		db.Model(&models.User{}).
			Where("created_at >= ?", sevenDaysAgo).
			Count(&stats.UsersLast7Days)

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

		type timeBucket struct {
			Period time.Time
			Count  int64
		}

		startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		windowStart := startOfToday.AddDate(0, 0, -29)

		var postBuckets []timeBucket
		db.Model(&models.Post{}).
			Select("DATE_TRUNC('day', created_at) AS period, COUNT(*) AS count").
			Where("created_at >= ?", windowStart).
			Group("period").
			Order("period").
			Scan(&postBuckets)

		var commentBuckets []timeBucket
		db.Model(&models.Comment{}).
			Select("DATE_TRUNC('day', created_at) AS period, COUNT(*) AS count").
			Where("created_at >= ?", windowStart).
			Group("period").
			Order("period").
			Scan(&commentBuckets)

		postCounts := make(map[string]int64, len(postBuckets))
		for _, bucket := range postBuckets {
			key := bucket.Period.UTC().Format("2006-01-02")
			postCounts[key] = bucket.Count
		}

		commentCounts := make(map[string]int64, len(commentBuckets))
		for _, bucket := range commentBuckets {
			key := bucket.Period.UTC().Format("2006-01-02")
			commentCounts[key] = bucket.Count
		}

		activityTrend := make([]gin.H, 0, 30)
		for day := 0; day < 30; day++ {
			point := windowStart.AddDate(0, 0, day)
			key := point.Format("2006-01-02")
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
