package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/qodesrl/gardbase/apps/api/internal/services"
)

type DependenciesHandler struct {
	Dependencies *services.Dependencies
}

// HandleListDependencies lists every dependency declared in the repository's
// go.mod, including its current version and whether a newer version is
// available on the Go module proxy.
func (h *DependenciesHandler) HandleListDependencies(c *gin.Context) {
	deps, err := h.Dependencies.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	updatable := 0
	for _, dep := range deps {
		if dep.Updatable {
			updatable++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total":        len(deps),
		"updatable":    updatable,
		"dependencies": deps,
	})
}

// updateDependencyRequest is the body accepted by HandleUpdateDependency.
type updateDependencyRequest struct {
	// Path is the module path of the dependency to upgrade (required).
	Path string `json:"path" binding:"required"`
	// Version is the target version to upgrade to. When empty the dependency is
	// upgraded to its latest available version on the module proxy.
	Version string `json:"version"`
}

// HandleUpdateDependency upgrades a single dependency declared in the
// repository's go.mod to the requested version (or to the latest available
// version when none is provided) and returns the applied change.
func (h *DependenciesHandler) HandleUpdateDependency(c *gin.Context) {
	var req updateDependencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.Dependencies.Update(c.Request.Context(), req.Path, req.Version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"updated": result,
	})
}
