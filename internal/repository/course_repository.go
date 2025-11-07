package repository

import (
	"errors"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
)

type CourseVideoRepository interface {
	Create(video *models.CourseVideo) error
	Update(video *models.CourseVideo) error
	Delete(id uint) error
	GetByID(id uint) (*models.CourseVideo, error)
	List() ([]models.CourseVideo, error)
	Exists(id uint) (bool, error)
	GetByIDs(ids []uint) ([]models.CourseVideo, error)
}

type CourseTopicRepository interface {
	Create(topic *models.CourseTopic) error
	Update(topic *models.CourseTopic) error
	Delete(id uint) error
	GetByID(id uint) (*models.CourseTopic, error)
	GetByIDs(ids []uint) ([]models.CourseTopic, error)
	List() ([]models.CourseTopic, error)
	Exists(id uint) (bool, error)
	SetVideos(topicID uint, videoIDs []uint) error
	ListVideoLinks(topicIDs []uint) (map[uint][]models.CourseTopicVideo, error)
}

type CoursePackageRepository interface {
	Create(pkg *models.CoursePackage) error
	Update(pkg *models.CoursePackage) error
	Delete(id uint) error
	GetByID(id uint) (*models.CoursePackage, error)
	GetByIDs(ids []uint) ([]models.CoursePackage, error)
	List() ([]models.CoursePackage, error)
	Exists(id uint) (bool, error)
	SetTopics(packageID uint, topicIDs []uint) error
	ListTopicLinks(packageIDs []uint) (map[uint][]models.CoursePackageTopic, error)
}

type courseVideoRepository struct {
	db *gorm.DB
}

type courseTopicRepository struct {
	db *gorm.DB
}

type coursePackageRepository struct {
	db *gorm.DB
}

func NewCourseVideoRepository(db *gorm.DB) CourseVideoRepository {
	return &courseVideoRepository{db: db}
}

func NewCourseTopicRepository(db *gorm.DB) CourseTopicRepository {
	return &courseTopicRepository{db: db}
}

func NewCoursePackageRepository(db *gorm.DB) CoursePackageRepository {
	return &coursePackageRepository{db: db}
}

func (r *courseVideoRepository) Create(video *models.CourseVideo) error {
	if r == nil || r.db == nil {
		return errors.New("course video repository is not initialised")
	}
	if video == nil {
		return errors.New("video is required")
	}
	return r.db.Create(video).Error
}

func (r *courseVideoRepository) Update(video *models.CourseVideo) error {
	if r == nil || r.db == nil {
		return errors.New("course video repository is not initialised")
	}
	if video == nil {
		return errors.New("video is required")
	}
	return r.db.Save(video).Error
}

func (r *courseVideoRepository) Delete(id uint) error {
	if r == nil || r.db == nil {
		return errors.New("course video repository is not initialised")
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("video_id = ?", id).Delete(&models.CourseTopicVideo{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.CourseVideo{}, id).Error
	})
}

func (r *courseVideoRepository) GetByID(id uint) (*models.CourseVideo, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("course video repository is not initialised")
	}
	var video models.CourseVideo
	if err := r.db.First(&video, id).Error; err != nil {
		return nil, err
	}
	return &video, nil
}

func (r *courseVideoRepository) List() ([]models.CourseVideo, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("course video repository is not initialised")
	}
	var videos []models.CourseVideo
	if err := r.db.Order("created_at DESC").Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}

func (r *courseVideoRepository) Exists(id uint) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("course video repository is not initialised")
	}
	var count int64
	if err := r.db.Model(&models.CourseVideo{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *courseVideoRepository) GetByIDs(ids []uint) ([]models.CourseVideo, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("course video repository is not initialised")
	}
	if len(ids) == 0 {
		return []models.CourseVideo{}, nil
	}
	var videos []models.CourseVideo
	if err := r.db.Where("id IN ?", ids).Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}

func (r *courseTopicRepository) Create(topic *models.CourseTopic) error {
	if r == nil || r.db == nil {
		return errors.New("course topic repository is not initialised")
	}
	if topic == nil {
		return errors.New("topic is required")
	}
	return r.db.Create(topic).Error
}

func (r *courseTopicRepository) Update(topic *models.CourseTopic) error {
	if r == nil || r.db == nil {
		return errors.New("course topic repository is not initialised")
	}
	if topic == nil {
		return errors.New("topic is required")
	}
	return r.db.Save(topic).Error
}

func (r *courseTopicRepository) Delete(id uint) error {
	if r == nil || r.db == nil {
		return errors.New("course topic repository is not initialised")
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("topic_id = ?", id).Delete(&models.CourseTopicVideo{}).Error; err != nil {
			return err
		}
		if err := tx.Where("topic_id = ?", id).Delete(&models.CoursePackageTopic{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.CourseTopic{}, id).Error
	})
}

func (r *courseTopicRepository) GetByID(id uint) (*models.CourseTopic, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("course topic repository is not initialised")
	}
	var topic models.CourseTopic
	if err := r.db.First(&topic, id).Error; err != nil {
		return nil, err
	}
	return &topic, nil
}

