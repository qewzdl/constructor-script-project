# Scalable Section System Architecture

## Overview

The section system has been completely redesigned for maximum scalability and extensibility. All section types are now dynamically registered through a unified Registry system, making it easy to add new sections without modifying core code.

## Key Features

### 1. **Registry-Based Architecture**
All sections are registered through a central `Registry` that maps section types to their renderers.

```go
type Renderer func(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string)
```

### 2. **Metadata Support**
Sections can include rich metadata for admin interfaces:

```go
type SectionMetadata struct {
    Type        string                 // "posts_list"
    Name        string                 // "Posts List"
    Description string                 // Human-readable description
    Category    string                 // "content", "navigation", etc.
    Icon        string                 // Icon identifier
    Schema      map[string]interface{} // Configuration schema (JSON Schema)
    Preview     string                 // Preview image URL
}
```

### 3. **Service Access**
Section renderers have access to application services through the `RenderContext`:

```go
type RenderContext interface {
    SanitizeHTML(input string) string
    CloneTemplates() (*template.Template, error)
    Services() ServiceProvider
}

type ServiceProvider interface {
    PostService() interface{}
    CategoryService() interface{}
    CoursePackageService() interface{}
    CourseCheckoutService() interface{}
    SearchService() interface{}
    ThemeManager() interface{}
}
```

### 4. **Validation**
Optional validators can be attached to sections:

```go
type Validator func(elem interface{}) error
```

### 5. **Plugin Support**
Plugins can register custom sections at runtime:

```go
func RegisterCustomSection(templateHandler interface{}) error {
    th := templateHandler.(SectionRegistrar)
    
    desc := &sections.SectionDescriptor{
        Renderer: myRenderer,
        Metadata: sections.SectionMetadata{
            Type:        "my_custom_section",
            Name:        "My Custom Section",
            Description: "Does something amazing",
            Category:    "custom",
        },
    }
    
    return th.RegisterSectionWithMetadata(desc)
}
```

## Built-in Section Types

### Content Sections
- `paragraph` - Text paragraph with HTML support
- `image` - Single image with caption
- `image_group` - Multiple images in a gallery
- `file_group` - Downloadable files list
- `list` - Ordered or unordered list

### Dynamic List Sections
- `posts_list` - Recent blog posts
- `categories_list` - Blog categories
- `courses_list` - Course packages (catalog or owned)

### Profile Sections
- `profile_account` - Account settings
- `profile_security` - Security settings
- `profile_courses` - User's enrolled courses

### Utility Sections
- `search` - Search interface
- `grid` - Grid layout wrapper
- `standard` - Standard content wrapper

## API Endpoints

### Get Available Sections
```
GET /api/admin/sections/available
```

Returns metadata for all registered sections:

```json
{
  "sections": [
    {
      "type": "posts_list",
      "name": "Posts List",
      "description": "Displays a list of recent blog posts",
      "category": "content",
      "icon": "list",
      "schema": {
        "limit": {
          "type": "number",
          "default": 6,
          "min": 1,
          "max": 24
        }
      }
    }
  ],
  "has_metadata": true
}
```

## Creating a Custom Section

### Step 1: Create the Renderer

```go
package mysection

import (
    "fmt"
    "constructor-script-backend/internal/sections"
    "constructor-script-backend/internal/models"
)

func renderMySection(ctx sections.RenderContext, prefix string, elem models.SectionElement) (string, []string) {
    // Access services
    services := ctx.Services()
    postSvc := services.PostService()
    
    // Extract configuration
    content := elem.Content.(map[string]interface{})
    title := content["title"].(string)
    
    // Sanitize HTML
    safeTitle := ctx.SanitizeHTML(title)
    
    // Render HTML
    html := fmt.Sprintf(`<div class="%s__my-section">%s</div>`, prefix, safeTitle)
    
    // Return HTML and optional scripts
    scripts := []string{"/static/js/my-section.js"}
    return html, scripts
}
```

### Step 2: Register the Section

```go
func RegisterMySection(reg *sections.RegistryWithMetadata) error {
    desc := &sections.SectionDescriptor{
        Renderer: renderMySection,
        Metadata: sections.SectionMetadata{
            Type:        "my_section",
            Name:        "My Section",
            Description: "Custom section that does X",
            Category:    "custom",
            Icon:        "star",
            Schema: map[string]interface{}{
                "title": map[string]interface{}{
                    "type": "string",
                    "required": true,
                },
                "show_icon": map[string]interface{}{
                    "type": "boolean",
                    "default": true,
                },
            },
        },
        Validate: func(elem interface{}) error {
            // Optional validation
            return nil
        },
    }
    
    return reg.RegisterWithMetadata(desc)
}
```

### Step 3: Use in Plugin

```go
func (p *MyPlugin) Init(app interface{}) error {
    // Get template handler
    th := app.(TemplateHandlerProvider).TemplateHandler()
    
    // Register section
    return th.RegisterSectionWithMetadata(myDescriptor)
}
```

## Migration from Old System

### Before (Hardcoded)
```go
switch sectionType {
case "my_section":
    skipElements = true
    sb.WriteString(h.renderMySection(section))
}
```

### After (Registry-Based)
```go
// Automatic - just register in defaults.go
RegisterMySection(reg)
```

## Benefits

### ✅ Scalability
- Add unlimited section types without code changes
- Plugin-friendly architecture
- No switch-case statements to maintain

### ✅ Discoverability
- Admin interfaces can query available sections
- Rich metadata for better UX
- Schema validation support

### ✅ Maintainability
- Centralized registration
- Clear separation of concerns
- Type-safe service access

### ✅ Extensibility
- Plugins can add custom sections
- Override built-in sections
- Dynamic section loading

## Performance

- **No overhead**: Registry lookup is O(1) via hash map
- **Lazy loading**: Services accessed only when needed
- **Template caching**: Templates cloned efficiently
- **Concurrent-safe**: Registry uses RWMutex for thread safety

## Future Enhancements

1. **Hot Reload**: Reload sections without restart
2. **Version Management**: Support multiple section versions
3. **A/B Testing**: Register alternative renderers
4. **Analytics**: Track section usage and performance
5. **Visual Builder**: Drag-and-drop section composer
6. **Marketplace**: Share/download custom sections

## Examples

See `internal/sections/example_plugin.go` for a complete example of creating a custom section from a plugin.
