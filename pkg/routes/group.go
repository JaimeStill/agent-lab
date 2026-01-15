package routes

// Group represents a collection of routes under a common URL prefix.
// Groups can contain child groups for hierarchical route organization.
type Group struct {
	Prefix      string
	Tags        []string
	Description string
	Routes      []Route
	Children    []Group
}
