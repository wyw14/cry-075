package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-075/internal/application"
	"github.com/wyw14/cry-075/internal/domain"
	"github.com/wyw14/cry-075/internal/middleware"
)

type API struct {
	Campaigns    application.CampaignService
	Catalog      application.CatalogService
	Review       application.ReviewService
	Preview      application.PreviewService
	Invalidation application.InvalidationService
	Releases     application.ReleaseService
	Scheduling   application.ScheduleService
	Operations   application.OperationsService
	Attachments  application.AttachmentService
	Ready        func(*gin.Context) error
}

func NewRouter(api API, allowedOrigins []string, timeout time.Duration, webDir string) *gin.Engine {
	router := gin.New()
	router.Use(middleware.RequestID(), gin.Recovery(), middleware.SecurityHeaders(), middleware.CORS(allowedOrigins), middleware.Timeout(timeout))
	router.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	router.GET("/readyz", func(c *gin.Context) {
		if err := api.Ready(c); err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	v1 := router.Group("/api/v1", middleware.Actor())
	v1.POST("/campaigns", api.createCampaign)
	v1.GET("/campaigns", api.listCampaigns)
	v1.PATCH("/campaigns/:id/composition", api.bindCampaign)
	v1.POST("/campaigns/:id/submit", api.submitReview)
	v1.POST("/campaigns/:id/review", api.reviewCampaign)
	v1.GET("/campaigns/:id/impact", api.impact)
	v1.POST("/campaigns/:id/snapshots", api.snapshot)
	v1.POST("/campaigns/:id/schedules", api.schedule)
	v1.POST("/previews", api.preview)
	v1.POST("/assets/:id/takedown", api.takedown)
	v1.POST("/snapshots/:id/publish", api.publish)
	v1.POST("/snapshots/:id/rollback", api.rollback)
	v1.POST("/attachments", api.uploadAttachment)
	v1.POST("/audit/export", api.exportAudit)
	router.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet || strings.HasPrefix(c.Request.URL.Path, "/api/") {
			fail(c, domain.ErrNotFound)
			return
		}
		clean := filepath.Clean(strings.TrimPrefix(c.Request.URL.Path, "/"))
		if clean != "." && !strings.HasPrefix(clean, "..") {
			candidate := filepath.Join(webDir, clean)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				c.File(candidate)
				return
			}
		}
		index := filepath.Join(webDir, "index.html")
		if _, err := os.Stat(index); err != nil {
			fail(c, domain.ErrNotFound)
			return
		}
		c.File(index)
	})
	return router
}
