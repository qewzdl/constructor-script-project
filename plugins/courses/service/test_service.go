package service

import (
	"encoding/json"
	"errors"
	"strings"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
)

type TestService struct {
	testRepo repository.CourseTestRepository
}

func NewTestService(testRepo repository.CourseTestRepository) *TestService {
	return &TestService{testRepo: testRepo}
}

func (s *TestService) SetRepository(testRepo repository.CourseTestRepository) {
	if s == nil {
		return
	}
	s.testRepo = testRepo
}

func (s *TestService) Create(req models.CreateCourseTestRequest) (*models.CourseTest, error) {
	if s == nil || s.testRepo == nil {
		return nil, errors.New("course test repository is not configured")
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, newValidationError("test title is required")
	}

	test := models.CourseTest{
		Title:       title,
		Description: strings.TrimSpace(req.Description),
	}

	if err := s.testRepo.Create(&test); err != nil {
		return nil, err
	}

	if err := s.replaceStructure(test.ID, req.Questions); err != nil {
		return nil, err
	}

	return s.GetByID(test.ID)
}

func (s *TestService) Update(id uint, req models.UpdateCourseTestRequest) (*models.CourseTest, error) {
	if s == nil || s.testRepo == nil {
		return nil, errors.New("course test repository is not configured")
	}

	test, err := s.testRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, newValidationError("test title is required")
	}

	test.Title = title
	test.Description = strings.TrimSpace(req.Description)

	if err := s.testRepo.Update(test); err != nil {
		return nil, err
	}

	if err := s.replaceStructure(test.ID, req.Questions); err != nil {
		return nil, err
	}

	return s.GetByID(test.ID)
}

func (s *TestService) Delete(id uint) error {
	if s == nil || s.testRepo == nil {
		return errors.New("course test repository is not configured")
	}
	return s.testRepo.Delete(id)
}

func (s *TestService) GetByID(id uint) (*models.CourseTest, error) {
	if s == nil || s.testRepo == nil {
		return nil, errors.New("course test repository is not configured")
	}

	test, err := s.testRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if err := s.populateQuestions([]*models.CourseTest{test}); err != nil {
		return nil, err
	}

	return test, nil
}

func (s *TestService) List() ([]models.CourseTest, error) {
	if s == nil || s.testRepo == nil {
		return nil, errors.New("course test repository is not configured")
	}

	tests, err := s.testRepo.List()
	if err != nil {
		return nil, err
	}

	if len(tests) == 0 {
		return tests, nil
	}

	ptrs := make([]*models.CourseTest, 0, len(tests))
	for i := range tests {
		ptrs = append(ptrs, &tests[i])
	}

	if err := s.populateQuestions(ptrs); err != nil {
		return nil, err
	}

	return tests, nil
}

func (s *TestService) Exists(id uint) (bool, error) {
	if s == nil || s.testRepo == nil {
		return false, errors.New("course test repository is not configured")
	}
	return s.testRepo.Exists(id)
}

