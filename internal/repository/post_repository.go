package repository

import (
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

	if published != nil {
		query = query.Where("published = ?", *published)
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
		Order("created_at DESC").
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

		if err := tx.Unscoped().Where("post_id = ?", id).Delete(&models.Comment{}).Error; err != nil {
			return err
		}

		return tx.Unscoped().Delete(&models.Post{}, id).Error
	})
}

func (r *postRepository) GetBySlug(slug string) (*models.Post, error) {
	var post models.Post
	err := r.db.Where("slug = ?", slug).
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Preload("Comments", func(db *gorm.DB) *gorm.DB {
			return db.Where("parent_id IS NULL").Order("created_at DESC")
		}).
		First(&post).Error
	return &post, err
}

func (r *postRepository) GetPopular(limit int) ([]models.Post, error) {
	var posts []models.Post
	err := r.db.Where("published = ?", true).
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
	err := r.db.Where("published = ?", true).
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Order("created_at DESC").
		Limit(limit).
		Find(&posts).Error
	return posts, err
}

func (r *postRepository) GetRelated(postID uint, categoryID uint, limit int) ([]models.Post, error) {
	var posts []models.Post
	err := r.db.Where("id != ? AND category_id = ? AND published = ?", postID, categoryID, true).
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Order("created_at DESC").
		Limit(limit).
		Find(&posts).Error
	return posts, err
}

func (r *postRepository) IncrementViews(id uint) error {
	return r.db.Model(&models.Post{}).
		Where("id = ?", id).
		UpdateColumn("views", gorm.Expr("views + ?", 1)).Error
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
	err := r.db.Select("id", "slug", "updated_at", "created_at").
		Where("published = ?", true).
		Order("updated_at DESC").
		Find(&posts).Error
	return posts, err
}
