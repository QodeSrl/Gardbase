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