func (s *TestService) Submit(testID uint, userID uint, req models.SubmitCourseTestRequest) (*models.CourseTestSubmissionResult, error) {
	if s == nil || s.testRepo == nil {
		return nil, errors.New("course test repository is not configured")
	}
	if userID == 0 {
		return nil, errors.New("user id is required")
	}

	test, err := s.GetByID(testID)
	if err != nil {
		return nil, err
	}

	answerMap := make(map[uint]models.CourseTestAnswerSubmission, len(req.Answers))
	for _, answer := range req.Answers {
		answerMap[answer.QuestionID] = answer
	}

	score := 0
	maxScore := len(test.Questions)
	results := make([]models.CourseTestAnswerResult, 0, len(test.Questions))
	stored := make([]courseTestStoredAnswer, 0, len(test.Questions))

	for _, question := range test.Questions {
		submission, ok := answerMap[question.ID]
		evaluation := s.evaluateAnswer(question, submission, ok)
		if evaluation.Correct {
			score++
		}
		results = append(results, models.CourseTestAnswerResult{
			QuestionID:  question.ID,
			Correct:     evaluation.Correct,
			Explanation: question.Explanation,
		})
		stored = append(stored, evaluation.Stored)
	}

	payload, err := json.Marshal(stored)
	if err != nil {
		return nil, err
	}

	record := models.CourseTestResult{
		TestID:   test.ID,
		UserID:   userID,
		Score:    score,
		MaxScore: maxScore,
		Answers:  payload,
	}

	if err := s.testRepo.SaveResult(&record); err != nil {
		return nil, err
	}

	best, attempts, err := s.testRepo.GetBestResult(test.ID, userID)
	if err != nil {
		return nil, err
	}

	var submissionRecord *models.CourseTestRecord
	if best != nil {
		submissionRecord = &models.CourseTestRecord{
			Score:    best.Score,
			MaxScore: best.MaxScore,
			Attempts: int(attempts),
		}
		if !best.CreatedAt.IsZero() {
			achievedAt := best.CreatedAt.UTC()
			submissionRecord.AchievedAt = &achievedAt
		}
	}

	return &models.CourseTestSubmissionResult{
		Score:    score,
		MaxScore: maxScore,
		Answers:  results,
		Record:   submissionRecord,
	}, nil
}

type courseTestStoredAnswer struct {
	QuestionID uint   `json:"question_id"`
	Text       string `json:"text,omitempty"`
	OptionIDs  []uint `json:"option_ids,omitempty"`
	Correct    bool   `json:"correct"`
}

type answerEvaluation struct {
	Correct bool
	Stored  courseTestStoredAnswer
}

func (s *TestService) evaluateAnswer(question models.CourseTestQuestion, submission models.CourseTestAnswerSubmission, provided bool) answerEvaluation {
	result := answerEvaluation{
		Stored: courseTestStoredAnswer{
			QuestionID: question.ID,
			OptionIDs:  append([]uint{}, submission.OptionIDs...),
			Text:       strings.TrimSpace(submission.Text),
		},
	}

	switch question.Type {
	case models.CourseTestQuestionTypeText:
		expected := strings.TrimSpace(strings.ToLower(question.AnswerText))
		if expected == "" {
			break
		}
		providedText := strings.TrimSpace(strings.ToLower(submission.Text))
		if providedText != "" {
			result.Correct = providedText == expected
		}
	case models.CourseTestQuestionTypeSingleChoice:
		if len(question.Options) == 0 || !provided {
			break
		}
		correctID := uint(0)
		optionSet := make(map[uint]struct{}, len(question.Options))
		for _, option := range question.Options {
			optionSet[option.ID] = struct{}{}
			if option.Correct {
				correctID = option.ID
			}
		}
		if correctID == 0 {
			break
		}
		if len(submission.OptionIDs) != 1 {
			break
		}
		choice := submission.OptionIDs[0]
		if _, exists := optionSet[choice]; !exists {
			break
		}
		if choice == correctID {
			result.Correct = true
		}
	case models.CourseTestQuestionTypeMultipleChoice:
		if len(question.Options) == 0 || !provided {
			break
		}
		correctSet := make(map[uint]struct{})
		optionSet := make(map[uint]struct{}, len(question.Options))
		for _, option := range question.Options {
			optionSet[option.ID] = struct{}{}
			if option.Correct {
				correctSet[option.ID] = struct{}{}
			}
		}
		if len(correctSet) == 0 {
			break
		}
		submitted := make(map[uint]struct{}, len(submission.OptionIDs))
		for _, id := range submission.OptionIDs {
			if _, exists := optionSet[id]; exists {
				submitted[id] = struct{}{}
			}
		}
		if len(submitted) != len(correctSet) {
			break
		}
		matched := true
		for id := range correctSet {
			if _, exists := submitted[id]; !exists {
				matched = false
				break
			}
		}
		result.Correct = matched
	}

	result.Stored.Correct = result.Correct
	return result
}

