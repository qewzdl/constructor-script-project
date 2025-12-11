# Quick Start: Creating Custom Sections

## 🚀 3 Ways to Create Sections

### 1️⃣ Builder Pattern (Recommended for most cases)

```go
import "constructor-script-backend/internal/sections"

// Define renderer
renderer := func(ctx sections.RenderContext, prefix string, elem models.SectionElement) (string, []string) {
    return `<div>Your HTML</div>`, []string{"/static/js/script.js"}
}

// Build descriptor
desc, _ := sections.NewSectionBuilder("my_section").
    WithName("My Section").
    WithCategory("custom").
    WithRenderer(renderer).
    AddNumberField("limit", 1, 10, 5).
    Build()

// Register
templateHandler.RegisterSectionWithMetadata(desc)
```

### 2️⃣ Factory Pattern (For complex sections)

```go
factory := sections.NewSectionFactory()

blueprint := &sections.SectionBlueprint{
    Type: "my_section",
    Builder: func() *sections.SectionBuilder {
        return sections.NewSectionBuilder("my_section").
            WithRenderer(myRenderer)
    },
}

factory.RegisterBlueprint(blueprint)
```

### 3️⃣ Template Pattern (For lists/grids)

```go
template := &sections.SectionTemplate{
    Type:         "my_list",
    WrapperClass: "my-wrapper",
    EmptyMessage: "No items",
}

renderer := template.TemplateRenderer(fetchData, renderItems)
```

## 📋 Available Builder Methods

| Method | Purpose | Example |
|--------|---------|---------|
| `WithName(string)` | Display name | `.WithName("Features")` |
| `WithDescription(string)` | Description | `.WithDescription("Show features")` |
| `WithCategory(string)` | Category | `.WithCategory("marketing")` |
| `WithIcon(string)` | Icon name | `.WithIcon("star")` |
| `WithRenderer(Renderer)` | Render function | `.WithRenderer(myFunc)` |
| `AddStringField(name, required, default)` | String config | `.AddStringField("title", true)` |
| `AddNumberField(name, min, max, default)` | Number config | `.AddNumberField("limit", 1, 10, 5)` |
| `AddBooleanField(name, default)` | Boolean config | `.AddBooleanField("enabled", true)` |
| `AddEnumField(name, options, default)` | Dropdown config | `.AddEnumField("style", []string{"grid", "list"}, "grid")` |
| `AddArrayField(name, type, min, max)` | Array config | `.AddArrayField("items", "string", 0, 10)` |

## 🎨 Common Patterns

### List Section
```go
sections.NewSectionBuilder("items_list").
    WithName("Items List").
    WithRenderer(listRenderer).
    AddNumberField("limit", 1, 50, 10).
    AddEnumField("layout", []string{"grid", "list"}, "grid").
    Build()
```

### Content Section
```go
sections.NewSectionBuilder("custom_content").
    WithName("Custom Content").
    WithRenderer(contentRenderer).
    AddStringField("title", false).
    AddBooleanField("show_border", true).
    Build()
```

### Media Section
```go
sections.NewSectionBuilder("gallery").
    WithName("Photo Gallery").
    WithRenderer(galleryRenderer).
    AddArrayField("images", "object", 1, 20).
    AddEnumField("columns", []string{"2", "3", "4"}, "3").
    Build()
```

## 🔌 Plugin Usage

```go
func (p *MyPlugin) OnLoad(handler *handlers.TemplateHandler) error {
    desc, err := sections.NewSectionBuilder("plugin_section").
        WithName("Plugin Section").
        WithCategory("plugin").
        WithRenderer(p.render).
        Build()
    
    if err != nil {
        return err
    }
    
    return handler.RegisterSectionWithMetadata(desc)
}

func (p *MyPlugin) render(ctx sections.RenderContext, prefix string, elem models.SectionElement) (string, []string) {
    services := ctx.Services()
    // Access your services here
    return `<div>Plugin content</div>`, nil
}
```

## ✅ Best Practices

1. ✅ Always call `Build()` to validate
2. ✅ Add schema fields for configurable options
3. ✅ Use meaningful category names
4. ✅ Include icons for better UX
5. ✅ Return empty string if no content to render
6. ✅ Sanitize user input via `ctx.SanitizeHTML()`
7. ✅ Deduplicate scripts (system does this automatically)

## 📖 Full Documentation

See `docs/flexible-sections.md` for complete documentation.
