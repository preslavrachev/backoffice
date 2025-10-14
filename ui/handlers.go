package ui

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/preslavrachev/backoffice/core"
	"github.com/preslavrachev/backoffice/middleware/auth"
)

// Handler returns an HTTP handler for the admin panel
func Handler(bo *core.BackOffice, basePath string) http.Handler {
	// Create a wrapper struct to hold the BackOffice instance and provide the handler methods
	handler := &BackOfficeHandler{bo: bo}

	mux := http.NewServeMux()

	// Authentication routes (if auth is enabled)
	authConfig := bo.GetAuth()
	if authConfig != nil && authConfig.Enabled {
		mux.HandleFunc(basePath+authConfig.LoginPath, handler.loginHandler)
		mux.HandleFunc(basePath+authConfig.LogoutPath, handler.logoutHandler)
	}

	// HTML routes
	mux.HandleFunc(basePath+"/", handler.indexHandler)
	mux.HandleFunc(basePath+"/api/", handler.apiRouter) // Keep API for HTMX operations

	// Apply auth middleware
	var finalHandler http.Handler = mux
	if authConfig != nil {
		authMiddleware := auth.CreateAuthMiddleware(authConfig)
		finalHandler = authMiddleware(finalHandler)
	}

	return finalHandler
}

// BackOfficeHandler wraps BackOffice to provide HTTP handler methods
type BackOfficeHandler struct {
	bo *core.BackOffice
}

// indexHandler serves the main index page
func (h *BackOfficeHandler) indexHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, h.bo.GetConfig().BasePath)
	path = strings.Trim(path, "/")

	if path == "" {
		// Main admin index
		h.renderIndex(w, r)
		return
	}

	// Parse path segments for resource routing
	segments := strings.Split(path, "/")
	resourceName := segments[0]

	resource, exists := h.bo.GetResource(resourceName)
	if !exists {
		http.NotFound(w, r)
		return
	}

	switch len(segments) {
	case 1:
		// /admin/users - resource list
		h.renderResourceList(w, r, resource)
	case 2:
		if segments[1] == "new" {
			// /admin/users/new - create form
			h.renderCreateForm(w, r, resource)
		} else {
			// /admin/users/123 - resource detail
			// Handle DELETE method (via form with _method=DELETE)
			if r.Method == http.MethodPost && r.FormValue("_method") == "DELETE" {
				h.handleDeleteResourceFromDetail(w, r, resource, segments[1])
				return
			}
			h.renderResourceDetail(w, r, resource, segments[1])
		}
	case 3:
		if segments[2] == "edit" {
			// /admin/users/123/edit - edit form
			h.renderEditForm(w, r, resource, segments[1])
		} else {
			http.NotFound(w, r)
		}
	default:
		http.NotFound(w, r)
	}
}

// renderIndex renders the main admin index page
func (h *BackOfficeHandler) renderIndex(w http.ResponseWriter, r *http.Request) {
	// Get resources in registration order and filter visible ones
	var visibleResources []*core.Resource
	for _, resource := range h.bo.GetResources() {
		if !resource.Hidden {
			visibleResources = append(visibleResources, resource)
		}
	}

	indexComponent := Index(visibleResources)

	// Get user from context for auth-aware layout
	user, _ := auth.GetAuthUser(r.Context())
	layoutComponent := LayoutWithAuth(h.bo.GetConfig().Title, indexComponent, user)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := layoutComponent.Render(context.Background(), w); err != nil {
		h.writeHTTPError(w, "Template rendering error", http.StatusInternalServerError)
	}
}

