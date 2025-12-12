package repository

import (
	"constructor-script-backend/plugins/blog/models"

	"gorm.io/gorm"
)

// PostRepository defines the interface for post data operations
type PostRepository interface {
	Create(post *models.Post) error
	Update(post *models.Post) error
	Delete(id uint) error
	FindByID(id uint) (*models.Post, error)
	FindBySlug(slug string) (*models.Post, error)
	FindAll(page, pageSize int, published *bool) ([]models.Post, int64, error)
	FindByCategory(categoryID uint, page, pageSize int) ([]models.Post, int64, error)
	FindByTag(tagID uint, page, pageSize int) ([]models.Post, int64, error)
	FindByAuthor(authorID uint, page, pageSize int) ([]models.Post, int64, error)
	Search(query string, page, pageSize int) ([]models.Post, int64, error)
	IncrementViews(id uint) error
	GetScheduledPosts() ([]models.Post, error)
	CountByCategory(categoryID uint) (int64, error)
}

// CategoryRepository defines the interface for category data operations
type CategoryRepository interface {
	Create(category *models.Category) error
	Update(category *models.Category) error
	Delete(id uint) error
	FindByID(id uint) (*models.Category, error)
	FindBySlug(slug string) (*models.Category, error)
	FindAll() ([]models.Category, error)
	FindWithPosts(id uint) (*models.Category, error)
}

// TagRepository defines the interface for tag data operations
type TagRepository interface {
	Create(tag *models.Tag) error
	Update(tag *models.Tag) error
	Delete(id uint) error
	FindByID(id uint) (*models.Tag, error)
	FindBySlug(slug string) (*models.Tag, error)
	FindByName(name string) (*models.Tag, error)
	FindAll() ([]models.Tag, error)
	FindOrCreate(names []string) ([]models.Tag, error)
	FindUnused() ([]models.Tag, error)
	MarkAsUnused(id uint) error
	MarkAsUsed(id uint) error
}

// CommentRepository defines the interface for comment data operations
type CommentRepository interface {
	Create(comment *models.Comment) error
	Update(comment *models.Comment) error
	Delete(id uint) error
	FindByID(id uint) (*models.Comment, error)
	FindByPost(postID uint, page, pageSize int) ([]models.Comment, int64, error)
	FindByAuthor(authorID uint, page, pageSize int) ([]models.Comment, int64, error)
	FindUnapproved(page, pageSize int) ([]models.Comment, int64, error)
	CountByPost(postID uint) (int64, error)
}

// SearchRepository defines the interface for search operations
type SearchRepository interface {
	SearchPosts(query string, filters map[string]interface{}, page, pageSize int) ([]models.Post, int64, error)
}

// Implementation using GORM

type postRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) PostRepository {
	return &postRepository{db: db}
}

type categoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

type tagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) TagRepository {
	return &tagRepository{db: db}
}

type commentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) CommentRepository {
	return &commentRepository{db: db}
}

type searchRepository struct {
	db *gorm.DB
}

func NewSearchRepository(db *gorm.DB) SearchRepository {
	return &searchRepository{db: db}
}

// PostRepository implementation
func (r *postRepository) Create(post *models.Post) error {
	return r.db.Create(post).Error
}

func (r *postRepository) Update(post *models.Post) error {
	return r.db.Save(post).Error
}

func (r *postRepository) Delete(id uint) error {
	return r.db.Delete(&models.Post{}, id).Error
}

func (r *postRepository) FindByID(id uint) (*models.Post, error) {
	var post models.Post
	err := r.db.Preload("Author").Preload("Category").Preload("Tags").Preload("Comments").First(&post, id).Error
	return &post, err
}

func (r *postRepository) FindBySlug(slug string) (*models.Post, error) {
	var post models.Post
	err := r.db.Preload("Author").Preload("Category").Preload("Tags").Where("slug = ?", slug).First(&post).Error
	return &post, err
}

