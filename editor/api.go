package editor

import (
	"context"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/validation"
)

// EditorAPI provides HTTP handlers for editor operations
type EditorAPI struct {
	fs  FileSystemManager
	val *validation.Validator
}

// NewEditorAPI creates a new editor API
func NewEditorAPI(fs FileSystemManager) *EditorAPI {
	return &EditorAPI{
		fs:  fs,
		val: validation.NewValidator(),
	}
}

// ListDir lists directory contents
func (api *EditorAPI) ListDir(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		path = "."
	}

	// Validate path
	val := validation.NewValidator()
	val.ValidateFilePath(path, ".")
	if val.HasErrors() {
		c.JSON(http.StatusBadRequest, gin.H{"error": val.FirstError().Error()})
		return
	}

	files, err := api.fs.ListDir(context.Background(), path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, files)
}

// GetFile gets file content
func (api *EditorAPI) GetFile(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	// Validate path
	val := validation.NewValidator()
	val.ValidateFilePath(path, ".")
	if val.HasErrors() {
		c.JSON(http.StatusBadRequest, gin.H{"error": val.FirstError().Error()})
		return
	}

	content, err := api.fs.ReadFile(context.Background(), path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, content)
}

// CreateFile creates a new file
func (api *EditorAPI) CreateFile(c *gin.Context) {
	var req struct {
		Path    string `json:"path" binding:"required"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate
	val := validation.NewValidator()
	val.ValidateFilePath(req.Path, ".").
		ValidateFileName(filepath.Base(req.Path))
	if val.HasErrors() {
		c.JSON(http.StatusBadRequest, gin.H{"error": val.FirstError().Error()})
		return
	}

	if err := api.fs.CreateFile(context.Background(), req.Path, req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "file created", "path": req.Path})
}

// UpdateFile updates a file
func (api *EditorAPI) UpdateFile(c *gin.Context) {
	var req struct {
		Path    string `json:"path" binding:"required"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate path
	val := validation.NewValidator()
	val.ValidateFilePath(req.Path, ".")
	if val.HasErrors() {
		c.JSON(http.StatusBadRequest, gin.H{"error": val.FirstError().Error()})
		return
	}

	if err := api.fs.WriteFile(context.Background(), req.Path, req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "file updated", "path": req.Path})
}

// DeleteFile deletes a file
func (api *EditorAPI) DeleteFile(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	// Validate path
	val := validation.NewValidator()
	val.ValidateFilePath(path, ".")
	if val.HasErrors() {
		c.JSON(http.StatusBadRequest, gin.H{"error": val.FirstError().Error()})
		return
	}

	if err := api.fs.DeleteFile(context.Background(), path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "file deleted", "path": path})
}

// RenameFile renames a file
func (api *EditorAPI) RenameFile(c *gin.Context) {
	var req struct {
		OldPath string `json:"old_path" binding:"required"`
		NewPath string `json:"new_path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate paths
	val := validation.NewValidator()
	val.ValidateFilePath(req.OldPath, ".").
		ValidateFilePath(req.NewPath, ".")
	if val.HasErrors() {
		c.JSON(http.StatusBadRequest, gin.H{"error": val.FirstError().Error()})
		return
	}

	if err := api.fs.RenameFile(context.Background(), req.OldPath, req.NewPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "file renamed", "old_path": req.OldPath, "new_path": req.NewPath})
}

// CreateDirectory creates a directory
func (api *EditorAPI) CreateDirectory(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate path
	val := validation.NewValidator()
	val.ValidateFilePath(req.Path, ".")
	if val.HasErrors() {
		c.JSON(http.StatusBadRequest, gin.H{"error": val.FirstError().Error()})
		return
	}

	if err := api.fs.CreateDirectory(context.Background(), req.Path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "directory created", "path": req.Path})
}

// SearchFiles searches for files
func (api *EditorAPI) SearchFiles(c *gin.Context) {
	query := c.Query("q")
	path := c.Query("path")
	if path == "" {
		path = "."
	}

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter is required"})
		return
	}

	// TODO: Implement file search
	c.JSON(http.StatusOK, gin.H{"results": []interface{}{}})
}

// GetProjectStructure gets project structure
func (api *EditorAPI) GetProjectStructure(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		path = "."
	}

	// TODO: Implement project structure analysis
	c.JSON(http.StatusOK, gin.H{
		"root":  path,
		"files": []interface{}{},
		"dirs":  []interface{}{},
		"total": 0,
		"stats": map[string]interface{}{},
	})
}

// RegisterEditorRoutes registers editor API routes
func RegisterEditorRoutes(router *gin.Engine, fs FileSystemManager) {
	api := NewEditorAPI(fs)

	editor := router.Group("/api/v1/editor")
	{
		// File operations
		editor.GET("/files/list", api.ListDir)
		editor.GET("/files/get", api.GetFile)
		editor.POST("/files/create", api.CreateFile)
		editor.POST("/files/update", api.UpdateFile)
		editor.DELETE("/files/delete", api.DeleteFile)
		editor.POST("/files/rename", api.RenameFile)

		// Directory operations
		editor.POST("/dirs/create", api.CreateDirectory)

		// Search and analysis
		editor.GET("/search", api.SearchFiles)
		editor.GET("/project-structure", api.GetProjectStructure)
	}
}
