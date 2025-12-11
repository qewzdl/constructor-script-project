# Flexible Section Creation System

## Overview

The section system has been redesigned to be highly flexible and scalable, making it easy to create new section types without modifying core code.

## Key Features

### 1. **Builder Pattern** 🏗️
Create sections with a fluent, chainable API:

```go
desc, err := sections.NewSectionBuilder("testimonials").
    WithName("Customer Testimonials").
    WithDescription("Display customer reviews").
    WithCategory("marketing").
    WithIcon("quote").
    WithRenderer(testimonialRenderer).
    AddNumberField("limit", 1, 10, 5).
    AddBooleanField("show_avatars", true).
    AddEnumField("layout", []string{"grid", "carousel"}, "grid").
    Build()
```

### 2. **Factory Pattern** 🏭
Define blueprints and instantiate sections:

```go
factory := sections.NewSectionFactory()

blueprint := &sections.SectionBlueprint{
    Type: "faq",
    Name: "FAQ Section",
    Builder: func() *sections.SectionBuilder {
        return sections.NewSectionBuilder("faq").
            WithName("FAQ").
            WithRenderer(faqRenderer).
            AddArrayField("questions", "object", 1, 20)
    },
}

factory.RegisterBlueprint(blueprint)
```

### 3. **Template Renderers** 📄
Reusable patterns for common section types:

```go
template := &sections.SectionTemplate{
    Type:         "features_list",
    WrapperClass: "features",
    EmptyMessage: "No features available",
}

renderer := template.TemplateRenderer(fetchData, renderItems)
```

### 4. **Composable Renderers** 🔗
Chain multiple renderers together:

```go
renderer := sections.ChainRenderer(
    headerRenderer,
    contentRenderer,
    footerRenderer,
)
```

### 5. **Conditional Rendering** ⚡
Execute renderers based on conditions:

```go
renderer := sections.ConditionalRenderer(
    func(ctx) bool { return userIsLoggedIn(ctx) },
    authenticatedRenderer,
    publicRenderer,
)
```

## Creating a New Section Type

### Method 1: Using Builder (Simple)

```go
// 1. Define the renderer
myRenderer := func(ctx sections.RenderContext, prefix string, elem models.SectionElement) (string, []string) {
    return `<div class="my-section">Content</div>`, []string{"/static/js/my-section.js"}
}

// 2. Build the descriptor
desc, err := sections.NewSectionBuilder("my_custom_section").
    WithName("My Custom Section").
    WithDescription("A custom section type").
    WithCategory("custom").
    WithIcon("star").
    WithRenderer(myRenderer).
    AddStringField("title", true).
    AddNumberField("columns", 1, 6, 3).
    Build()

if err != nil {
    log.Fatal(err)
}

// 3. Register with the handler
templateHandler.RegisterSectionWithMetadata(desc)
```

### Method 2: Using Factory (Advanced)

```go
// 1. Create a factory
factory := sections.NewSectionFactory()

// 2. Define a blueprint
blueprint := &sections.SectionBlueprint{
    Type:        "product_showcase",
    Name:        "Product Showcase",
    Description: "Display products in a grid",
    Category:    "ecommerce",
    Icon:        "shopping-cart",
    Builder: func() *sections.SectionBuilder {
        renderer := func(ctx sections.RenderContext, prefix string, elem models.SectionElement) (string, []string) {
            // Access services
            services := ctx.Services()
            productSvc := services.ProductService()
            
            // Fetch and render products
            products := productSvc.GetFeatured()
            return renderProducts(products), []string{"/static/js/products.js"}
        }

        return sections.NewSectionBuilder("product_showcase").
            WithName("Product Showcase").
            WithRenderer(renderer).
            AddNumberField("limit", 1, 24, 6).
            AddEnumField("style", []string{"grid", "carousel", "list"}, "grid").
            AddBooleanField("show_prices", true)
    },
}

// 3. Register the blueprint
factory.RegisterBlueprint(blueprint)

// 4. The section is now available
registry := factory.GetRegistry()
```

