package ide

import (
	"embed"
	"errors"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/envconfig"
)

//go:embed static/*
var staticAssets embed.FS

type API struct {
	service *Service
	initErr error
}

func RegisterRoutes(router *gin.Engine) {
	cwd, err := os.Getwd()
	var svc *Service
	if err == nil {
		svc, err = NewService(cwd)
	}

	api := &API{service: svc, initErr: err}
	assets, _ := fs.Sub(staticAssets, "static")

	router.GET("/ide", api.index)
	router.GET("/ide/", api.index)
	router.StaticFS("/ide/assets", http.FS(assets))

	group := router.Group("/api/v1/ide", localOnlyIDE())
	{
		group.GET("/health", api.health)
		group.GET("/workspace", api.workspace)
		group.POST("/workspace/open", api.openWorkspace)
		group.GET("/tree", api.tree)
		group.GET("/files/list", api.listDir)
		group.GET("/files/read", api.readFile)
		group.PUT("/files/write", api.writeFile)
		group.POST("/files/create", api.createFile)
		group.POST("/files/delete", api.deleteFile)
		group.GET("/search", api.search)

		group.GET("/settings", api.settings)
		group.PUT("/settings", api.updateSettings)
		group.GET("/models", api.models)

		group.POST("/agent/run", api.runAgent)
		group.POST("/changes/apply", api.applyChanges)

		group.POST("/tools/read_file", api.toolReadFile)
		group.POST("/tools/write_file", api.toolWriteFile)
		group.POST("/tools/create_file", api.toolCreateFile)
		group.POST("/tools/delete_file", api.toolDeleteFile)
		group.POST("/tools/list_directory", api.toolListDirectory)
		group.POST("/tools/search_project", api.toolSearchProject)
		group.POST("/tools/apply_patch", api.toolApplyPatch)
	}
}

func (api *API) index(c *gin.Context) {
	data, err := staticAssets.ReadFile("static/index.html")
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

func (api *API) ready(c *gin.Context) bool {
	if api.initErr != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": api.initErr.Error()})
		return false
	}
	return true
}

func (api *API) workspace(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	c.JSON(http.StatusOK, api.service.WorkspaceState())
}

func (api *API) health(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	c.JSON(http.StatusOK, api.service.Health())
}

func (api *API) openWorkspace(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := api.service.OpenWorkspace(req.Path); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, api.service.WorkspaceState())
}

func (api *API) tree(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	depth, _ := strconv.Atoi(c.DefaultQuery("depth", "3"))
	nodes, err := api.service.workspace.Tree(c.Request.Context(), c.DefaultQuery("path", "."), depth)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"nodes": nodes})
}

func (api *API) listDir(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	nodes, err := api.service.workspace.ListDir(c.Request.Context(), c.DefaultQuery("path", "."))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"nodes": nodes})
}

func (api *API) readFile(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	file, err := api.service.workspace.ReadFile(c.Request.Context(), c.Query("path"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, file)
}

func (api *API) writeFile(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	before := ""
	if current, err := api.service.workspace.ReadFile(c.Request.Context(), req.Path); err == nil {
		before = current.Content
	}
	if err := api.service.workspace.WriteFile(c.Request.Context(), req.Path, req.Content); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, BuildChange(req.Path, actionUpdate, before, req.Content))
}

func (api *API) createFile(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := api.service.workspace.CreateFile(c.Request.Context(), req.Path, req.Content); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, BuildChange(req.Path, actionCreate, "", req.Content))
}

