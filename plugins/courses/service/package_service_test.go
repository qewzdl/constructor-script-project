package service

import (
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
)

type mockPackageRepo struct {
	pkg      *models.CoursePackage
	packages map[uint]models.CoursePackage
	topics   map[uint][]models.CoursePackageTopic
}

func (m *mockPackageRepo) Create(pkg *models.CoursePackage) error { return nil }
func (m *mockPackageRepo) Update(pkg *models.CoursePackage) error { return nil }
func (m *mockPackageRepo) Delete(id uint) error                   { return nil }
func (m *mockPackageRepo) GetByID(id uint) (*models.CoursePackage, error) {
	if m.pkg != nil && m.pkg.ID == id {
		copy := *m.pkg
		return &copy, nil
	}
	if pkg, ok := m.packages[id]; ok {
		copy := pkg
		return &copy, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockPackageRepo) GetByIDs(ids []uint) ([]models.CoursePackage, error) {
	result := make([]models.CoursePackage, 0, len(ids))
	for _, id := range ids {
		if m.pkg != nil && m.pkg.ID == id {
			copy := *m.pkg
			result = append(result, copy)
			continue
		}
		if m.packages == nil {
			continue
		}
		if pkg, ok := m.packages[id]; ok {
			copy := pkg
			result = append(result, copy)
		}
	}
	return result, nil
}
func (m *mockPackageRepo) List() ([]models.CoursePackage, error) {
	return []models.CoursePackage{}, nil
}
func (m *mockPackageRepo) Exists(id uint) (bool, error) {
	if m.pkg != nil && m.pkg.ID == id {
		return true, nil
	}
	if m.packages != nil {
		_, ok := m.packages[id]
		return ok, nil
	}
	return false, nil
}
func (m *mockPackageRepo) SetTopics(packageID uint, topicIDs []uint) error { return nil }
func (m *mockPackageRepo) ListTopicLinks(packageIDs []uint) (map[uint][]models.CoursePackageTopic, error) {
	if len(packageIDs) == 0 {
		return map[uint][]models.CoursePackageTopic{}, nil
	}
	result := make(map[uint][]models.CoursePackageTopic, len(packageIDs))
	if m.topics == nil {
		return result, nil
	}
	for _, id := range packageIDs {
		if links, ok := m.topics[id]; ok {
			cloned := make([]models.CoursePackageTopic, len(links))
			copy(cloned, links)
			result[id] = cloned
		}
	}
	return result, nil
}

type mockTopicRepo struct {
	topics map[uint]models.CourseTopic
	steps  map[uint][]models.CourseTopicStep
}

type mockVideoRepo struct {
	videos map[uint]models.CourseVideo
}

type mockTestRepo struct {
	tests      map[uint]models.CourseTest
	structures map[uint][]models.CourseTestQuestion
}

type mockAccessRepo struct {
	access    *models.CoursePackageAccess
	err       error
	list      []models.CoursePackageAccess
	listErr   error
	accessMap map[[2]uint]*models.CoursePackageAccess
}

func (m *mockTopicRepo) Create(topic *models.CourseTopic) error { return nil }
func (m *mockTopicRepo) Update(topic *models.CourseTopic) error { return nil }
func (m *mockTopicRepo) Delete(id uint) error                   { return nil }
func (m *mockTopicRepo) GetByID(id uint) (*models.CourseTopic, error) {
	if topic, ok := m.topics[id]; ok {
		copy := topic
		return &copy, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockTopicRepo) GetByIDs(ids []uint) ([]models.CourseTopic, error) {
	result := make([]models.CourseTopic, 0, len(ids))
	for _, id := range ids {
		if topic, ok := m.topics[id]; ok {
			copy := topic
			result = append(result, copy)
		}
	}
	return result, nil
}
func (m *mockTopicRepo) List() ([]models.CourseTopic, error)                         { return []models.CourseTopic{}, nil }
func (m *mockTopicRepo) Exists(id uint) (bool, error)                                { return false, nil }
func (m *mockTopicRepo) SetSteps(topicID uint, steps []models.CourseTopicStep) error { return nil }
func (m *mockTopicRepo) ListStepLinks(topicIDs []uint) (map[uint][]models.CourseTopicStep, error) {
	result := make(map[uint][]models.CourseTopicStep, len(topicIDs))
	if m.steps == nil {
		return result, nil
	}
	for _, id := range topicIDs {
		if links, ok := m.steps[id]; ok {
			cloned := make([]models.CourseTopicStep, len(links))
			copy(cloned, links)
			result[id] = cloned
		}
	}
	return result, nil
}

func (m *mockVideoRepo) Create(video *models.CourseVideo) error { return nil }
func (m *mockVideoRepo) Update(video *models.CourseVideo) error { return nil }
func (m *mockVideoRepo) Delete(id uint) error                   { return nil }
func (m *mockVideoRepo) GetByID(id uint) (*models.CourseVideo, error) {
	if video, ok := m.videos[id]; ok {
		copy := video
		return &copy, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockVideoRepo) List() ([]models.CourseVideo, error) { return []models.CourseVideo{}, nil }
func (m *mockVideoRepo) Exists(id uint) (bool, error) {
	if m.videos == nil {
		return false, nil
	}
	_, ok := m.videos[id]
	return ok, nil
}
func (m *mockVideoRepo) GetByIDs(ids []uint) ([]models.CourseVideo, error) {
	result := make([]models.CourseVideo, 0, len(ids))
	for _, id := range ids {
		if video, ok := m.videos[id]; ok {
			copy := video
			result = append(result, copy)
		}
	}
	return result, nil
}

func (m *mockTestRepo) Create(test *models.CourseTest) error { return nil }
func (m *mockTestRepo) Update(test *models.CourseTest) error { return nil }
func (m *mockTestRepo) Delete(id uint) error                 { return nil }
func (m *mockTestRepo) GetByID(id uint) (*models.CourseTest, error) {
	if test, ok := m.tests[id]; ok {
		copy := test
		return &copy, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockTestRepo) GetByIDs(ids []uint) ([]models.CourseTest, error) {
	result := make([]models.CourseTest, 0, len(ids))
	for _, id := range ids {
		if test, ok := m.tests[id]; ok {
			copy := test
			result = append(result, copy)
		}
	}
	return result, nil
}
func (m *mockTestRepo) List() ([]models.CourseTest, error) { return []models.CourseTest{}, nil }
func (m *mockTestRepo) Exists(id uint) (bool, error) {
	if m.tests == nil {
		return false, nil
	}
	_, ok := m.tests[id]
	return ok, nil
}
func (m *mockTestRepo) ReplaceStructure(testID uint, questions []models.CourseTestQuestion) error {
	return nil
}
func (m *mockTestRepo) ListStructure(testIDs []uint) (map[uint][]models.CourseTestQuestion, error) {
	result := make(map[uint][]models.CourseTestQuestion, len(testIDs))
	if m.structures == nil {
		return result, nil
	}
	for _, id := range testIDs {
		if questions, ok := m.structures[id]; ok {
			cloned := make([]models.CourseTestQuestion, len(questions))
			copy(cloned, questions)
			result[id] = cloned
		}
	}
	return result, nil
}
func (m *mockTestRepo) SaveResult(result *models.CourseTestResult) error { return nil }

func (m *mockAccessRepo) Upsert(access *models.CoursePackageAccess) error { return nil }

func (m *mockAccessRepo) GetByUserAndPackage(userID, packageID uint) (*models.CoursePackageAccess, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.access != nil && m.access.UserID == userID && m.access.PackageID == packageID {
		copy := *m.access
		return &copy, nil
	}
	if m.accessMap != nil {
		if access, ok := m.accessMap[[2]uint{userID, packageID}]; ok && access != nil {
			copy := *access
			return &copy, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockAccessRepo) ListActiveByUser(userID uint) ([]models.CoursePackageAccess, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	if len(m.list) == 0 {
		return []models.CoursePackageAccess{}, nil
	}
	result := make([]models.CoursePackageAccess, len(m.list))
	copy(result, m.list)
	return result, nil
}

func TestPackageServiceGetForUser(t *testing.T) {
	pkg := &models.CoursePackage{ID: 7, Title: "Advanced Go", Description: "Deep dive"}
	access := &models.CoursePackageAccess{UserID: 3, PackageID: 7, CreatedAt: time.Now()}

	svc := &PackageService{
		packageRepo: &mockPackageRepo{pkg: pkg},
		topicRepo:   &mockTopicRepo{},
		videoRepo:   &mockVideoRepo{},
		testRepo:    &mockTestRepo{},
		accessRepo:  &mockAccessRepo{access: access},
	}

	userCourse, err := svc.GetForUser(pkg.ID, access.UserID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if userCourse == nil {
		t.Fatalf("expected course data")
	}
	if userCourse.Package.ID != pkg.ID {
		t.Fatalf("expected package id %d, got %d", pkg.ID, userCourse.Package.ID)
	}
	if userCourse.Access.UserID != access.UserID {
		t.Fatalf("expected access for user %d, got %d", access.UserID, userCourse.Access.UserID)
	}
}

func TestPackageServiceGetForUserExpired(t *testing.T) {
	past := time.Now().Add(-2 * time.Hour)
	pkg := &models.CoursePackage{ID: 1, Title: "Basics"}
	access := &models.CoursePackageAccess{UserID: 9, PackageID: 1, ExpiresAt: &past}

	svc := &PackageService{
		packageRepo: &mockPackageRepo{pkg: pkg},
		topicRepo:   &mockTopicRepo{},
		videoRepo:   &mockVideoRepo{},
		testRepo:    &mockTestRepo{},
		accessRepo:  &mockAccessRepo{access: access},
	}

	_, err := svc.GetForUser(pkg.ID, access.UserID)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found error for expired access, got %v", err)
	}
}

func TestPackageServiceGetForUserValidation(t *testing.T) {
	svc := &PackageService{
		packageRepo: &mockPackageRepo{},
		topicRepo:   &mockTopicRepo{},
		videoRepo:   &mockVideoRepo{},
		testRepo:    &mockTestRepo{},
		accessRepo:  &mockAccessRepo{},
	}

	if _, err := svc.GetForUser(0, 1); err == nil || !IsValidationError(err) {
		t.Fatalf("expected validation error for missing package id")
	}

	if _, err := svc.GetForUser(1, 0); err == nil || !IsValidationError(err) {
		t.Fatalf("expected validation error for missing user id")
	}
}

func TestPackageServiceListForUserPopulatesContent(t *testing.T) {
	userID := uint(42)
	now := time.Now()

	videoID := uint(11)
	testID := uint(22)
	topicID := uint(33)

	packages := map[uint]models.CoursePackage{
		1: {ID: 1, Title: "Go Foundations"},
		2: {ID: 2, Title: "Go Patterns"},
	}

	accessRepo := &mockAccessRepo{
		list: []models.CoursePackageAccess{
			{UserID: userID, PackageID: 1, CreatedAt: now.Add(-time.Hour)},
			{UserID: userID, PackageID: 2, CreatedAt: now},
		},
	}

	accessRepo.accessMap = map[[2]uint]*models.CoursePackageAccess{
		{userID, 1}: {UserID: userID, PackageID: 1, CreatedAt: now},
		{userID, 2}: {UserID: userID, PackageID: 2, CreatedAt: now},
	}

	packageRepo := &mockPackageRepo{
		packages: packages,
		topics: map[uint][]models.CoursePackageTopic{
			1: {{PackageID: 1, TopicID: topicID, Position: 1}},
		},
	}

	topicRepo := &mockTopicRepo{
		topics: map[uint]models.CourseTopic{
			topicID: {ID: topicID, Title: "Concurrency", Description: "Manage goroutines"},
		},
                steps: map[uint][]models.CourseTopicStep{
                        topicID: {
                                {ID: 100, TopicID: topicID, StepType: models.CourseTopicStepTypeVideo, Position: 1, VideoID: &videoID},
                                {ID: 101, TopicID: topicID, StepType: models.CourseTopicStepTypeTest, Position: 2, TestID: &testID},
                        },
                },
	}

	videoRepo := &mockVideoRepo{
		videos: map[uint]models.CourseVideo{
			videoID: {
				ID:              videoID,
				Title:           "Intro to Goroutines",
				Description:     "Understand concurrency primitives",
				DurationSeconds: 180,
				FileURL:         "https://cdn.example.com/videos/goroutines.mp4",
			},
		},
	}

	questions := []models.CourseTestQuestion{
		{
			ID:     200,
			Prompt: "What does go routine scheduling rely on?",
			Type:   models.CourseTestQuestionTypeText,
		},
	}

	testRepo := &mockTestRepo{
		tests: map[uint]models.CourseTest{
			testID: {
				ID:          testID,
				Title:       "Concurrency quiz",
				Description: "Check your concurrency knowledge",
			},
		},
		structures: map[uint][]models.CourseTestQuestion{
			testID: questions,
		},
	}

	svc := &PackageService{
		packageRepo: packageRepo,
		topicRepo:   topicRepo,
		videoRepo:   videoRepo,
		testRepo:    testRepo,
		accessRepo:  accessRepo,
	}

	result, err := svc.ListForUser(userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 course packages, got %d", len(result))
	}

	first := result[0]
	if first.Package.ID != 1 {
		t.Fatalf("expected first package id 1, got %d", first.Package.ID)
	}

	if len(first.Package.Topics) != 1 {
		t.Fatalf("expected 1 topic for first package, got %d", len(first.Package.Topics))
	}

	topic := first.Package.Topics[0]
	if len(topic.Steps) != 2 {
		t.Fatalf("expected 2 steps in topic, got %d", len(topic.Steps))
	}

	videoStep := topic.Steps[0]
	if videoStep.Video == nil || videoStep.Video.ID != videoID {
		t.Fatalf("expected populated video for step, got %#v", videoStep.Video)
	}

	testStep := topic.Steps[1]
	if testStep.Test == nil || len(testStep.Test.Questions) != len(questions) {
		t.Fatalf("expected populated test with questions")
	}

	second := result[1]
	if second.Package.ID != 2 {
		t.Fatalf("expected second package id 2, got %d", second.Package.ID)
	}
	if len(second.Package.Topics) != 0 {
		t.Fatalf("expected no topics for second package, got %d", len(second.Package.Topics))
	}
}
