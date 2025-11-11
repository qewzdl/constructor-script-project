package repository

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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
	GetBySlug(slug string) (*models.CourseTopic, error)
	GetByIDs(ids []uint) ([]models.CourseTopic, error)
	List() ([]models.CourseTopic, error)
	Exists(id uint) (bool, error)
	SetSteps(topicID uint, steps []models.CourseTopicStep) error
	ListStepLinks(topicIDs []uint) (map[uint][]models.CourseTopicStep, error)
}

type CoursePackageRepository interface {
	Create(pkg *models.CoursePackage) error
	Update(pkg *models.CoursePackage) error
	Delete(id uint) error
	GetByID(id uint) (*models.CoursePackage, error)
	GetBySlug(slug string) (*models.CoursePackage, error)
	GetByIDs(ids []uint) ([]models.CoursePackage, error)
	List() ([]models.CoursePackage, error)
	Exists(id uint) (bool, error)
	SetTopics(packageID uint, topicIDs []uint) error
	ListTopicLinks(packageIDs []uint) (map[uint][]models.CoursePackageTopic, error)
}

type CoursePackageAccessRepository interface {
	Upsert(access *models.CoursePackageAccess) error
	GetByUserAndPackage(userID, packageID uint) (*models.CoursePackageAccess, error)
	ListActiveByUser(userID uint) ([]models.CoursePackageAccess, error)
}