### Method 3: Using Templates (Pattern-Based)

```go
template := &sections.SectionTemplate{
    Type:            "team_members",
    WrapperClass:    "team-grid",
    ItemClass:       "team-member",
    EmptyMessage:    "No team members found",
    RequiresService: "team",
}

fetchData := func(ctx sections.RenderContext, section models.Section) (interface{}, error) {
    teamSvc := ctx.Services().TeamService()
    return teamSvc.GetMembers(section.Limit)
}

renderItems := func(ctx sections.RenderContext, data interface{}, prefix string) (string, []string, error) {
    members := data.([]TeamMember)
    var html strings.Builder
    
    for _, member := range members {
        html.WriteString(fmt.Sprintf(
            `<div class="team-member">
                <img src="%s" alt="%s">
                <h3>%s</h3>
                <p>%s</p>
            </div>`,
            member.Photo, member.Name, member.Name, member.Role,
        ))
    }
    
    return html.String(), nil, nil
}

renderer := template.TemplateRenderer(fetchData, renderItems)

desc := sections.NewSectionBuilder("team_members").
    WithName("Team Members").
    WithRenderer(renderer).
    Build()
```

## Schema Field Types

The builder supports various field types for section configuration:

### String Field
```go
AddStringField("title", required bool, defaultValue ...string)
```

### Number Field
```go
AddNumberField("limit", min int, max int, defaultValue ...int)
```

### Boolean Field
```go
AddBooleanField("show_images", defaultValue bool)
```

### Enum Field
```go
AddEnumField("layout", options []string, defaultValue ...string)
```

### Array Field
```go
AddArrayField("items", itemType string, minItems int, maxItems int)
```

### Custom Schema
```go
WithSchema(map[string]interface{}{
    "custom_field": map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "nested": map[string]interface{}{
                "type": "string",
            },
        },
    },
})
```

## Plugin Integration

Plugins can register custom sections easily:

```go
// In plugin initialization
func (p *MyPlugin) Initialize(app *application.Application) error {
    // Get template handler
    handler := app.GetTemplateHandler()
    
    // Create section descriptor
    desc, err := sections.NewSectionBuilder("plugin_section").
        WithName("Plugin Section").
        WithRenderer(p.renderSection).
        Build()
    
    if err != nil {
        return err
    }
    
    // Register with the system
    return handler.RegisterSectionWithMetadata(desc)
}
```

## Best Practices

1. **Use Builder for Simple Sections** - Quick and straightforward
2. **Use Factory for Complex Sections** - Better organization and reusability
3. **Use Templates for Common Patterns** - DRY principle
4. **Add Validation** - Use `WithValidator()` to validate section data
5. **Document Schema** - Clear field descriptions help editors
6. **Version Your Sections** - Include version in metadata for backward compatibility
7. **Test Renderers** - Unit test each renderer independently

## API Endpoint

Get all available sections with metadata:

```
GET /api/admin/sections/available
```

Response:
```json
{
  "sections": [
    {
      "type": "testimonials",
      "name": "Customer Testimonials",
      "description": "Display customer reviews",
      "category": "marketing",
      "icon": "quote",
      "schema": {
        "limit": {
          "type": "number",
          "min": 1,
          "max": 10,
          "default": 5
        }
      }
    }
  ],
  "has_metadata": true
}
```

## Migration from Old System

Old hardcoded sections are automatically registered. To migrate:

1. Create a builder for your section type
2. Register it with metadata
3. Old code continues to work during transition
4. Remove old hardcoded renderers when ready

## Performance

- ✅ Registry lookup: O(1)
- ✅ Metadata cached in memory
- ✅ No reflection overhead
- ✅ Lazy initialization supported
- ✅ Clone operations are efficient

## Examples

See `internal/sections/examples_test.go` for complete working examples.