// renderResourceList renders the resource list page
func (h *BackOfficeHandler) renderResourceList(w http.ResponseWriter, r *http.Request, resource *core.Resource) {
	// Handle POST for create operations
	if r.Method == http.MethodPost {
		h.handleCreateResource(w, r, resource)
		return
	}

	// Parse query from request parameters
	query := parseQueryFromRequest(r, resource)

	// Check if this is a "load more" request (HTMX partial response)
	isLoadMore := r.URL.Query().Get("load_more") == "true"

	// Execute query using the new Find method
	result, err := h.bo.GetAdapter().Find(r.Context(), resource, query)
	if err != nil {
		h.writeHTTPError(w, fmt.Sprintf("Failed to get items: %v", err), http.StatusInternalServerError)
		return
	}

	if isLoadMore {
		// Return only the additional rows for HTMX append
		h.renderLoadMoreRows(w, r, resource, result)
		return
	}

	// Create context with sort information for templates
	ctx := r.Context()
	if primarySort := query.GetPrimarySort(); primarySort != nil {
		ctx = context.WithValue(ctx, "currentSortField", primarySort.Field)
		ctx = context.WithValue(ctx, "currentSortDirection", string(primarySort.Direction))
	}

	// Generate Load More URL if needed
	var loadMoreURL string
	if result.HasMore {
		nextQuery := result.Query.NextPage()
		loadMoreURL = NewAdminURL(resource.Name).
			PreserveFromRequest(r).
			WithPagination(nextQuery.Pagination.Offset, nextQuery.Pagination.Limit).
			WithLoadMore().
			String()
	}

	// Render full list page
	listComponent := List(resource, result.Items, int(result.TotalCount), loadMoreURL)

	// Get user from context for auth-aware layout
	user, _ := auth.GetAuthUser(ctx)
	layoutComponent := LayoutWithAuth(resource.PluralName, listComponent, user)

	// Check for success messages
	if successType := r.URL.Query().Get("success"); successType == "delete" {
		if resourceName := r.URL.Query().Get("resource"); resourceName != "" {
			w.Header().Set("HX-Trigger", fmt.Sprintf(`{"showToast": {"message": "%s deleted successfully", "type": "success"}}`, resourceName))
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := layoutComponent.Render(ctx, w); err != nil {
		h.writeHTTPError(w, "Template rendering error", http.StatusInternalServerError)
	}
}

// renderResourceDetail renders the resource detail page
func (h *BackOfficeHandler) renderResourceDetail(w http.ResponseWriter, r *http.Request, resource *core.Resource, idStr string) {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.writeHTTPError(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	item, err := h.bo.GetAdapter().GetByID(r.Context(), resource, uint(id))
	if err != nil {
		h.writeHTTPError(w, fmt.Sprintf("Failed to get item: %v", err), http.StatusNotFound)
		return
	}

	detailComponent := Detail(resource, item)
	layoutComponent := Layout(resource.DisplayName+" Detail", detailComponent)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := layoutComponent.Render(context.Background(), w); err != nil {
		h.writeHTTPError(w, "Template rendering error", http.StatusInternalServerError)
	}
}

// renderCreateForm renders the create form page
func (h *BackOfficeHandler) renderCreateForm(w http.ResponseWriter, r *http.Request, resource *core.Resource) {
	formComponent := Form(resource, nil, false)
	layoutComponent := Layout("Create "+resource.DisplayName, formComponent)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := layoutComponent.Render(context.Background(), w); err != nil {
		h.writeHTTPError(w, "Template rendering error", http.StatusInternalServerError)
	}
}

// renderEditForm renders the edit form page
func (h *BackOfficeHandler) renderEditForm(w http.ResponseWriter, r *http.Request, resource *core.Resource, idStr string) {
	// Handle PUT for update operations
	if r.Method == http.MethodPost {
		method := r.FormValue("_method")
		if method == "PUT" {
			h.handleUpdateResource(w, r, resource, idStr)
			return
		}
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.writeHTTPError(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	item, err := h.bo.GetAdapter().GetByID(r.Context(), resource, uint(id))
	if err != nil {
		h.writeHTTPError(w, fmt.Sprintf("Failed to get item: %v", err), http.StatusNotFound)
		return
	}

	formComponent := Form(resource, item, true)
	layoutComponent := Layout("Edit "+resource.DisplayName, formComponent)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := layoutComponent.Render(context.Background(), w); err != nil {
		h.writeHTTPError(w, "Template rendering error", http.StatusInternalServerError)
	}
}

// handleCreateResource handles POST requests for creating resources
func (h *BackOfficeHandler) handleCreateResource(w http.ResponseWriter, r *http.Request, resource *core.Resource) {
	if resource.ReadOnly {
		h.writeHTTPError(w, "Resource is read-only", http.StatusForbidden)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.writeHTTPError(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Convert form data to struct instance
	item, err := h.formToStruct(r, resource)
	if err != nil {
		h.writeHTTPError(w, fmt.Sprintf("Invalid data format: %v", err), http.StatusBadRequest)
		return
	}

	// Validate data
	if err := h.bo.GetAdapter().ValidateData(resource, item); err != nil {
		h.writeHTTPError(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	// Create item
	if err := h.bo.GetAdapter().Create(r.Context(), resource, item); err != nil {
		h.writeHTTPError(w, fmt.Sprintf("Failed to create item: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect to list view
	http.Redirect(w, r, h.bo.GetConfig().BasePath+"/"+resource.Name, http.StatusSeeOther)
}

// handleUpdateResource handles PUT requests for updating resources
func (h *BackOfficeHandler) handleUpdateResource(w http.ResponseWriter, r *http.Request, resource *core.Resource, idStr string) {
	if resource.ReadOnly {
		h.writeHTTPError(w, "Resource is read-only", http.StatusForbidden)
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.writeHTTPError(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.writeHTTPError(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Convert form data to struct instance
	item, err := h.formToStruct(r, resource)
	if err != nil {
		h.writeHTTPError(w, fmt.Sprintf("Invalid data format: %v", err), http.StatusBadRequest)
		return
	}

	// Update item
	if err := h.bo.GetAdapter().Update(r.Context(), resource, uint(id), item); err != nil {
		h.writeHTTPError(w, fmt.Sprintf("Failed to update item: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect to detail view
	http.Redirect(w, r, h.bo.GetConfig().BasePath+"/"+resource.Name+"/"+idStr, http.StatusSeeOther)
}

// apiRouter routes API requests (for HTMX operations)
func (h *BackOfficeHandler) apiRouter(w http.ResponseWriter, r *http.Request) {
	// Remove the base path and /api/ prefix
	path := strings.TrimPrefix(r.URL.Path, h.bo.GetConfig().BasePath+"/api/")
	segments := strings.Split(strings.Trim(path, "/"), "/")

	if len(segments) < 1 || segments[0] == "" {
		h.writeHTTPError(w, "Invalid API path", http.StatusBadRequest)
		return
	}

	resourceName := segments[0]
	resource, exists := h.bo.GetResource(resourceName)
	if !exists {
		h.writeHTTPError(w, fmt.Sprintf("Resource '%s' not found", resourceName), http.StatusNotFound)
		return
	}

	switch len(segments) {
	case 1:
		if r.Method == http.MethodPost {
			// POST /api/users - handle create
			h.handleCreateResourceAPI(w, r, resource)
		} else {
			h.writeHTTPError(w, "Invalid API operation", http.StatusMethodNotAllowed)
		}
	case 2:
		if segments[1] == "new" && r.Method == http.MethodGet {
			// GET /api/users/new - return create form side pane
			h.renderCreateSidePane(w, r, resource)
		} else if r.Method == http.MethodDelete {
			// DELETE /api/users/123
			h.handleDeleteResource(w, r, resource, segments[1])
		} else if r.Method == http.MethodPost {
			// POST /api/users/123 - handle update
			h.handleUpdateResourceAPI(w, r, resource, segments[1])
		} else {
			h.writeHTTPError(w, "Invalid API operation", http.StatusMethodNotAllowed)
		}
	case 3:
		if segments[2] == "edit" && r.Method == http.MethodGet {
			// GET /api/users/123/edit - return edit form side pane
			h.renderEditSidePane(w, r, resource, segments[1])
		} else if segments[2] == "action" && r.Method == http.MethodPost {
			// POST /api/users/123/action - execute custom action
			h.handleCustomAction(w, r, resource, segments[1])
		} else {
			h.writeHTTPError(w, "Invalid API operation", http.StatusMethodNotAllowed)
		}
	case 4:
		if segments[2] == "related" && r.Method == http.MethodGet {
			// GET /api/Category/123/related/Children - return related items modal
			h.handleRelatedItemsModal(w, r, resource, segments[1], segments[3])
		} else {
			h.writeHTTPError(w, "Invalid API operation", http.StatusMethodNotAllowed)
		}
	default:
		h.writeHTTPError(w, "Invalid API path", http.StatusBadRequest)
	}
}

// handleDeleteResource handles DELETE requests
func (h *BackOfficeHandler) handleDeleteResource(w http.ResponseWriter, r *http.Request, resource *core.Resource, idStr string) {
	if resource.ReadOnly {
		h.writeHTTPErrorWithToast(w, "Cannot delete: Resource is read-only", http.StatusForbidden, "error")
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.writeHTTPErrorWithToast(w, "Invalid ID format", http.StatusBadRequest, "error")
		return
	}

	// First check if the resource exists
	_, err = h.bo.GetAdapter().GetByID(r.Context(), resource, uint(id))
	if err != nil {
		h.writeHTTPErrorWithToast(w, fmt.Sprintf("%s not found", resource.DisplayName), http.StatusNotFound, "error")
		return
	}

	// Perform the deletion
	if err := h.bo.GetAdapter().Delete(r.Context(), resource, uint(id)); err != nil {
		h.writeHTTPErrorWithToast(w, fmt.Sprintf("Failed to delete %s: %v", resource.DisplayName, err), http.StatusInternalServerError, "error")
		return
	}

	// Return success response with toast notification
	w.Header().Set("HX-Trigger", fmt.Sprintf(`{"showToast": {"message": "%s deleted successfully", "type": "success"}}`, resource.DisplayName))
	w.WriteHeader(http.StatusOK)
}

// writeHTTPError writes an HTTP error response
func (h *BackOfficeHandler) writeHTTPError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, "<html><body><h1>Error %d</h1><p>%s</p></body></html>", statusCode, message)
}

// writeHTTPErrorWithToast writes an HTTP error response with toast notification
func (h *BackOfficeHandler) writeHTTPErrorWithToast(w http.ResponseWriter, message string, statusCode int, toastType string) {
	w.Header().Set("HX-Trigger", fmt.Sprintf(`{"showToast": {"message": "%s", "type": "%s"}}`, message, toastType))
	w.WriteHeader(statusCode)
}

// handleDeleteResourceFromDetail handles DELETE requests from the detail view
func (h *BackOfficeHandler) handleDeleteResourceFromDetail(w http.ResponseWriter, r *http.Request, resource *core.Resource, idStr string) {
	if resource.ReadOnly {
		h.writeHTTPError(w, "Resource is read-only", http.StatusForbidden)
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.writeHTTPError(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	// First check if the resource exists
	_, err = h.bo.GetAdapter().GetByID(r.Context(), resource, uint(id))
	if err != nil {
		h.writeHTTPError(w, fmt.Sprintf("%s not found", resource.DisplayName), http.StatusNotFound)
		return
	}

	// Perform the deletion
	if err := h.bo.GetAdapter().Delete(r.Context(), resource, uint(id)); err != nil {
		h.writeHTTPError(w, fmt.Sprintf("Failed to delete %s: %v", resource.DisplayName, err), http.StatusInternalServerError)
		return
	}

	// Redirect to list view with success message
	http.Redirect(w, r, h.bo.GetConfig().BasePath+"/"+resource.Name+"?success=delete&resource="+resource.DisplayName, http.StatusSeeOther)
}

// formToStruct converts form data to a struct instance
func (h *BackOfficeHandler) formToStruct(r *http.Request, resource *core.Resource) (interface{}, error) {
	// Create new instance of the model type
	item := newInstance(resource.ModelType)
	val := reflect.ValueOf(item).Elem()

	// Set form values to struct fields
	for _, field := range resource.Fields {
		if field.PrimaryKey {
			continue // Skip ID field for creation
		}

		formValue := r.FormValue(field.Name)
		fieldVal := val.FieldByName(field.Name)

		if !fieldVal.IsValid() || !fieldVal.CanSet() {
			continue
		}

		if err := h.setFieldValue(fieldVal, formValue, field.Type); err != nil {
			return nil, fmt.Errorf("error setting field %s: %v", field.Name, err)
		}
	}

	return item, nil
}

// setFieldValue sets a struct field value from a string
func (h *BackOfficeHandler) setFieldValue(fieldVal reflect.Value, value, fieldType string) error {
	if value == "" {
		return nil // Skip empty values
	}

	switch fieldType {
	case "string":
		fieldVal.SetString(value)
	case "int", "int32", "int64":
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			fieldVal.SetInt(intVal)
		}
	case "uint", "uint32", "uint64":
		if uintVal, err := strconv.ParseUint(value, 10, 64); err == nil {
			fieldVal.SetUint(uintVal)
		}
	case "float32", "float64":
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			fieldVal.SetFloat(floatVal)
		}
	case "bool":
		fieldVal.SetBool(value == "true" || value == "on")
	}

	return nil
}

// newInstance creates a new instance of the given type
func newInstance(t reflect.Type) interface{} {
	if t.Kind() == reflect.Ptr {
		return reflect.New(t.Elem()).Interface()
	}
	return reflect.New(t).Interface()
}

// renderCreateSidePane renders the create form in a side pane
func (h *BackOfficeHandler) renderCreateSidePane(w http.ResponseWriter, r *http.Request, resource *core.Resource) {
	title := "Create " + resource.DisplayName
	formComponent := SidePaneForm(resource, nil, false)
	sidePaneComponent := SidePane(title, formComponent)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := sidePaneComponent.Render(context.Background(), w); err != nil {
		h.writeHTTPError(w, "Template rendering error", http.StatusInternalServerError)
	}
}

// renderEditSidePane renders the edit form in a side pane
func (h *BackOfficeHandler) renderEditSidePane(w http.ResponseWriter, r *http.Request, resource *core.Resource, idStr string) {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.writeHTTPError(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	item, err := h.bo.GetAdapter().GetByID(r.Context(), resource, uint(id))
	if err != nil {
		h.writeHTTPError(w, fmt.Sprintf("Failed to get item: %v", err), http.StatusNotFound)
		return
	}

	title := "Edit " + resource.DisplayName
	formComponent := SidePaneForm(resource, item, true)
	sidePaneComponent := SidePane(title, formComponent)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := sidePaneComponent.Render(context.Background(), w); err != nil {
		h.writeHTTPError(w, "Template rendering error", http.StatusInternalServerError)
	}
}

// handleCreateResourceAPI handles API POST requests for creating resources with HTMX response
func (h *BackOfficeHandler) handleCreateResourceAPI(w http.ResponseWriter, r *http.Request, resource *core.Resource) {
	fmt.Printf("🔍 DEBUG: handleCreateResourceAPI called for resource: %s\n", resource.Name)

	if resource.ReadOnly {
		fmt.Printf("❌ DEBUG: Resource %s is read-only\n", resource.Name)
		h.writeHTTPErrorWithToast(w, "Resource is read-only", http.StatusForbidden, "error")
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		fmt.Printf("❌ DEBUG: Failed to parse form: %v\n", err)
		h.writeHTTPErrorWithToast(w, "Invalid form data", http.StatusBadRequest, "error")
		return
	}
	fmt.Printf("✅ DEBUG: Form data parsed successfully: %v\n", r.Form)

	// Convert form data to struct instance
	item, err := h.formToStruct(r, resource)
	if err != nil {
		fmt.Printf("❌ DEBUG: Failed to convert form to struct: %v\n", err)
		h.writeHTTPErrorWithToast(w, fmt.Sprintf("Invalid data format: %v", err), http.StatusBadRequest, "error")
		return
	}
	fmt.Printf("✅ DEBUG: Form converted to struct: %+v\n", item)

	// Validate data
	if err := h.bo.GetAdapter().ValidateData(resource, item); err != nil {
		fmt.Printf("❌ DEBUG: Validation failed: %v\n", err)
		h.writeHTTPErrorWithToast(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest, "error")
		return
	}
	fmt.Printf("✅ DEBUG: Data validation passed\n")

	// Create item
	if err := h.bo.GetAdapter().Create(r.Context(), resource, item); err != nil {
		fmt.Printf("❌ DEBUG: Failed to create item: %v\n", err)
		h.writeHTTPErrorWithToast(w, fmt.Sprintf("Failed to create item: %v", err), http.StatusInternalServerError, "error")
		return
	}
	fmt.Printf("✅ DEBUG: Item created successfully\n")

	// Get the ID of the created item
	createdID := core.GetFieldValue(item, resource.IDField)
	fmt.Printf("✅ DEBUG: Created item ID: %v\n", createdID)

	w.WriteHeader(http.StatusOK)
	// Return a script to close the side pane, show toast, and reload with highlight
	fmt.Fprintf(w, `<script>
		console.log('✅ DEBUG: Create successful, closing side pane and reloading');
		// Close side pane
		const sidePane = document.getElementById('sidepane-overlay');
		if (sidePane) {
			const alpineData = Alpine.$data(sidePane.querySelector('[x-data]'));
			if (alpineData) {
				alpineData.show = false;
				setTimeout(() => sidePane.remove(), 300);
			}
		}
		// Show toast
		showToast('%s created successfully', 'success');
		// Store the ID to highlight after reload
		sessionStorage.setItem('highlightItemId', '%v');
		sessionStorage.setItem('highlightAction', 'created');
		// Reload page after short delay
		setTimeout(() => {
			console.log('🔄 Reloading page...');
			window.location.reload();
		}, 500);
	</script>`, resource.DisplayName, createdID)
}

// handleUpdateResourceAPI handles API PUT/POST requests for updating resources with HTMX response
func (h *BackOfficeHandler) handleUpdateResourceAPI(w http.ResponseWriter, r *http.Request, resource *core.Resource, idStr string) {
	if resource.ReadOnly {
		h.writeHTTPErrorWithToast(w, "Resource is read-only", http.StatusForbidden, "error")
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.writeHTTPErrorWithToast(w, "Invalid ID format", http.StatusBadRequest, "error")
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.writeHTTPErrorWithToast(w, "Invalid form data", http.StatusBadRequest, "error")
		return
	}

	// Convert form data to struct instance
	item, err := h.formToStruct(r, resource)
	if err != nil {
		h.writeHTTPErrorWithToast(w, fmt.Sprintf("Invalid data format: %v", err), http.StatusBadRequest, "error")
		return
	}

	// Update item
	if err := h.bo.GetAdapter().Update(r.Context(), resource, uint(id), item); err != nil {
		h.writeHTTPErrorWithToast(w, fmt.Sprintf("Failed to update item: %v", err), http.StatusInternalServerError, "error")
		return
	}

	w.WriteHeader(http.StatusOK)
	// Return a script to close the side pane, show toast, and reload with highlight
	fmt.Fprintf(w, `<script>
		console.log('✅ DEBUG: Update successful, closing side pane and reloading');
		// Close side pane
		const sidePane = document.getElementById('sidepane-overlay');
		if (sidePane) {
			const alpineData = Alpine.$data(sidePane.querySelector('[x-data]'));
			if (alpineData) {
				alpineData.show = false;
				setTimeout(() => sidePane.remove(), 300);
			}
		}
		// Show toast
		showToast('%s updated successfully', 'success');
		// Store the ID to highlight after reload
		sessionStorage.setItem('highlightItemId', '%s');
		sessionStorage.setItem('highlightAction', 'updated');
		// Reload page after short delay
		setTimeout(() => {
			console.log('🔄 Reloading page...');
			window.location.reload();
		}, 500);
	</script>`, resource.DisplayName, idStr)
}

// handleRelatedItemsModal handles requests for showing related items in a modal
func (h *BackOfficeHandler) handleRelatedItemsModal(w http.ResponseWriter, r *http.Request, resource *core.Resource, idStr, fieldName string) {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.writeHTTPError(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	// Get the parent item with preloaded relationships
	item, err := h.bo.GetAdapter().GetByID(r.Context(), resource, uint(id))
	if err != nil {
		h.writeHTTPError(w, fmt.Sprintf("Failed to get item: %v", err), http.StatusNotFound)
		return
	}

	// Extract the related items from the specified field
	relatedItems := core.GetFieldValue(item, fieldName)
	if relatedItems == nil {
		relatedItems = []interface{}{}
	}

	// Convert to slice of interfaces
	var items []interface{}
	reflectVal := reflect.ValueOf(relatedItems)
	if reflectVal.Kind() == reflect.Slice || reflectVal.Kind() == reflect.Array {
		for i := 0; i < reflectVal.Len(); i++ {
			items = append(items, reflectVal.Index(i).Interface())
		}
	}

	// Determine the related resource (for proper display and actions)
	var relatedResource *core.Resource
	if fieldName == "Children" {
		// Children are of the same type as parent
		relatedResource = resource
	} else {
		// Try to find the related resource by field name
		for _, res := range h.bo.GetResources() {
			// Match field name with resource name (e.g., "Products" -> "Product")
			if fieldName == res.PluralName || fieldName == res.Name+"s" {
				relatedResource = res
				break
			}
		}
	}

	// Generate modal title
	title := fmt.Sprintf("%s for %s", fieldName, core.GetFieldValue(item, getDisplayFieldName(resource)))

	// Render modal
	modalComponent := RelatedItemsModal(title, items, relatedResource, fieldName)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := modalComponent.Render(context.Background(), w); err != nil {
		h.writeHTTPError(w, "Template rendering error", http.StatusInternalServerError)
	}
}

// Helper function to get the best display field name for a resource
func getDisplayFieldName(resource *core.Resource) string {
	// Try common display field names in order of preference
	possibleFields := []string{"Name", "Title", "DisplayName"}

	for _, fieldName := range possibleFields {
		for _, field := range resource.Fields {
			if field.Name == fieldName {
				return fieldName
			}
		}
	}

	// Fallback to ID field
	return resource.IDField
}

// loginHandler handles login requests
func (h *BackOfficeHandler) loginHandler(w http.ResponseWriter, r *http.Request) {
	authConfig := h.bo.GetAuth()
	if authConfig == nil || !authConfig.Enabled {
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodGet {
		// Show login form
		h.renderLoginForm(w, r)
		return
	}

	if r.Method == http.MethodPost {
		// Process login
		fmt.Printf("🔐 DEBUG: Processing login form submission\n")
		if err := r.ParseForm(); err != nil {
			fmt.Printf("❌ DEBUG: Failed to parse form: %v\n", err)
			h.writeHTTPError(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")
		fmt.Printf("🔐 DEBUG: Login attempt - Username: '%s', Password length: %d\n", username, len(password))

		// Authenticate user
		user, err := authConfig.Authenticator(r.Context(), username, password)
		if err != nil {
			fmt.Printf("❌ DEBUG: Authentication failed: %v\n", err)
			// Login failed - show form with error
			h.renderLoginFormWithError(w, r, "Invalid username or password")
			return
		}
		fmt.Printf("✅ DEBUG: Authentication successful for user: %s\n", user.Username)

		// Create session
		fmt.Printf("🔐 DEBUG: Creating session for user: %s\n", user.Username)
		sessionID, err := authConfig.SessionStore.CreateSession(r.Context(), user)
		if err != nil {
			fmt.Printf("❌ DEBUG: Failed to create session: %v\n", err)
			h.writeHTTPError(w, "Failed to create session", http.StatusInternalServerError)
			return
		}
		fmt.Printf("✅ DEBUG: Session created with ID: %s\n", sessionID)

		// Set session cookie
		cookie := auth.CreateSessionCookie(sessionID)
		fmt.Printf("🔐 DEBUG: Setting session cookie: %s\n", cookie.String())
		http.SetCookie(w, cookie)

		// Redirect to original page or admin home
		redirectURL := r.FormValue("return")
		if redirectURL == "" {
			redirectURL = authConfig.LoginRedirect
		}
		fmt.Printf("🔐 DEBUG: Redirecting to: %s\n", redirectURL)
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// logoutHandler handles logout requests
func (h *BackOfficeHandler) logoutHandler(w http.ResponseWriter, r *http.Request) {
	authConfig := h.bo.GetAuth()
	if authConfig == nil || !authConfig.Enabled {
		http.NotFound(w, r)
		return
	}

	// Get current session to delete it
	if cookie, err := r.Cookie("backoffice_session"); err == nil {
		authConfig.SessionStore.DeleteSession(r.Context(), cookie.Value)
	}

	// Delete session cookie
	http.SetCookie(w, auth.DeleteSessionCookie())

	// Redirect to logout page
	http.Redirect(w, r, authConfig.LogoutRedirect, http.StatusSeeOther)
}

// renderLoginForm renders the login form
func (h *BackOfficeHandler) renderLoginForm(w http.ResponseWriter, r *http.Request) {
	h.renderLoginFormWithError(w, r, "")
}

// renderLoginFormWithError renders the login form with an error message
func (h *BackOfficeHandler) renderLoginFormWithError(w http.ResponseWriter, r *http.Request, errorMsg string) {
	returnURL := r.URL.Query().Get("return")

	// For now, render a simple HTML form
	// TODO: Replace with templ component in next task
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>Login - %s</title>
    <style>
        body { font-family: sans-serif; margin: 40px; }
        .login-form { max-width: 400px; margin: 0 auto; }
        .form-group { margin-bottom: 15px; }
        label { display: block; margin-bottom: 5px; }
        input[type=text], input[type=password] { width: 100%%; padding: 8px; border: 1px solid #ccc; }
        button { background: #007bff; color: white; padding: 10px 20px; border: none; cursor: pointer; }
        .error { color: red; margin-bottom: 15px; }
    </style>
</head>
<body>
    <div class="login-form">
        <h1>Login to %s</h1>
        %s
        <form method="post">
            <input type="hidden" name="return" value="%s">
            <div class="form-group">
                <label>Username:</label>
                <input type="text" name="username" required>
            </div>
            <div class="form-group">
                <label>Password:</label>
                <input type="password" name="password" required>
            </div>
            <button type="submit">Login</button>
        </form>
    </div>
</body>
</html>`,
		h.bo.GetConfig().Title,
		h.bo.GetConfig().Title,
		func() string {
			if errorMsg != "" {
				return fmt.Sprintf(`<div class="error">%s</div>`, errorMsg)
			}
			return ""
		}(),
		returnURL)
}

// parseQueryFromRequest parses HTTP request parameters into a Query struct
func parseQueryFromRequest(r *http.Request, resource *core.Resource) *core.Query {
	query := core.NewQuery()

	// Parse filters (exclude UI and pagination parameters)
	filters := make(map[string]any)
	for key, values := range r.URL.Query() {
		if len(values) > 0 && !isReservedParam(key) {
			filters[key] = values[0]
		}
	}
	query.WithFilters(filters)

	// Parse sorting
	if sortBy := r.URL.Query().Get("sort"); sortBy != "" {
		direction := core.SortAsc // default
		if sortDir := r.URL.Query().Get("direction"); sortDir == "desc" {
			direction = core.SortDesc
		}
		query.WithSort(sortBy, direction)
	}

	// Parse pagination
	limit := core.DefaultPageSize
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil {
			offset = parsedOffset
		}
	}

	query.WithPagination(limit, offset)

	return query
}

// isReservedParam checks if a parameter is reserved for UI functionality
func isReservedParam(param string) bool {
	reserved := []string{
		"limit", "offset", "sort", "direction",
		"success", "resource", "page", "load_more",
	}

	for _, r := range reserved {
		if param == r {
			return true
		}
	}
	return false
}

// handleCustomAction handles execution of custom actions
func (h *BackOfficeHandler) handleCustomAction(w http.ResponseWriter, r *http.Request, resource *core.Resource, idStr string) {
	// Parse form to get action ID
	if err := r.ParseForm(); err != nil {
		h.writeHTTPErrorWithToast(w, "Invalid form data", http.StatusBadRequest, "error")
		return
	}

	actionID := r.FormValue("action_id")
	if actionID == "" {
		h.writeHTTPErrorWithToast(w, "Action ID is required", http.StatusBadRequest, "error")
		return
	}

	// Find the action
	var action *core.CustomAction
	for i := range resource.Actions {
		if resource.Actions[i].ID == actionID {
			action = &resource.Actions[i]
			break
		}
	}

	if action == nil {
		h.writeHTTPErrorWithToast(w, "Action not found", http.StatusNotFound, "error")
		return
	}

	// Parse ID
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.writeHTTPErrorWithToast(w, "Invalid ID format", http.StatusBadRequest, "error")
		return
	}

	// Execute the action
	if err := action.Handler(r.Context(), uint(id)); err != nil {
		h.writeHTTPErrorWithToast(w, fmt.Sprintf("Action failed: %v", err), http.StatusInternalServerError, "error")
		return
	}

	// Success - send toast notification
	w.Header().Set("HX-Trigger", fmt.Sprintf(`{"showToast": {"message": "%s completed successfully", "type": "success"}}`, action.Title))
	w.WriteHeader(http.StatusOK)
}

// renderLoadMoreRows renders additional rows for HTMX "Load More" functionality
func (h *BackOfficeHandler) renderLoadMoreRows(w http.ResponseWriter, r *http.Request, resource *core.Resource, result *core.Result) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Render only the additional rows
	for _, item := range result.Items {
		rowComponent := ListRow(resource, item)
		if err := rowComponent.Render(context.Background(), w); err != nil {
			h.writeHTTPError(w, "Template rendering error", http.StatusInternalServerError)
			return
		}
	}

	// If there are more results, include a new "Load More" button
	if result.HasMore {
		nextQuery := result.Query.NextPage()
		// Use AdminURLBuilder to construct URL, preserving all user parameters
		loadMoreURL := NewAdminURL(resource.Name).
			PreserveFromRequest(r).
			WithPagination(nextQuery.Pagination.Offset, nextQuery.Pagination.Limit).
			WithLoadMore().
			String()

		fmt.Fprintf(w, `
		<tr id="load-more-row">
			<td colspan="%d" class="px-6 py-4 text-center">
				<button hx-get="%s" 
				        hx-target="#load-more-row" 
				        hx-swap="outerHTML"
				        class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 transition-colors">
					Load More (%d more available)
				</button>
			</td>
		</tr>`, len(resource.Fields)+1, loadMoreURL, result.TotalCount-int64(result.Query.Pagination.Offset+len(result.Items)))
	}
}