func (r *postRepository) FindAll(page, pageSize int, published *bool) ([]models.Post, int64, error) {
	var posts []models.Post
	var total int64

	query := r.db.Model(&models.Post{})
	if published != nil {
		query = query.Where("published = ?", *published)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Preload("Author").Preload("Category").Preload("Tags").Find(&posts).Error

	return posts, total, err
}

func (r *postRepository) FindByCategory(categoryID uint, page, pageSize int) ([]models.Post, int64, error) {
	var posts []models.Post
	var total int64

	query := r.db.Model(&models.Post{}).Where("category_id = ? AND published = ?", categoryID, true)
	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Preload("Author").Preload("Category").Preload("Tags").Find(&posts).Error

	return posts, total, err
}

func (r *postRepository) FindByTag(tagID uint, page, pageSize int) ([]models.Post, int64, error) {
	var posts []models.Post
	var total int64

	query := r.db.Model(&models.Post{}).
		Joins("JOIN post_tags ON post_tags.post_id = posts.id").
		Where("post_tags.tag_id = ? AND posts.published = ?", tagID, true)

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Preload("Author").Preload("Category").Preload("Tags").Find(&posts).Error

	return posts, total, err
}

func (r *postRepository) FindByAuthor(authorID uint, page, pageSize int) ([]models.Post, int64, error) {
	var posts []models.Post
	var total int64

	query := r.db.Model(&models.Post{}).Where("author_id = ?", authorID)
	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Preload("Author").Preload("Category").Preload("Tags").Find(&posts).Error

	return posts, total, err
}

func (r *postRepository) Search(query string, page, pageSize int) ([]models.Post, int64, error) {
	var posts []models.Post
	var total int64

	searchQuery := r.db.Model(&models.Post{}).
		Where("published = ? AND (title ILIKE ? OR content ILIKE ? OR description ILIKE ?)",
			true, "%"+query+"%", "%"+query+"%", "%"+query+"%")

	searchQuery.Count(&total)

	offset := (page - 1) * pageSize
	err := searchQuery.Offset(offset).Limit(pageSize).Preload("Author").Preload("Category").Preload("Tags").Find(&posts).Error

	return posts, total, err
}

func (r *postRepository) IncrementViews(id uint) error {
	return r.db.Model(&models.Post{}).Where("id = ?", id).UpdateColumn("views", gorm.Expr("views + ?", 1)).Error
}

func (r *postRepository) GetScheduledPosts() ([]models.Post, error) {
	var posts []models.Post
	err := r.db.Where("published = ? AND publish_at IS NOT NULL AND publish_at <= NOW()", false).Find(&posts).Error
	return posts, err
}

func (r *postRepository) CountByCategory(categoryID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Post{}).Where("category_id = ? AND published = ?", categoryID, true).Count(&count).Error
	return count, err
}

// CategoryRepository implementation
func (r *categoryRepository) Create(category *models.Category) error {
	return r.db.Create(category).Error
}

func (r *categoryRepository) Update(category *models.Category) error {
	return r.db.Save(category).Error
}

func (r *categoryRepository) Delete(id uint) error {
	return r.db.Delete(&models.Category{}, id).Error
}

func (r *categoryRepository) FindByID(id uint) (*models.Category, error) {
	var category models.Category
	err := r.db.First(&category, id).Error
	return &category, err
}

func (r *categoryRepository) FindBySlug(slug string) (*models.Category, error) {
	var category models.Category
	err := r.db.Where("slug = ?", slug).First(&category).Error
	return &category, err
}

func (r *categoryRepository) FindAll() ([]models.Category, error) {
	var categories []models.Category
	err := r.db.Order("`order` ASC, name ASC").Find(&categories).Error
	return categories, err
}

func (r *categoryRepository) FindWithPosts(id uint) (*models.Category, error) {
	var category models.Category
	err := r.db.Preload("Posts").First(&category, id).Error
	return &category, err
}

// TagRepository implementation
func (r *tagRepository) Create(tag *models.Tag) error {
	return r.db.Create(tag).Error
}

