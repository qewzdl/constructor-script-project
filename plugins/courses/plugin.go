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
	courseServices := f.host.CourseServices()
	courseHandlers := f.host.CourseHandlers()

	videoRepo := repos.CourseVideo()
	topicRepo := repos.CourseTopic()
	packageRepo := repos.CoursePackage()
	testRepo := repos.CourseTest()

	if videoRepo == nil || topicRepo == nil || packageRepo == nil || testRepo == nil {
		return fmt.Errorf("course repositories are not configured")
	}

	uploadService := coreServices.Upload()
	if uploadService == nil {
		return fmt.Errorf("upload service is not configured")
	}

	videoService := courseServices.Video()
	if videoService == nil {
		videoService = courseservice.NewVideoService(videoRepo, uploadService)
		courseServices.SetVideo(videoService)
	} else {
		videoService.SetUploadService(uploadService)
	}

	testService := courseServices.Test()
	if testService == nil {
		testService = courseservice.NewTestService(testRepo)
		courseServices.SetTest(testService)
	} else {
		testService.SetRepository(testRepo)
	}

	topicService := courseServices.Topic()
	if topicService == nil {
		topicService = courseservice.NewTopicService(topicRepo, videoRepo, testRepo)
		courseServices.SetTopic(topicService)
	} else {
		topicService.SetRepositories(topicRepo, videoRepo, testRepo)
	}

	packageService := courseServices.Package()
	if packageService == nil {
		packageService = courseservice.NewPackageService(packageRepo, topicRepo, videoRepo, testRepo)
		courseServices.SetPackage(packageService)
	} else {
		packageService.SetRepositories(packageRepo, topicRepo, videoRepo, testRepo)
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

	checkoutService := courseServices.Checkout()
	if checkoutService == nil {
		checkoutService = courseservice.NewCheckoutService(packageRepo, checkoutProvider, checkoutConfig)
		courseServices.SetCheckout(checkoutService)
	} else {
		checkoutService.SetDependencies(packageRepo, checkoutProvider)
		checkoutService.SetConfig(checkoutConfig)
	}

	if courseHandlers != nil {
		if handler := courseHandlers.Video(); handler == nil {
			courseHandlers.SetVideo(coursehandlers.NewVideoHandler(videoService))
		} else {
			handler.SetService(videoService)
		}

		if handler := courseHandlers.Topic(); handler == nil {
			courseHandlers.SetTopic(coursehandlers.NewTopicHandler(topicService))
		} else {
			handler.SetService(topicService)
		}

		if handler := courseHandlers.Test(); handler == nil {
			courseHandlers.SetTest(coursehandlers.NewTestHandler(testService))
		} else {
			handler.SetService(testService)
		}

		if handler := courseHandlers.Package(); handler == nil {
			courseHandlers.SetPackage(coursehandlers.NewPackageHandler(packageService))
		} else {
			handler.SetService(packageService)
		}
		if handler := courseHandlers.Checkout(); handler == nil {
			courseHandlers.SetCheckout(coursehandlers.NewCheckoutHandler(checkoutService))
		} else {
			handler.SetService(checkoutService)
		}
	}

	if templateHandler := f.host.TemplateHandler(); templateHandler != nil {
		templateHandler.SetCoursePackageService(packageService)
		templateHandler.SetCourseCheckoutService(checkoutService)
	}

	return nil
}

func (f *Feature) Deactivate() error {
	if f == nil || f.host == nil {
		return nil
	}

	courseHandlers := f.host.CourseHandlers()
	if courseHandlers != nil {
		if handler := courseHandlers.Video(); handler != nil {
			handler.SetService(nil)
		}
		if handler := courseHandlers.Topic(); handler != nil {
			handler.SetService(nil)
		}
		if handler := courseHandlers.Package(); handler != nil {
			handler.SetService(nil)
		}
		if handler := courseHandlers.Checkout(); handler != nil {
			handler.SetService(nil)
		}
	}

	courseServices := f.host.CourseServices()
	if courseServices != nil {
		courseServices.SetVideo(nil)
		courseServices.SetTopic(nil)
		courseServices.SetPackage(nil)
		courseServices.SetCheckout(nil)
	}

	if templateHandler := f.host.TemplateHandler(); templateHandler != nil {
		templateHandler.SetCoursePackageService(nil)
		templateHandler.SetCourseCheckoutService(nil)
	}

	return nil
}
