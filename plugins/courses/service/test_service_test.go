package service

import (
	"encoding/json"
	"testing"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
)

type mockCourseTestRepository struct {
	test       *models.CourseTest
	structures map[uint][]models.CourseTestQuestion
	saved      []*models.CourseTestResult
}

func (m *mockCourseTestRepository) Create(test *models.CourseTest) error { return nil }
func (m *mockCourseTestRepository) Update(test *models.CourseTest) error { return nil }
func (m *mockCourseTestRepository) Delete(id uint) error                 { return nil }
func (m *mockCourseTestRepository) GetByID(id uint) (*models.CourseTest, error) {
	return m.test, nil
}
func (m *mockCourseTestRepository) GetByIDs(ids []uint) ([]models.CourseTest, error) {
	if m.test == nil {
		return []models.CourseTest{}, nil
	}
	return []models.CourseTest{*m.test}, nil
}
func (m *mockCourseTestRepository) List() ([]models.CourseTest, error) {
	return []models.CourseTest{}, nil
}
func (m *mockCourseTestRepository) Exists(id uint) (bool, error) { return true, nil }
func (m *mockCourseTestRepository) ReplaceStructure(testID uint, questions []models.CourseTestQuestion) error {
	return nil
}
func (m *mockCourseTestRepository) ListStructure(testIDs []uint) (map[uint][]models.CourseTestQuestion, error) {
	return m.structures, nil
}
func (m *mockCourseTestRepository) SaveResult(result *models.CourseTestResult) error {
	m.saved = append(m.saved, result)
	return nil
}

func (m *mockCourseTestRepository) GetBestResult(testID, userID uint) (*models.CourseTestResult, int64, error) {
	var attempts int64
	var best *models.CourseTestResult
	for _, result := range m.saved {
		if result == nil || result.TestID != testID || result.UserID != userID {
			continue
		}
		attempts++
		if best == nil || result.Score > best.Score || (result.Score == best.Score && result.MaxScore > best.MaxScore) {
			copy := *result
			best = &copy
		}
	}
	return best, attempts, nil
}

