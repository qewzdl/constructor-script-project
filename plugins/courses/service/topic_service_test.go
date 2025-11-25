package service

import (
	"errors"
	"strings"
	"testing"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
)

func TestTopicServiceGetByIdentifierWithSlug(t *testing.T) {
	topic := models.CourseTopic{ID: 12, Title: "Intro", Slug: "intro"}

	topicRepo := &mockTopicRepo{topics: map[uint]models.CourseTopic{topic.ID: topic}}
	videoRepo := &mockVideoRepo{}
	testRepo := &mockTestRepo{}

	svc := NewTopicService(topicRepo, videoRepo, testRepo)

	result, err := svc.GetByIdentifier("Intro")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatalf("expected topic, got nil")
	}
	if result.ID != topic.ID {
		t.Fatalf("expected topic id %d, got %d", topic.ID, result.ID)
	}
	if result.Slug != topic.Slug {
		t.Fatalf("expected slug %q, got %q", topic.Slug, result.Slug)
	}
}

func TestTopicServiceGetByIdentifierWithNumericID(t *testing.T) {
	topic := models.CourseTopic{ID: 21, Title: "Testing", Slug: "testing"}

	topicRepo := &mockTopicRepo{topics: map[uint]models.CourseTopic{topic.ID: topic}}
	videoRepo := &mockVideoRepo{}
	testRepo := &mockTestRepo{}

	svc := NewTopicService(topicRepo, videoRepo, testRepo)

	result, err := svc.GetByIdentifier("21")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatalf("expected topic, got nil")
	}
	if result.ID != topic.ID {
		t.Fatalf("expected topic id %d, got %d", topic.ID, result.ID)
	}
	if result.Slug != topic.Slug {
		t.Fatalf("expected slug %q, got %q", topic.Slug, result.Slug)
	}
}

func TestTopicServiceGetByIdentifierRequiresIdentifier(t *testing.T) {
	svc := NewTopicService(&mockTopicRepo{}, &mockVideoRepo{}, &mockTestRepo{})

	if _, err := svc.GetByIdentifier(" "); err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found error for empty identifier")
	}
}

func TestTopicServiceUpdateStepsSupportsContent(t *testing.T) {
	topicID := uint(7)
	repo := &mockTopicRepo{topics: map[uint]models.CourseTopic{topicID: {ID: topicID, Title: "Basics", Slug: "basics"}}}

	svc := NewTopicService(repo, &mockVideoRepo{}, &mockTestRepo{})
	sections := []models.Section{{Title: "Overview", Type: "hero"}}
	steps := []models.CourseTopicStepReference{
		{Type: models.CourseTopicStepTypeContent, Title: "Intro", Sections: sections},
	}

	updated, err := svc.UpdateSteps(topicID, steps)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated == nil {
		t.Fatalf("expected updated topic")
	}

	savedSteps := repo.steps[topicID]
	if len(savedSteps) != 1 {
		t.Fatalf("expected 1 saved step, got %d", len(savedSteps))
	}
	saved := savedSteps[0]
	if saved.StepType != models.CourseTopicStepTypeContent {
		t.Fatalf("expected content step type, got %s", saved.StepType)
	}
	if strings.TrimSpace(saved.Title) != "Intro" {
		t.Fatalf("expected step title to be preserved")
	}
	if len(saved.Sections) != len(sections) {
		t.Fatalf("expected sections to be stored")
	}
}
