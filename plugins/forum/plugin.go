package forum

import (
	"fmt"

	"constructor-script-backend/internal/plugin/host"
	"constructor-script-backend/internal/plugin/registry"
	pluginruntime "constructor-script-backend/internal/plugin/runtime"
	forumapi "constructor-script-backend/plugins/forum/api"
	forumhandlers "constructor-script-backend/plugins/forum/handlers"
	forumservice "constructor-script-backend/plugins/forum/service"
)

func init() {
	registry.Register("forum", NewFeature)
}

type Feature struct {
	host host.Host
}

func NewFeature(h host.Host) (pluginruntime.Feature, error) {
	if h == nil {
		return nil, fmt.Errorf("host is required")
	}
	return &Feature{host: h}, nil
}

func (f *Feature) Activate() error {
	if f == nil || f.host == nil {
		return fmt.Errorf("feature host is not configured")
	}

	repos := f.host.Repositories()
	if repos == nil {
		return fmt.Errorf("repository access is not available")
	}

	services := f.host.Services(forumapi.Namespace)

	var questionSvc *forumservice.QuestionService
	var categorySvc *forumservice.CategoryService
	if value, ok := services.Get(forumapi.ServiceQuestion).(*forumservice.QuestionService); ok {
		questionSvc = value
	}
	if questionSvc == nil {
		questionSvc = forumservice.NewQuestionService(repos.ForumQuestion(), repos.ForumCategory(), repos.ForumQuestionVote())
		services.Set(forumapi.ServiceQuestion, questionSvc)
	} else {
		questionSvc.SetRepositories(repos.ForumQuestion(), repos.ForumCategory(), repos.ForumQuestionVote())
	}

	if value, ok := services.Get(forumapi.ServiceCategory).(*forumservice.CategoryService); ok {
		categorySvc = value
	}
	if categorySvc == nil {
		categorySvc = forumservice.NewCategoryService(repos.ForumCategory())
		services.Set(forumapi.ServiceCategory, categorySvc)
	} else {
		categorySvc.SetRepository(repos.ForumCategory())
	}

	var answerSvc *forumservice.AnswerService
	if value, ok := services.Get(forumapi.ServiceAnswer).(*forumservice.AnswerService); ok {
		answerSvc = value
	}
	if answerSvc == nil {
		answerSvc = forumservice.NewAnswerService(repos.ForumAnswer(), repos.ForumQuestion(), repos.ForumAnswerVote())
		services.Set(forumapi.ServiceAnswer, answerSvc)
	} else {
		answerSvc.SetRepositories(repos.ForumAnswer(), repos.ForumQuestion(), repos.ForumAnswerVote())
	}

	handlers := f.host.Handlers(forumapi.Namespace)

	var questionHandler *forumhandlers.QuestionHandler
	if value, ok := handlers.Get(forumapi.HandlerQuestion).(*forumhandlers.QuestionHandler); ok {
		questionHandler = value
	}
	if questionHandler == nil {
		questionHandler = forumhandlers.NewQuestionHandler(questionSvc)
		handlers.Set(forumapi.HandlerQuestion, questionHandler)
	} else {
		questionHandler.SetService(questionSvc)
	}

	var categoryHandler *forumhandlers.CategoryHandler
	if value, ok := handlers.Get(forumapi.HandlerCategory).(*forumhandlers.CategoryHandler); ok {
		categoryHandler = value
	}
	if categoryHandler == nil {
		categoryHandler = forumhandlers.NewCategoryHandler(categorySvc)
		handlers.Set(forumapi.HandlerCategory, categoryHandler)
	} else {
		categoryHandler.SetService(categorySvc)
	}

	var answerHandler *forumhandlers.AnswerHandler
	if value, ok := handlers.Get(forumapi.HandlerAnswer).(*forumhandlers.AnswerHandler); ok {
		answerHandler = value
	}
	if answerHandler == nil {
		answerHandler = forumhandlers.NewAnswerHandler(answerSvc)
		handlers.Set(forumapi.HandlerAnswer, answerHandler)
	} else {
		answerHandler.SetService(answerSvc)
	}

	if templateHandler := f.host.TemplateHandler(); templateHandler != nil {
		templateHandler.SetForumServices(questionSvc, answerSvc, categorySvc)
	}

	return nil
}

func (f *Feature) Deactivate() error {
	if f == nil || f.host == nil {
		return nil
	}

	handlers := f.host.Handlers(forumapi.Namespace)
	if questionHandler, _ := handlers.Get(forumapi.HandlerQuestion).(*forumhandlers.QuestionHandler); questionHandler != nil {
		questionHandler.SetService(nil)
	}
	if categoryHandler, _ := handlers.Get(forumapi.HandlerCategory).(*forumhandlers.CategoryHandler); categoryHandler != nil {
		categoryHandler.SetService(nil)
	}
	if answerHandler, _ := handlers.Get(forumapi.HandlerAnswer).(*forumhandlers.AnswerHandler); answerHandler != nil {
		answerHandler.SetService(nil)
	}

	services := f.host.Services(forumapi.Namespace)
	services.Set(forumapi.ServiceQuestion, nil)
	services.Set(forumapi.ServiceCategory, nil)
	services.Set(forumapi.ServiceAnswer, nil)

	if templateHandler := f.host.TemplateHandler(); templateHandler != nil {
		templateHandler.SetForumServices(nil, nil, nil)
	}

	return nil
}