func (api *API) deleteFile(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	var req struct {
		Path    string `json:"path"`
		Confirm bool   `json:"confirm"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !req.Confirm {
		c.JSON(http.StatusBadRequest, gin.H{"error": "delete requires confirmation"})
		return
	}
	before := ""
	if current, err := api.service.workspace.ReadFile(c.Request.Context(), req.Path); err == nil {
		before = current.Content
	}
	if err := api.service.workspace.DeleteFile(c.Request.Context(), req.Path); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, BuildChange(req.Path, actionDelete, before, ""))
}

func (api *API) search(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	results, err := api.service.workspace.Search(c.Request.Context(), c.Query("q"), c.DefaultQuery("path", "."))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}

func (api *API) settings(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	c.JSON(http.StatusOK, api.service.Settings())
}

func (api *API) updateSettings(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	var req Settings
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, api.service.UpdateSettings(req))
}

func (api *API) models(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	models, err := api.service.ListModels(c.Request.Context(), c.DefaultQuery("provider", "local"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "models": []string{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"models": models})
}

func (api *API) runAgent(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	var req AgentRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := api.service.RunAgent(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (api *API) applyChanges(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	var req ApplyChangesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := api.service.ApplyChanges(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (api *API) toolReadFile(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if !bindTool(c, &req) {
		return
	}
	file, err := api.service.workspace.ReadFile(c.Request.Context(), req.Path)
	respondTool(c, "read_file", req.Path, file, err)
}

func (api *API) toolWriteFile(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if !bindTool(c, &req) {
		return
	}
	before := ""
	action := actionUpdate
	if current, err := api.service.workspace.ReadFile(c.Request.Context(), req.Path); err == nil {
		before = current.Content
	} else {
		action = actionCreate
	}
	change := BuildChange(req.Path, action, before, req.Content)
	respondTool(c, "write_file", req.Path, change, nil)
}

func (api *API) toolCreateFile(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if !bindTool(c, &req) {
		return
	}
	change := BuildChange(req.Path, actionCreate, "", req.Content)
	respondTool(c, "create_file", req.Path, change, nil)
}

func (api *API) toolDeleteFile(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	var req struct {
		Path    string `json:"path"`
		Confirm bool   `json:"confirm"`
	}
	if !bindTool(c, &req) {
		return
	}
	if !req.Confirm {
		respondTool(c, "delete_file", req.Path, nil, errors.New("delete requires confirmation"))
		return
	}
	before := ""
	if current, err := api.service.workspace.ReadFile(c.Request.Context(), req.Path); err == nil {
		before = current.Content
	}
	change := BuildChange(req.Path, actionDelete, before, "")
	respondTool(c, "delete_file", req.Path, change, nil)
}

func (api *API) toolListDirectory(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if !bindTool(c, &req) {
		return
	}
	nodes, err := api.service.workspace.ListDir(c.Request.Context(), req.Path)
	respondTool(c, "list_directory", req.Path, nodes, err)
}

func (api *API) toolSearchProject(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	var req struct {
		Query string `json:"query"`
		Path  string `json:"path"`
	}
	if !bindTool(c, &req) {
		return
	}
	results, err := api.service.workspace.Search(c.Request.Context(), req.Query, req.Path)
	respondTool(c, "search_project", req.Path, results, err)
}

func (api *API) toolApplyPatch(c *gin.Context) {
	if !api.ready(c) {
		return
	}
	var req ApplyChangesRequest
	if !bindTool(c, &req) {
		return
	}
	changes := make([]FileChange, 0, len(req.Changes))
	for _, change := range req.Changes {
		changes = append(changes, api.service.previewChange(c.Request.Context(), change))
	}
	respondTool(c, "apply_patch", "", gin.H{"changes": changes}, nil)
}

func bindTool(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, ToolResult{Error: err.Error()})
		return false
	}
	return true
}

func respondTool(c *gin.Context, tool string, path string, output any, err error) {
	status := http.StatusOK
	result := ToolResult{Tool: tool, Path: path, Output: output}
	if err != nil {
		status = http.StatusBadRequest
		result.Error = err.Error()
	}
	c.JSON(status, result)
}

func localOnlyIDE() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.EqualFold(envconfig.Var("OLLAMA_IDE_ALLOW_REMOTE"), "true") {
			c.Next()
			return
		}

		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err == nil {
			if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "IDE filesystem API is limited to loopback clients"})
	}
}
