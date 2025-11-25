package service

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
)

func TestTopicServiceGetByIdentifierWithSlug(t *testing.T) {
	topic := models.CourseTopic{ID: 12, Title: "Intro", Slug: "intro"}

	topicRepo := &mockTopicRepo{topics: map[uint]models.CourseTopic{topic.ID: topic}}
	videoRepo := &mockVideoRepo{}
	testRepo := &mockTestRepo{}
	contentRepo := &mockContentRepo{}

	svc := NewTopicService(topicRepo, videoRepo, testRepo, contentRepo)

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
	contentRepo := &mockContentRepo{}

	svc := NewTopicService(topicRepo, videoRepo, testRepo, contentRepo)

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
	svc := NewTopicService(&mockTopicRepo{}, &mockVideoRepo{}, &mockTestRepo{}, &mockContentRepo{})

	if _, err := svc.GetByIdentifier(" "); err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found error for empty identifier")
	}
}
