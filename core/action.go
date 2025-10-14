package core

import "context"

// CustomAction represents a custom action that can be performed on a resource
type CustomAction struct {
	// ID is the unique identifier for the action (used in URLs)
	ID string `json:"id"`

	// Title is the display name for the action shown in the UI
	Title string `json:"title"`

	// Handler is the function that executes when the action is triggered
	// It receives the context and the ID of the record to act upon
	// Returns an error if the action fails
	Handler func(ctx context.Context, id any) error `json:"-"`
}

// ActionBuilder provides a fluent API for configuring custom actions
type ActionBuilder struct {
	action *CustomAction
}

// NewAction creates a new action builder
func NewAction(id, title string, handler func(ctx context.Context, id any) error) *ActionBuilder {
	return &ActionBuilder{
		action: &CustomAction{
			ID:      id,
			Title:   title,
			Handler: handler,
		},
	}
}

// Build returns the built custom action
func (ab *ActionBuilder) Build() CustomAction {
	return *ab.action
}
