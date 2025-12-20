package courses

import (
	"fmt"
	"strings"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/payments/stripe"
	"constructor-script-backend/internal/plugin/host"
	"constructor-script-backend/internal/plugin/registry"
	pluginruntime "constructor-script-backend/internal/plugin/runtime"
	"constructor-script-backend/pkg/logger"
	courseapi "constructor-script-backend/plugins/courses/api"
	coursehandlers "constructor-script-backend/plugins/courses/handlers"
	courseservice "constructor-script-backend/plugins/courses/service"
)

func init() {
	registry.Register("courses", NewFeature)
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
		return fmt.Errorf("repository access is not configured")
	}

	coreServices := f.host.CoreServices()
	services := f.host.Services(courseapi.Namespace)
	handlers := f.host.Handlers(courseapi.Namespace)

	videoRepo := repos.CourseVideo()
	contentRepo := repos.CourseContent()
	topicRepo := repos.CourseTopic()
	packageRepo := repos.CoursePackage()
	accessRepo := repos.CoursePackageAccess()
	userRepo := repos.User()
	testRepo := repos.CourseTest()

	if videoRepo == nil || contentRepo == nil || topicRepo == nil || packageRepo == nil || accessRepo == nil || userRepo == nil || testRepo == nil {
		return fmt.Errorf("course repositories are not configured")
	}

	uploadService := coreServices.Upload()
	if uploadService == nil {
		return fmt.Errorf("upload service is not configured")
	}

	var videoService *courseservice.VideoService
	if value, ok := services.Get(courseapi.ServiceVideo).(*courseservice.VideoService); ok {
		videoService = value
	}
	if videoService == nil {
		videoService = courseservice.NewVideoService(videoRepo, uploadService, f.host.ThemeManager())
		services.Set(courseapi.ServiceVideo, videoService)
	} else {
		videoService.SetUploadService(uploadService)
		videoService.SetThemeManager(f.host.ThemeManager())
	}

	var testService *courseservice.TestService
	if value, ok := services.Get(courseapi.ServiceTest).(*courseservice.TestService); ok {
		testService = value
	}
	if testService == nil {
		testService = courseservice.NewTestService(testRepo)
		services.Set(courseapi.ServiceTest, testService)
	} else {
		testService.SetRepository(testRepo)
	}

	var contentService *courseservice.ContentService
	if value, ok := services.Get(courseapi.ServiceContent).(*courseservice.ContentService); ok {
		contentService = value
	}
	if contentService == nil {
		contentService = courseservice.NewContentService(contentRepo, f.host.ThemeManager())
		services.Set(courseapi.ServiceContent, contentService)
	} else {
		contentService.SetThemeManager(f.host.ThemeManager())
	}

	var topicService *courseservice.TopicService
	if value, ok := services.Get(courseapi.ServiceTopic).(*courseservice.TopicService); ok {
		topicService = value
	}
	if topicService == nil {
		topicService = courseservice.NewTopicService(topicRepo, videoRepo, testRepo, contentRepo)
		services.Set(courseapi.ServiceTopic, topicService)
	} else {
		topicService.SetRepositories(topicRepo, videoRepo, testRepo, contentRepo)
	}

	var packageService *courseservice.PackageService
	if value, ok := services.Get(courseapi.ServicePackage).(*courseservice.PackageService); ok {
		packageService = value
	}
	if packageService == nil {
		packageService = courseservice.NewPackageService(packageRepo, topicRepo, videoRepo, testRepo, contentRepo, accessRepo, userRepo)
		services.Set(courseapi.ServicePackage, packageService)
	} else {
		packageService.SetRepositories(packageRepo, topicRepo, videoRepo, testRepo, contentRepo, accessRepo, userRepo)
	}

	cfg := f.host.Config()
	checkoutConfig := courseservice.CheckoutConfig{}
	var (
		checkoutProvider *stripe.Provider
		stripeSecret     string
		stripePublish    string
		stripeWebhook    string
	)
	if cfg != nil {
		checkoutConfig = courseservice.CheckoutConfig{
			SuccessURL: cfg.CourseCheckoutSuccessURL,
			CancelURL:  cfg.CourseCheckoutCancelURL,
			Currency:   cfg.CourseCheckoutCurrency,
		}
		stripeSecret = strings.TrimSpace(cfg.StripeSecretKey)
		stripePublish = strings.TrimSpace(cfg.StripePublishableKey)
		stripeWebhook = strings.TrimSpace(cfg.StripeWebhookSecret)
	} else {
		logger.Debug("Configuration unavailable; course checkout remains disabled", map[string]interface{}{"feature": "courses"})
	}

	if setupService := coreServices.Setup(); setupService != nil {
		defaults := models.SiteSettings{
			StripeSecretKey:          stripeSecret,
			StripePublishableKey:     stripePublish,
			StripeWebhookSecret:      stripeWebhook,
			CourseCheckoutSuccessURL: checkoutConfig.SuccessURL,
			CourseCheckoutCancelURL:  checkoutConfig.CancelURL,
			CourseCheckoutCurrency:   checkoutConfig.Currency,
		}
		if settings, err := setupService.GetSiteSettings(defaults); err != nil {
			logger.Error(err, "Failed to load site settings for checkout", map[string]interface{}{"feature": "courses"})
		} else {
			if key := strings.TrimSpace(settings.StripeSecretKey); key != "" {
				stripeSecret = key
			}
			if key := strings.TrimSpace(settings.StripePublishableKey); key != "" {
				stripePublish = key
			}
			if key := strings.TrimSpace(settings.StripeWebhookSecret); key != "" {
				stripeWebhook = key
			}
			if url := strings.TrimSpace(settings.CourseCheckoutSuccessURL); url != "" {
				checkoutConfig.SuccessURL = url
			}
			if url := strings.TrimSpace(settings.CourseCheckoutCancelURL); url != "" {
				checkoutConfig.CancelURL = url
			}
			if currency := strings.TrimSpace(settings.CourseCheckoutCurrency); currency != "" {
				checkoutConfig.Currency = strings.ToLower(currency)
			}
		}
	}

	if stripeSecret != "" && !stripe.IsSecretKey(stripeSecret) {
		logger.Warn("Invalid Stripe secret key provided; course checkout disabled", map[string]interface{}{"feature": "courses"})
		stripeSecret = ""
	}
	if stripePublish != "" && !stripe.IsPublishableKey(stripePublish) {
		logger.Warn("Invalid Stripe publishable key provided; ignoring value", map[string]interface{}{"feature": "courses"})
		stripePublish = ""
	}
	if stripeWebhook != "" && !stripe.IsWebhookSecret(stripeWebhook) {
		logger.Warn("Invalid Stripe webhook secret provided; ignoring value", map[string]interface{}{"feature": "courses"})
		stripeWebhook = ""
	}

	if stripeSecret != "" {
		provider, err := stripe.NewProvider(stripeSecret)
		if err != nil {
			logger.Error(err, "Failed to initialise Stripe provider", map[string]interface{}{"feature": "courses"})
		} else {
			checkoutProvider = provider
		}
	} else {
		logger.Debug("Stripe secret key not provided; course checkout remains disabled", map[string]interface{}{"feature": "courses"})
	}

	var checkoutService *courseservice.CheckoutService
	if value, ok := services.Get(courseapi.ServiceCheckout).(*courseservice.CheckoutService); ok {
		checkoutService = value
	}
	if checkoutService == nil {
		checkoutService = courseservice.NewCheckoutService(packageRepo, checkoutProvider, checkoutConfig)
		services.Set(courseapi.ServiceCheckout, checkoutService)
	} else {
		checkoutService.SetDependencies(packageRepo, checkoutProvider)
		checkoutService.SetConfig(checkoutConfig)
	}

	if handler, ok := handlers.Get(courseapi.HandlerVideo).(*coursehandlers.VideoHandler); handler == nil || !ok {
		handlers.Set(courseapi.HandlerVideo, coursehandlers.NewVideoHandler(videoService))
	} else {
		handler.SetService(videoService)
	}

	if handler, ok := handlers.Get(courseapi.HandlerContent).(*coursehandlers.ContentHandler); handler == nil || !ok {
		handlers.Set(courseapi.HandlerContent, coursehandlers.NewContentHandler(contentService))
	} else {
		handler.SetService(contentService)
	}

	if handler, ok := handlers.Get(courseapi.HandlerTopic).(*coursehandlers.TopicHandler); handler == nil || !ok {
		handlers.Set(courseapi.HandlerTopic, coursehandlers.NewTopicHandler(topicService))
	} else {
		handler.SetService(topicService)
	}

	if handler, ok := handlers.Get(courseapi.HandlerTest).(*coursehandlers.TestHandler); handler == nil || !ok {
		handlers.Set(courseapi.HandlerTest, coursehandlers.NewTestHandler(testService))
	} else {
		handler.SetService(testService)
	}

	if handler, ok := handlers.Get(courseapi.HandlerPackage).(*coursehandlers.PackageHandler); handler == nil || !ok {
		handlers.Set(courseapi.HandlerPackage, coursehandlers.NewPackageHandler(packageService))
	} else {
		handler.SetService(packageService)
	}

	if handler, ok := handlers.Get(courseapi.HandlerCheckout).(*coursehandlers.CheckoutHandler); handler == nil || !ok {
		handler = coursehandlers.NewCheckoutHandler(checkoutService)
		handler.SetPackageService(packageService)
		handler.SetWebhookSecret(stripeWebhook)
		handlers.Set(courseapi.HandlerCheckout, handler)
	} else {
		handler.SetService(checkoutService)
		handler.SetPackageService(packageService)
		handler.SetWebhookSecret(stripeWebhook)
	}

	if templateHandler := f.host.TemplateHandler(); templateHandler != nil {
		templateHandler.SetCoursePackageService(packageService)
		templateHandler.SetCourseCheckoutService(checkoutService)
	}

	if authHandler := f.host.AuthHandler(); authHandler != nil {
		authHandler.SetCoursePackageService(packageService)
	}

	return nil
}