type CourseTestRepository interface {
	Create(test *models.CourseTest) error
	Update(test *models.CourseTest) error
	Delete(id uint) error
	GetByID(id uint) (*models.CourseTest, error)
	GetByIDs(ids []uint) ([]models.CourseTest, error)
	List() ([]models.CourseTest, error)
	Exists(id uint) (bool, error)
	ReplaceStructure(testID uint, questions []models.CourseTestQuestion) error
	ListStructure(testIDs []uint) (map[uint][]models.CourseTestQuestion, error)
	SaveResult(result *models.CourseTestResult) error
	GetBestResult(testID, userID uint) (*models.CourseTestResult, int64, error)
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

type coursePackageAccessRepository struct {
	db *gorm.DB
}

type courseTestRepository struct {
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

func NewCoursePackageAccessRepository(db *gorm.DB) CoursePackageAccessRepository {
	return &coursePackageAccessRepository{db: db}
}

func NewCourseTestRepository(db *gorm.DB) CourseTestRepository {
	return &courseTestRepository{db: db}
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
		if err := tx.Where("video_id = ?", id).Delete(&models.CourseTopicStep{}).Error; err != nil {
			return err
		}
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
		if err := tx.Where("topic_id = ?", id).Delete(&models.CourseTopicStep{}).Error; err != nil {
			return err
		}
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

func (r *courseTopicRepository) GetBySlug(slug string) (*models.CourseTopic, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("course topic repository is not initialised")
	}
	cleaned := strings.TrimSpace(slug)
	if cleaned == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var topic models.CourseTopic
	if err := r.db.Where("slug = ?", cleaned).First(&topic).Error; err != nil {
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

func (r *courseTopicRepository) SetSteps(topicID uint, steps []models.CourseTopicStep) error {
	if r == nil || r.db == nil {
		return errors.New("course topic repository is not initialised")
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("topic_id = ?", topicID).Delete(&models.CourseTopicStep{}).Error; err != nil {
			return err
		}
		for idx := range steps {
			step := steps[idx]
			step.ID = 0
			step.TopicID = topicID
			step.Position = idx
			if err := tx.Create(&step).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *courseTopicRepository) ListStepLinks(topicIDs []uint) (map[uint][]models.CourseTopicStep, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("course topic repository is not initialised")
	}
	result := make(map[uint][]models.CourseTopicStep, len(topicIDs))
	if len(topicIDs) == 0 {
		return result, nil
	}
	var links []models.CourseTopicStep
	if err := r.db.Where("topic_id IN ?", topicIDs).Order("topic_id ASC, position ASC").Find(&links).Error; err != nil {
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

func (r *coursePackageRepository) GetBySlug(slug string) (*models.CoursePackage, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("course package repository is not initialised")
	}
	cleaned := strings.TrimSpace(slug)
	if cleaned == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var pkg models.CoursePackage
	if err := r.db.Where("slug = ?", cleaned).First(&pkg).Error; err != nil {
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

func (r *coursePackageAccessRepository) Upsert(access *models.CoursePackageAccess) error {
	if r == nil || r.db == nil {
		return errors.New("course package access repository is not initialised")
	}
	if access == nil {
		return errors.New("access is required")
	}

	assignments := clause.Assignments(map[string]interface{}{
		"granted_by": access.GrantedBy,
		"expires_at": access.ExpiresAt,
		"updated_at": gorm.Expr("NOW()"),
	})

	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "package_id"}},
		DoUpdates: assignments,
	}).Create(access).Error
}

func (r *coursePackageAccessRepository) GetByUserAndPackage(userID, packageID uint) (*models.CoursePackageAccess, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("course package access repository is not initialised")
	}
	var access models.CoursePackageAccess
	if err := r.db.Where("user_id = ? AND package_id = ?", userID, packageID).First(&access).Error; err != nil {
		return nil, err
	}
	return &access, nil
}

func (r *coursePackageAccessRepository) ListActiveByUser(userID uint) ([]models.CoursePackageAccess, error) {
	accesses := make([]models.CoursePackageAccess, 0)
	if r == nil || r.db == nil {
		return accesses, errors.New("course package access repository is not initialised")
	}
	if userID == 0 {
		return accesses, nil
	}

	now := time.Now()
	err := r.db.Where("user_id = ? AND (expires_at IS NULL OR expires_at > ?)", userID, now).
		Order("created_at DESC").
		Find(&accesses).Error

	return accesses, err
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

func (r *courseTestRepository) Create(test *models.CourseTest) error {
	if r == nil || r.db == nil {
		return errors.New("course test repository is not initialised")
	}
	if test == nil {
		return errors.New("test is required")
	}
	return r.db.Create(test).Error
}

func (r *courseTestRepository) Update(test *models.CourseTest) error {
	if r == nil || r.db == nil {
		return errors.New("course test repository is not initialised")
	}
	if test == nil {
		return errors.New("test is required")
	}
	return r.db.Save(test).Error
}

func (r *courseTestRepository) Delete(id uint) error {
	if r == nil || r.db == nil {
		return errors.New("course test repository is not initialised")
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("test_id = ?", id).Delete(&models.CourseTopicStep{}).Error; err != nil {
			return err
		}
		if err := tx.Where("test_id = ?", id).Delete(&models.CourseTestResult{}).Error; err != nil {
			return err
		}
		subQuery := tx.Model(&models.CourseTestQuestion{}).Select("id").Where("test_id = ?", id)
		if err := tx.Where("question_id IN (?)", subQuery).Delete(&models.CourseTestQuestionOption{}).Error; err != nil {
			return err
		}
		if err := tx.Where("test_id = ?", id).Delete(&models.CourseTestQuestion{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.CourseTest{}, id).Error
	})
}

func (r *courseTestRepository) GetByID(id uint) (*models.CourseTest, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("course test repository is not initialised")
	}
	var test models.CourseTest
	if err := r.db.First(&test, id).Error; err != nil {
		return nil, err
	}
	return &test, nil
}

func (r *courseTestRepository) GetByIDs(ids []uint) ([]models.CourseTest, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("course test repository is not initialised")
	}
	if len(ids) == 0 {
		return []models.CourseTest{}, nil
	}
	var tests []models.CourseTest
	if err := r.db.Where("id IN ?", ids).Find(&tests).Error; err != nil {
		return nil, err
	}
	return tests, nil
}

func (r *courseTestRepository) List() ([]models.CourseTest, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("course test repository is not initialised")
	}
	var tests []models.CourseTest
	if err := r.db.Order("created_at DESC").Find(&tests).Error; err != nil {
		return nil, err
	}
	return tests, nil
}

func (r *courseTestRepository) Exists(id uint) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("course test repository is not initialised")
	}
	var count int64
	if err := r.db.Model(&models.CourseTest{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *courseTestRepository) ReplaceStructure(testID uint, questions []models.CourseTestQuestion) error {
	if r == nil || r.db == nil {
		return errors.New("course test repository is not initialised")
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		subQuery := tx.Model(&models.CourseTestQuestion{}).Select("id").Where("test_id = ?", testID)
		if err := tx.Where("question_id IN (?)", subQuery).Delete(&models.CourseTestQuestionOption{}).Error; err != nil {
			return err
		}
		if err := tx.Where("test_id = ?", testID).Delete(&models.CourseTestQuestion{}).Error; err != nil {
			return err
		}
		for idx := range questions {
			question := questions[idx]
			question.ID = 0
			question.TestID = testID
			question.Position = idx
			if err := tx.Create(&question).Error; err != nil {
				return err
			}
			for optIdx := range question.Options {
				option := question.Options[optIdx]
				option.ID = 0
				option.QuestionID = question.ID
				option.Position = optIdx
				if err := tx.Create(&option).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *courseTestRepository) ListStructure(testIDs []uint) (map[uint][]models.CourseTestQuestion, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("course test repository is not initialised")
	}
	result := make(map[uint][]models.CourseTestQuestion, len(testIDs))
	if len(testIDs) == 0 {
		return result, nil
	}
	var questions []models.CourseTestQuestion
	if err := r.db.Where("test_id IN ?", testIDs).Order("test_id ASC, position ASC").Find(&questions).Error; err != nil {
		return nil, err
	}
	if len(questions) == 0 {
		return result, nil
	}
	questionIDs := make([]uint, 0, len(questions))
	for _, question := range questions {
		questionIDs = append(questionIDs, question.ID)
	}
	var options []models.CourseTestQuestionOption
	if err := r.db.Where("question_id IN ?", questionIDs).Order("question_id ASC, position ASC").Find(&options).Error; err != nil {
		return nil, err
	}
	optionsByQuestion := make(map[uint][]models.CourseTestQuestionOption, len(questionIDs))
	for _, option := range options {
		optionsByQuestion[option.QuestionID] = append(optionsByQuestion[option.QuestionID], option)
	}
	for _, question := range questions {
		question.Options = optionsByQuestion[question.ID]
		result[question.TestID] = append(result[question.TestID], question)
	}
	return result, nil
}

func (r *courseTestRepository) SaveResult(result *models.CourseTestResult) error {
	if r == nil || r.db == nil {
		return errors.New("course test repository is not initialised")
	}
	if result == nil {
		return errors.New("result is required")
	}
	return r.db.Create(result).Error
}

func (r *courseTestRepository) GetBestResult(testID, userID uint) (*models.CourseTestResult, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, errors.New("course test repository is not initialised")
	}
	if testID == 0 {
		return nil, 0, errors.New("test id is required")
	}
	if userID == 0 {
		return nil, 0, errors.New("user id is required")
	}

	var attempts int64
	query := r.db.Model(&models.CourseTestResult{}).Where("test_id = ? AND user_id = ?", testID, userID)
	if err := query.Count(&attempts).Error; err != nil {
		return nil, 0, err
	}
	if attempts == 0 {
		return nil, 0, nil
	}

	var record models.CourseTestResult
	err := r.db.
		Where("test_id = ? AND user_id = ?", testID, userID).
		Order("score DESC, max_score DESC, created_at ASC, id ASC").
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, attempts, nil
		}
		return nil, 0, err
	}

	return &record, attempts, nil
}
