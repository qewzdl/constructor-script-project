package archive

import (
	"fmt"

	"constructor-script-backend/internal/plugin/host"
	"constructor-script-backend/internal/plugin/registry"
	pluginruntime "constructor-script-backend/internal/plugin/runtime"
	archiveapi "constructor-script-backend/plugins/archive/api"
	archivehandlers "constructor-script-backend/plugins/archive/handlers"
	archiveseed "constructor-script-backend/plugins/archive/seed"
	archiveservice "constructor-script-backend/plugins/archive/service"
)

func init() {
	registry.Register("archive", NewFeature)
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

	servicesRegistry := f.host.Services(archiveapi.Namespace)
	handlersRegistry := f.host.Handlers(archiveapi.Namespace)

	var directoryService *archiveservice.DirectoryService
	if existing, ok := servicesRegistry.Get(archiveapi.ServiceDirectory).(*archiveservice.DirectoryService); ok {
		directoryService = existing
	}
	if directoryService == nil {
		directoryService = archiveservice.NewDirectoryService(repos.ArchiveDirectory(), repos.ArchiveFile(), f.host.Cache())
		servicesRegistry.Set(archiveapi.ServiceDirectory, directoryService)
	} else {
		servicesRegistry.Set(archiveapi.ServiceDirectory, directoryService)
	}

	var fileService *archiveservice.FileService
	if existing, ok := servicesRegistry.Get(archiveapi.ServiceFile).(*archiveservice.FileService); ok {
		fileService = existing
	}
	if fileService == nil {
		fileService = archiveservice.NewFileService(repos.ArchiveFile(), repos.ArchiveDirectory(), directoryService)
		servicesRegistry.Set(archiveapi.ServiceFile, fileService)
	} else {
		servicesRegistry.Set(archiveapi.ServiceFile, fileService)
	}

	if handler, ok := handlersRegistry.Get(archiveapi.HandlerDirectory).(*archivehandlers.DirectoryHandler); ok {
		handler.SetService(directoryService)
	} else {
		handlersRegistry.Set(archiveapi.HandlerDirectory, archivehandlers.NewDirectoryHandler(directoryService))
	}

	if handler, ok := handlersRegistry.Get(archiveapi.HandlerFile).(*archivehandlers.FileHandler); ok {
		handler.SetService(fileService)
	} else {
		handlersRegistry.Set(archiveapi.HandlerFile, archivehandlers.NewFileHandler(fileService))
	}

	if handler, ok := handlersRegistry.Get(archiveapi.HandlerPublic).(*archivehandlers.PublicHandler); ok {
		handler.SetServices(directoryService, fileService)
	} else {
		handlersRegistry.Set(archiveapi.HandlerPublic, archivehandlers.NewPublicHandler(directoryService, fileService))
	}

	if templateHandler := f.host.TemplateHandler(); templateHandler != nil {
		templateHandler.SetArchiveServices(directoryService, fileService)
	}

	archiveseed.EnsureDefaultStructure(directoryService)

	return nil
}

func (f *Feature) Deactivate() error {
	if f == nil || f.host == nil {
		return nil
	}

	servicesRegistry := f.host.Services(archiveapi.Namespace)
	handlersRegistry := f.host.Handlers(archiveapi.Namespace)

	servicesRegistry.Delete(archiveapi.ServiceDirectory)
	servicesRegistry.Delete(archiveapi.ServiceFile)

	handlersRegistry.Delete(archiveapi.HandlerDirectory)
	handlersRegistry.Delete(archiveapi.HandlerFile)
	handlersRegistry.Delete(archiveapi.HandlerPublic)

	if templateHandler := f.host.TemplateHandler(); templateHandler != nil {
		templateHandler.SetArchiveServices(nil, nil)
	}

	return nil
}