func (s *TestService) populateQuestions(tests []*models.CourseTest) error {
	if len(tests) == 0 {
		return nil
	}
	if s.testRepo == nil {
		return errors.New("course test repository is not configured")
	}

	ids := make([]uint, 0, len(tests))
	testMap := make(map[uint]*models.CourseTest, len(tests))
	for _, test := range tests {
		if test == nil {
			continue
		}
		test.Questions = []models.CourseTestQuestion{}
		ids = append(ids, test.ID)
		testMap[test.ID] = test
	}

	structures, err := s.testRepo.ListStructure(ids)
	if err != nil {
		return err
	}

	for id, questions := range structures {
		if test, ok := testMap[id]; ok {
			test.Questions = questions
		}
	}

	return nil
}

func (s *TestService) replaceStructure(testID uint, questions []models.CourseTestQuestionRequest) error {
	if s == nil || s.testRepo == nil {
		return errors.New("course test repository is not configured")
	}
	modelsQuestions, err := s.buildQuestionModels(questions)
	if err != nil {
		return err
	}
	return s.testRepo.ReplaceStructure(testID, modelsQuestions)
}

func (s *TestService) buildQuestionModels(questions []models.CourseTestQuestionRequest) ([]models.CourseTestQuestion, error) {
	if len(questions) == 0 {
		return []models.CourseTestQuestion{}, nil
	}

	result := make([]models.CourseTestQuestion, 0, len(questions))
	for idx, question := range questions {
		qType := strings.ToLower(strings.TrimSpace(question.Type))
		if qType == "" {
			return nil, newValidationError("question %d type is required", idx+1)
		}

		prompt := strings.TrimSpace(question.Prompt)
		if prompt == "" {
			return nil, newValidationError("question %d prompt is required", idx+1)
		}

		modelQuestion := models.CourseTestQuestion{
			Prompt:      prompt,
			Type:        qType,
			Explanation: strings.TrimSpace(question.Explanation),
			AnswerText:  strings.TrimSpace(question.AnswerText),
			Options:     []models.CourseTestQuestionOption{},
			Position:    idx,
		}

		switch qType {
		case models.CourseTestQuestionTypeText:
			if modelQuestion.AnswerText == "" {
				return nil, newValidationError("question %d answer is required", idx+1)
			}
		case models.CourseTestQuestionTypeSingleChoice:
			options, err := s.buildOptions(question.Options, true, idx)
			if err != nil {
				return nil, err
			}
			modelQuestion.Options = options
		case models.CourseTestQuestionTypeMultipleChoice:
			options, err := s.buildOptions(question.Options, false, idx)
			if err != nil {
				return nil, err
			}
			modelQuestion.Options = options
		default:
			return nil, newValidationError("question %d has unsupported type: %s", idx+1, question.Type)
		}

		result = append(result, modelQuestion)
	}

	return result, nil
}

func (s *TestService) buildOptions(options []models.CourseTestQuestionOptionRequest, single bool, questionIndex int) ([]models.CourseTestQuestionOption, error) {
	if len(options) == 0 {
		return nil, newValidationError("question %d must include options", questionIndex+1)
	}

	result := make([]models.CourseTestQuestionOption, 0, len(options))
	correctCount := 0
	for idx, option := range options {
		text := strings.TrimSpace(option.Text)
		if text == "" {
			return nil, newValidationError("question %d option %d text is required", questionIndex+1, idx+1)
		}
		modelOption := models.CourseTestQuestionOption{
			Text:     text,
			Correct:  option.Correct,
			Position: idx,
		}
		if modelOption.Correct {
			correctCount++
		}
		result = append(result, modelOption)
	}

	if correctCount == 0 {
		return nil, newValidationError("question %d must have at least one correct option", questionIndex+1)
	}
	if single && correctCount != 1 {
		return nil, newValidationError("question %d must have exactly one correct option", questionIndex+1)
	}

	return result, nil
}
