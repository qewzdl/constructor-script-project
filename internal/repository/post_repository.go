package repository

import (
	"time"

	"constructor-script-backend/internal/models"

	"gorm.io/gorm"
)

type PostRepository interface {
	Create(post *models.Post) error
	GetByID(id uint) (*models.Post, error)
	GetAll(offset, limit int, categoryID *uint, tagName *string, authorID *uint, published *bool) ([]models.Post, int64, error)
	Update(post *models.Post) error
	Delete(id uint) error
	GetBySlug(slug string) (*models.Post, error)
	GetPopular(limit int) ([]models.Post, error)
	GetRecent(limit int) ([]models.Post, error)
	GetRelated(postID uint, categoryID uint, limit int) ([]models.Post, error)
	IncrementViews(id uint) error
	GetViewStats(postID uint, start time.Time) ([]DailyCount, error)
	GetAverageViews() (float64, error)
	GetAverageComments() (float64, error)
	GetViewRank(postID uint) (int64, int64, error)
	GetCommentRank(postID uint) (int64, int64, error)
	ExistsBySlug(slug string) (bool, error)
	ReassignCategory(fromCategoryID, toCategoryID uint) error
	GetAllPublished() ([]models.Post, error)
}

type postRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) PostRepository {
	return &postRepository{db: db}
}

func (r *postRepository) Create(post *models.Post) error {
	return r.db.Create(post).Error
}

func (r *postRepository) GetByID(id uint) (*models.Post, error) {
	var post models.Post
	err := r.db.Preload("Author").Preload("Category").Preload("Tags").Preload("Comments").First(&post, id).Error
	return &post, err
}

func (r *postRepository) GetAll(offset, limit int, categoryID *uint, tagName *string, authorID *uint, published *bool) ([]models.Post, int64, error) {
	var posts []models.Post
	var total int64

	query := r.db.Model(&models.Post{})
	now := time.Now().UTC()

	if published != nil {
		query = query.Where("published = ?", *published)
		if *published {
			query = query.Where("publish_at IS NULL OR publish_at <= ?", now)
		}
	}

	if categoryID != nil {
		query = query.Where("category_id = ?", *categoryID)
	}

	if authorID != nil {
		query = query.Where("author_id = ?", *authorID)
	}

	if tagName != nil {
		query = query.Joins("JOIN post_tags ON post_tags.post_id = posts.id").
			Joins("JOIN tags ON tags.id = post_tags.tag_id").
			Where("tags.slug = ?", *tagName)
	}

	query.Count(&total)

	err := query.Preload("Author").Preload("Category").Preload("Tags").
		Order("COALESCE(posts.publish_at, posts.created_at) DESC").
		Offset(offset).Limit(limit).
		Find(&posts).Error

	return posts, total, err
}

func (r *postRepository) Update(post *models.Post) error {
	return r.db.Session(&gorm.Session{FullSaveAssociations: true}).Omit("Category").Save(post).Error
}

func (r *postRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM post_tags WHERE post_id = ?", id).Error; err != nil {
			return err
		}

		if err := tx.Unscoped().Where("post_id = ?", id).Delete(&models.PostViewStat{}).Error; err != nil {
			return err
		}

		if err := tx.Unscoped().Where("post_id = ?", id).Delete(&models.Comment{}).Error; err != nil {
			return err
		}

		return tx.Unscoped().Delete(&models.Post{}, id).Error
	})
}

func (r *postRepository) GetBySlug(slug string) (*models.Post, error) {
	var post models.Post
	now := time.Now().UTC()

	err := r.db.Where("slug = ?", slug).
		Where("publish_at IS NULL OR publish_at <= ?", now).
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Preload("Comments", func(db *gorm.DB) *gorm.DB {
			return db.Where("parent_id IS NULL").Order("comments.created_at DESC")
		}).
		First(&post).Error
	return &post, err
}

func (r *postRepository) GetPopular(limit int) ([]models.Post, error) {
	var posts []models.Post
	now := time.Now().UTC()

	err := r.db.Where("published = ?", true).
		Where("publish_at IS NULL OR publish_at <= ?", now).
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Order("views DESC").
		Limit(limit).
		Find(&posts).Error
	return posts, err
}

func (r *postRepository) GetRecent(limit int) ([]models.Post, error) {
	var posts []models.Post
	now := time.Now().UTC()

	err := r.db.Where("published = ?", true).
		Where("publish_at IS NULL OR publish_at <= ?", now).
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Order("COALESCE(posts.publish_at, posts.created_at) DESC").
		Limit(limit).
		Find(&posts).Error
	return posts, err
}

func (r *postRepository) GetRelated(postID uint, categoryID uint, limit int) ([]models.Post, error) {
	var posts []models.Post
	now := time.Now().UTC()

	err := r.db.Where("id != ? AND category_id = ? AND published = ?", postID, categoryID, true).
		Where("publish_at IS NULL OR publish_at <= ?", now).
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Order("COALESCE(posts.publish_at, posts.created_at) DESC").
		Limit(limit).
		Find(&posts).Error
	return posts, err
}