func (f *Feature) Deactivate() error {
	if f == nil || f.host == nil {
		return nil
	}

	handlers := f.host.Handlers(courseapi.Namespace)
	if handler, _ := handlers.Get(courseapi.HandlerVideo).(*coursehandlers.VideoHandler); handler != nil {
		handler.SetService(nil)
	}
	if handler, _ := handlers.Get(courseapi.HandlerTopic).(*coursehandlers.TopicHandler); handler != nil {
		handler.SetService(nil)
	}
	if handler, _ := handlers.Get(courseapi.HandlerPackage).(*coursehandlers.PackageHandler); handler != nil {
		handler.SetService(nil)
	}
	if handler, _ := handlers.Get(courseapi.HandlerTest).(*coursehandlers.TestHandler); handler != nil {
		handler.SetService(nil)
	}
	if handler, _ := handlers.Get(courseapi.HandlerCheckout).(*coursehandlers.CheckoutHandler); handler != nil {
		handler.SetService(nil)
		handler.SetPackageService(nil)
		handler.SetWebhookSecret("")
	}

	services := f.host.Services(courseapi.Namespace)
	services.Set(courseapi.ServiceVideo, nil)
	services.Set(courseapi.ServiceTopic, nil)
	services.Set(courseapi.ServicePackage, nil)
	services.Set(courseapi.ServiceCheckout, nil)
	services.Set(courseapi.ServiceTest, nil)

	if templateHandler := f.host.TemplateHandler(); templateHandler != nil {
		templateHandler.SetCoursePackageService(nil)
		templateHandler.SetCourseCheckoutService(nil)
	}

	if authHandler := f.host.AuthHandler(); authHandler != nil {
		authHandler.SetCoursePackageService(nil)
	}

	return nil
}