func (r *tagRepository) Update(tag *models.Tag) error {
	return r.db.Save(tag).Error
}

func (r *tagRepository) Delete(id uint) error {
	return r.db.Delete(&models.Tag{}, id).Error
}

func (r *tagRepository) FindByID(id uint) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.First(&tag, id).Error
	return &tag, err
}

func (r *tagRepository) FindBySlug(slug string) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.Where("slug = ?", slug).First(&tag).Error
	return &tag, err
}

func (r *tagRepository) FindByName(name string) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.Where("name = ?", name).First(&tag).Error
	return &tag, err
}

func (r *tagRepository) FindAll() ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.Find(&tags).Error
	return tags, err
}

func (r *tagRepository) FindOrCreate(names []string) ([]models.Tag, error) {
	var tags []models.Tag
	for _, name := range names {
		var tag models.Tag
		err := r.db.Where("name = ?", name).FirstOrCreate(&tag, models.Tag{Name: name, Slug: name}).Error
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func (r *tagRepository) FindUnused() ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.Where("unused_since IS NOT NULL").Find(&tags).Error
	return tags, err
}

func (r *tagRepository) MarkAsUnused(id uint) error {
	now := gorm.Expr("NOW()")
	return r.db.Model(&models.Tag{}).Where("id = ?", id).Update("unused_since", now).Error
}

func (r *tagRepository) MarkAsUsed(id uint) error {
	return r.db.Model(&models.Tag{}).Where("id = ?", id).Update("unused_since", nil).Error
}

// CommentRepository implementation
func (r *commentRepository) Create(comment *models.Comment) error {
	return r.db.Create(comment).Error
}

func (r *commentRepository) Update(comment *models.Comment) error {
	return r.db.Save(comment).Error
}

func (r *commentRepository) Delete(id uint) error {
	return r.db.Delete(&models.Comment{}, id).Error
}

func (r *commentRepository) FindByID(id uint) (*models.Comment, error) {
	var comment models.Comment
	err := r.db.Preload("Author").Preload("Replies").First(&comment, id).Error
	return &comment, err
}

func (r *commentRepository) FindByPost(postID uint, page, pageSize int) ([]models.Comment, int64, error) {
	var comments []models.Comment
	var total int64

	query := r.db.Model(&models.Comment{}).Where("post_id = ? AND parent_id IS NULL", postID)
	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Preload("Author").Preload("Replies").Find(&comments).Error

	return comments, total, err
}

func (r *commentRepository) FindByAuthor(authorID uint, page, pageSize int) ([]models.Comment, int64, error) {
	var comments []models.Comment
	var total int64

	query := r.db.Model(&models.Comment{}).Where("author_id = ?", authorID)
	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Preload("Post").Find(&comments).Error

	return comments, total, err
}

func (r *commentRepository) FindUnapproved(page, pageSize int) ([]models.Comment, int64, error) {
	var comments []models.Comment
	var total int64

	query := r.db.Model(&models.Comment{}).Where("approved = ?", false)
	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Preload("Author").Preload("Post").Find(&comments).Error

	return comments, total, err
}

func (r *commentRepository) CountByPost(postID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Comment{}).Where("post_id = ?", postID).Count(&count).Error
	return count, err
}

// SearchRepository implementation
func (r *searchRepository) SearchPosts(query string, filters map[string]interface{}, page, pageSize int) ([]models.Post, int64, error) {
	var posts []models.Post
	var total int64

	searchQuery := r.db.Model(&models.Post{}).
		Where("published = ? AND (title ILIKE ? OR content ILIKE ? OR description ILIKE ?)",
			true, "%"+query+"%", "%"+query+"%", "%"+query+"%")

	for key, value := range filters {
		searchQuery = searchQuery.Where(key+" = ?", value)
	}

	searchQuery.Count(&total)

	offset := (page - 1) * pageSize
	err := searchQuery.Offset(offset).Limit(pageSize).
		Preload("Author").Preload("Category").Preload("Tags").
		Find(&posts).Error

	return posts, total, err
}