func (r *postRepository) IncrementViews(id uint) error {
	now := time.Now().UTC()
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Post{}).
			Where("id = ?", id).
			UpdateColumn("views", gorm.Expr("views + ?", 1)).Error; err != nil {
			return err
		}

		result := tx.Model(&models.PostViewStat{}).
			Where("post_id = ? AND date = ?", id, date).
			UpdateColumn("views", gorm.Expr("views + ?", 1))
		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			stat := models.PostViewStat{PostID: id, Date: date, Views: 1}
			if err := tx.Create(&stat).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *postRepository) GetViewStats(postID uint, start time.Time) ([]DailyCount, error) {
	var stats []DailyCount

	query := r.db.Model(&models.PostViewStat{}).
		Select("date AS period, views AS count").
		Where("post_id = ?", postID)

	if !start.IsZero() {
		query = query.Where("date >= ?", start)
	}

	if err := query.Order("date").Scan(&stats).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

func (r *postRepository) GetAverageViews() (float64, error) {
	var result struct {
		Avg float64
	}

	err := r.db.Model(&models.Post{}).
		Where("published = ?", true).
		Select("COALESCE(AVG(views), 0) AS avg").
		Scan(&result).Error
	if err != nil {
		return 0, err
	}

	return result.Avg, nil
}

func (r *postRepository) GetAverageComments() (float64, error) {
	subQuery := r.db.Model(&models.Comment{}).
		Select("post_id, COUNT(*) AS count").
		Group("post_id")

	var result struct {
		Avg float64
	}

	err := r.db.Model(&models.Post{}).
		Where("published = ?", true).
		Joins("LEFT JOIN (?) AS comment_counts ON comment_counts.post_id = posts.id", subQuery).
		Select("COALESCE(AVG(COALESCE(comment_counts.count, 0)), 0) AS avg").
		Scan(&result).Error
	if err != nil {
		return 0, err
	}

	return result.Avg, nil
}

func (r *postRepository) GetViewRank(postID uint) (int64, int64, error) {
	var info struct {
		Views     int64
		Published bool
	}

	if err := r.db.Model(&models.Post{}).
		Select("views, published").
		Where("id = ?", postID).
		Scan(&info).Error; err != nil {
		return 0, 0, err
	}

	if !info.Published {
		return 0, 0, gorm.ErrRecordNotFound
	}

	var result struct {
		Rank  int64
		Total int64
	}

	err := r.db.Raw(`
                SELECT
                        (SELECT COUNT(*) FROM posts WHERE published = TRUE AND views > ?) + 1 AS rank,
                        (SELECT COUNT(*) FROM posts WHERE published = TRUE) AS total
        `, info.Views).Scan(&result).Error
	if err != nil {
		return 0, 0, err
	}

	return result.Rank, result.Total, nil
}

func (r *postRepository) GetCommentRank(postID uint) (int64, int64, error) {
	var result struct {
		Rank  int64
		Total int64
	}

	query := r.db.Raw(`
WITH comment_counts AS (
        SELECT p.id, COALESCE(c.count, 0) AS comment_count
        FROM posts p
        LEFT JOIN (
                SELECT post_id, COUNT(*) AS count
                FROM comments
                GROUP BY post_id
        ) c ON c.post_id = p.id
        WHERE p.published = TRUE
),
target AS (
        SELECT * FROM comment_counts WHERE id = ?
)
SELECT
        (SELECT COUNT(*) FROM comment_counts WHERE comment_count > target.comment_count) + 1 AS rank,
        (SELECT COUNT(*) FROM comment_counts) AS total
FROM target
        `, postID)

	if err := query.Scan(&result).Error; err != nil {
		return 0, 0, err
	}

	if query.RowsAffected == 0 {
		return 0, 0, gorm.ErrRecordNotFound
	}

	return result.Rank, result.Total, nil
}

func (r *postRepository) ExistsBySlug(slug string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Post{}).Where("slug = ?", slug).Count(&count).Error
	return count > 0, err
}

func (r *postRepository) ReassignCategory(fromCategoryID, toCategoryID uint) error {
	return r.db.Model(&models.Post{}).
		Where("category_id = ?", fromCategoryID).
		Update("category_id", toCategoryID).Error
}

func (r *postRepository) GetAllPublished() ([]models.Post, error) {
	var posts []models.Post
	now := time.Now().UTC()

	err := r.db.Select("id", "slug", "updated_at", "created_at", "publish_at", "published_at").
		Where("published = ?", true).
		Where("publish_at IS NULL OR publish_at <= ?", now).
		Order("COALESCE(posts.publish_at, posts.updated_at, posts.created_at) DESC").
		Find(&posts).Error
	return posts, err
}