func TestTestServiceBuildQuestionModels(t *testing.T) {
	svc := &TestService{}

	questions, err := svc.buildQuestionModels([]models.CourseTestQuestionRequest{
		{
			Prompt:     "What is Go?",
			Type:       models.CourseTestQuestionTypeText,
			AnswerText: "A language",
		},
		{
			Prompt: "Pick one",
			Type:   models.CourseTestQuestionTypeSingleChoice,
			Options: []models.CourseTestQuestionOptionRequest{
				{Text: "Go", Correct: true},
				{Text: "Rust"},
			},
		},
		{
			Prompt: "Pick all that apply",
			Type:   models.CourseTestQuestionTypeMultipleChoice,
			Options: []models.CourseTestQuestionOptionRequest{
				{Text: "Go", Correct: true},
				{Text: "Rust", Correct: true},
				{Text: "PHP"},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(questions) != 3 {
		t.Fatalf("expected 3 questions, got %d", len(questions))
	}

	if questions[0].Position != 0 || questions[1].Position != 1 {
		t.Fatalf("unexpected positions: %#v", questions)
	}
	if questions[1].Type != models.CourseTestQuestionTypeSingleChoice {
		t.Errorf("unexpected question type: %s", questions[1].Type)
	}
	if len(questions[1].Options) != 2 {
		t.Fatalf("expected options to be built, got %d", len(questions[1].Options))
	}
}

func TestTestServiceBuildQuestionModelsValidation(t *testing.T) {
	svc := &TestService{}

	_, err := svc.buildQuestionModels([]models.CourseTestQuestionRequest{{
		Prompt: "",
		Type:   models.CourseTestQuestionTypeText,
	}})
	if err == nil {
		t.Fatalf("expected error for empty prompt")
	}

	_, err = svc.buildQuestionModels([]models.CourseTestQuestionRequest{{
		Prompt:     "Question",
		Type:       models.CourseTestQuestionTypeText,
		AnswerText: "",
	}})
	if err == nil {
		t.Fatalf("expected error for missing text answer")
	}

	_, err = svc.buildQuestionModels([]models.CourseTestQuestionRequest{{
		Prompt: "Question",
		Type:   models.CourseTestQuestionTypeSingleChoice,
		Options: []models.CourseTestQuestionOptionRequest{
			{Text: "A", Correct: true},
			{Text: "B", Correct: true},
		},
	}})
	if err == nil {
		t.Fatalf("expected error for multiple correct options in single choice question")
	}
}

func TestTestServiceSubmit(t *testing.T) {
	repo := &mockCourseTestRepository{}
	svc := &TestService{testRepo: repo}

	test := &models.CourseTest{ID: 1, Title: "Sample"}
	repo.test = test

	questions := []models.CourseTestQuestion{
		{
			ID:          101,
			TestID:      test.ID,
			Prompt:      "Name the language",
			Type:        models.CourseTestQuestionTypeText,
			AnswerText:  "Go",
			Explanation: "Go is correct",
		},
		{
			ID:     102,
			TestID: test.ID,
			Prompt: "Select one",
			Type:   models.CourseTestQuestionTypeSingleChoice,
			Options: []models.CourseTestQuestionOption{
				{ID: 201, QuestionID: 102, Text: "Go", Correct: true},
				{ID: 202, QuestionID: 102, Text: "Rust", Correct: false},
			},
		},
		{
			ID:     103,
			TestID: test.ID,
			Prompt: "Select two",
			Type:   models.CourseTestQuestionTypeMultipleChoice,
			Options: []models.CourseTestQuestionOption{
				{ID: 301, QuestionID: 103, Text: "Go", Correct: true},
				{ID: 302, QuestionID: 103, Text: "Rust", Correct: true},
				{ID: 303, QuestionID: 103, Text: "PHP", Correct: false},
			},
		},
	}

	repo.structures = map[uint][]models.CourseTestQuestion{
		test.ID: questions,
	}

	result, err := svc.Submit(test.ID, 5, models.SubmitCourseTestRequest{
		Answers: []models.CourseTestAnswerSubmission{
			{QuestionID: 101, Text: "go"},
			{QuestionID: 102, OptionIDs: []uint{201}},
			{QuestionID: 103, OptionIDs: []uint{301, 302}},
		},
	})
	if err != nil {
		t.Fatalf("expected no error submitting test, got %v", err)
	}

	if result.Score != 3 || result.MaxScore != 3 {
		t.Fatalf("unexpected scoring: %+v", result)
	}

	if len(result.Answers) != len(questions) {
		t.Fatalf("expected %d answer results, got %d", len(questions), len(result.Answers))
	}

	if len(repo.saved) != 1 {
		t.Fatalf("expected saved result, got %d", len(repo.saved))
	}

	if result.Record == nil {
		t.Fatalf("expected record to be returned")
	}
	if result.Record.Score != 3 || result.Record.MaxScore != 3 {
		t.Fatalf("unexpected record: %+v", result.Record)
	}
	if result.Record.Attempts != 1 {
		t.Fatalf("expected record attempts to equal 1, got %d", result.Record.Attempts)
	}

	var stored []struct {
		QuestionID uint   `json:"question_id"`
		Correct    bool   `json:"correct"`
		Text       string `json:"text,omitempty"`
		OptionIDs  []uint `json:"option_ids,omitempty"`
	}
	if err := json.Unmarshal(repo.saved[0].Answers, &stored); err != nil {
		t.Fatalf("failed to unmarshal stored answers: %v", err)
	}

	if len(stored) != len(questions) {
		t.Fatalf("expected stored answers for each question, got %d", len(stored))
	}

	for i, item := range stored {
		if !item.Correct {
			t.Fatalf("expected stored answer %d to be correct", i)
		}
		if item.QuestionID == 101 && item.Text != "go" {
			t.Fatalf("expected stored text to be preserved, got %q", item.Text)
		}
	}
}

var _ repository.CourseTestRepository = (*mockCourseTestRepository)(nil)
