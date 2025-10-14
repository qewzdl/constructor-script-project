package navigation

// Item represents a navigation link that can be rendered in shared layouts.
// Modules can contribute their own navigation entries by returning Item values
// from the module registration and the application will pass the aggregated
// list to the template renderer.
type Item struct {
	Label string
	Path  string
}