func (r *courseTopicRepository) GetByIDs(ids []uint) ([]models.CourseTopic, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("course topic repository is not initialised")
	}
	if len(ids) == 0 {
		return []models.CourseTopic{}, nil
	}
	var topics []models.CourseTopic
	if err := r.db.Where("id IN ?", ids).Find(&topics).Error; err != nil {
		return nil, err
	}
	return topics, nil
}

func (r *courseTopicRepository) List() ([]models.CourseTopic, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("course topic repository is not initialised")
	}
	var topics []models.CourseTopic
	if err := r.db.Order("created_at ASC").Find(&topics).Error; err != nil {
		return nil, err
	}
	return topics, nil
}

func (r *courseTopicRepository) Exists(id uint) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("course topic repository is not initialised")
	}
	var count int64
	if err := r.db.Model(&models.CourseTopic{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *courseTopicRepository) SetVideos(topicID uint, videoIDs []uint) error {
	if r == nil || r.db == nil {
		return errors.New("course topic repository is not initialised")
	}
	ordered := uniqueOrdered(videoIDs)
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("topic_id = ?", topicID).Delete(&models.CourseTopicVideo{}).Error; err != nil {
			return err
		}
		for idx, videoID := range ordered {
			link := models.CourseTopicVideo{
				TopicID:  topicID,
				VideoID:  videoID,
				Position: idx,
			}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *courseTopicRepository) ListVideoLinks(topicIDs []uint) (map[uint][]models.CourseTopicVideo, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("course topic repository is not initialised")
	}
	result := make(map[uint][]models.CourseTopicVideo, len(topicIDs))
	if len(topicIDs) == 0 {
		return result, nil
	}
	var links []models.CourseTopicVideo
	if err := r.db.Where("topic_id IN ?", topicIDs).Order("position ASC").Find(&links).Error; err != nil {
		return nil, err
	}
	for _, link := range links {
		result[link.TopicID] = append(result[link.TopicID], link)
	}
	return result, nil
}

func (r *coursePackageRepository) Create(pkg *models.CoursePackage) error {
	if r == nil || r.db == nil {
		return errors.New("course package repository is not initialised")
	}
	if pkg == nil {
		return errors.New("package is required")
	}
	return r.db.Create(pkg).Error
}

func (r *coursePackageRepository) Update(pkg *models.CoursePackage) error {
	if r == nil || r.db == nil {
		return errors.New("course package repository is not initialised")
	}
	if pkg == nil {
		return errors.New("package is required")
	}
	return r.db.Save(pkg).Error
}

func (r *coursePackageRepository) Delete(id uint) error {
	if r == nil || r.db == nil {
		return errors.New("course package repository is not initialised")
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("package_id = ?", id).Delete(&models.CoursePackageTopic{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.CoursePackage{}, id).Error
	})
}

func (r *coursePackageRepository) GetByID(id uint) (*models.CoursePackage, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("course package repository is not initialised")
	}
	var pkg models.CoursePackage
	if err := r.db.First(&pkg, id).Error; err != nil {
		return nil, err
	}
	return &pkg, nil
}

func (r *coursePackageRepository) GetByIDs(ids []uint) ([]models.CoursePackage, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("course package repository is not initialised")
	}
	if len(ids) == 0 {
		return []models.CoursePackage{}, nil
	}
	var pkgs []models.CoursePackage
	if err := r.db.Where("id IN ?", ids).Find(&pkgs).Error; err != nil {
		return nil, err
	}
	return pkgs, nil
}

func (r *coursePackageRepository) List() ([]models.CoursePackage, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("course package repository is not initialised")
	}
	var pkgs []models.CoursePackage
	if err := r.db.Order("created_at DESC").Find(&pkgs).Error; err != nil {
		return nil, err
	}
	return pkgs, nil
}

func (r *coursePackageRepository) Exists(id uint) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("course package repository is not initialised")
	}
	var count int64
	if err := r.db.Model(&models.CoursePackage{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *coursePackageRepository) SetTopics(packageID uint, topicIDs []uint) error {
	if r == nil || r.db == nil {
		return errors.New("course package repository is not initialised")
	}
	ordered := uniqueOrdered(topicIDs)
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("package_id = ?", packageID).Delete(&models.CoursePackageTopic{}).Error; err != nil {
			return err
		}
		for idx, topicID := range ordered {
			link := models.CoursePackageTopic{
				PackageID: packageID,
				TopicID:   topicID,
				Position:  idx,
			}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *coursePackageRepository) ListTopicLinks(packageIDs []uint) (map[uint][]models.CoursePackageTopic, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("course package repository is not initialised")
	}
	result := make(map[uint][]models.CoursePackageTopic, len(packageIDs))
	if len(packageIDs) == 0 {
		return result, nil
	}
	var links []models.CoursePackageTopic
	if err := r.db.Where("package_id IN ?", packageIDs).Order("position ASC").Find(&links).Error; err != nil {
		return nil, err
	}
	for _, link := range links {
		result[link.PackageID] = append(result[link.PackageID], link)
	}
	return result, nil
}

func uniqueOrdered(values []uint) []uint {
	if len(values) == 0 {
		return []uint{}
	}
	// preserve first occurrence order
	seen := make(map[uint]struct{}, len(values))
	ordered := make([]uint, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		ordered = append(ordered, value)
	}
	return ordered
}
