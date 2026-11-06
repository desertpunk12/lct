// Package httpapi exposes the leads API and the embedded frontend over HTTP.
package httpapi

import (
	"errors"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"leadcapture/api/internal/db"
	"leadcapture/api/internal/lead"
)

// New returns a handler serving the API under /api and the frontend files
// (rooted at index.html) everywhere else.
func New(store *db.Store, frontend fs.FS) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/api/leads", func(c *gin.Context) {
		leads, err := store.ListLeads(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load leads"})
			return
		}
		c.JSON(http.StatusOK, leads)
	})

	router.POST("/api/leads", func(c *gin.Context) {
		var input lead.Input
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
			return
		}
		created, err := store.CreateLead(c.Request.Context(), input)
		if err != nil {
			var fields lead.FieldErrors
			switch {
			case errors.Is(err, db.ErrDuplicateEmail):
				c.JSON(http.StatusConflict, gin.H{
					"errors": lead.FieldErrors{"email": "a lead with this email address already exists"},
				})
			case errors.As(err, &fields):
				c.JSON(http.StatusUnprocessableEntity, gin.H{"errors": fields})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save lead"})
			}
			return
		}
		c.JSON(http.StatusCreated, created)
	})

	router.NoRoute(func(c *gin.Context) {
		name := strings.TrimPrefix(c.Request.URL.Path, "/")
		if strings.HasPrefix(name, "api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if name == "" {
			name = "index.html"
		}
		if _, err := fs.Stat(frontend, name); err != nil {
			// Unknown paths serve the app shell so client-side routes resolve.
			name = "index.html"
		}
		if name == "index.html" {
			// Served directly: http.FileServer would redirect /index.html to ./.
			index, err := fs.ReadFile(frontend, name)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "frontend is unavailable"})
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", index)
			return
		}
		c.FileFromFS(name, http.FS(frontend))
	})
	return router
}
