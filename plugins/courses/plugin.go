package courses

import (
	"fmt"

	"constructor-script-backend/internal/plugin/host"
	"constructor-script-backend/internal/plugin/registry"
	pluginruntime "constructor-script-backend/internal/plugin/runtime"
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

	if videoRepo == nil || topicRepo == nil || packageRepo == nil {
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

	topicService := courseServices.Topic()
	if topicService == nil {
		topicService = courseservice.NewTopicService(topicRepo, videoRepo)
		courseServices.SetTopic(topicService)
	} else {
		topicService.SetRepositories(topicRepo, videoRepo)
	}

	packageService := courseServices.Package()
	if packageService == nil {
		packageService = courseservice.NewPackageService(packageRepo, topicRepo, videoRepo)
		courseServices.SetPackage(packageService)
	} else {
		packageService.SetRepositories(packageRepo, topicRepo, videoRepo)
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

		if handler := courseHandlers.Package(); handler == nil {
			courseHandlers.SetPackage(coursehandlers.NewPackageHandler(packageService))
		} else {
			handler.SetService(packageService)
		}
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
	}

	courseServices := f.host.CourseServices()
	if courseServices != nil {
		courseServices.SetVideo(nil)
		courseServices.SetTopic(nil)
		courseServices.SetPackage(nil)
	}

	return nil
}
