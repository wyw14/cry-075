package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-075/internal/application"
	"github.com/wyw14/cry-075/internal/domain"
	"github.com/wyw14/cry-075/internal/middleware"
)

func (a API) createCampaign(c *gin.Context) {
	var input struct {
		Name        string    `json:"name" binding:"required,min=2,max=120"`
		Season      string    `json:"season" binding:"required,max=80"`
		Environment string    `json:"environment" binding:"required,oneof=development staging production"`
		StartsAt    time.Time `json:"starts_at" binding:"required"`
		EndsAt      time.Time `json:"ends_at" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, &domain.ValidationError{Violations: []domain.FieldViolation{{Field: "body", Reason: err.Error()}}})
		return
	}
	value, err := a.Campaigns.Create(c, application.CreateCampaignCommand{Name: input.Name, Season: input.Season, Environment: input.Environment, Window: domain.TimeWindow{StartsAt: input.StartsAt, EndsAt: input.EndsAt}, Actor: middleware.GetActor(c), IdempotencyKey: c.GetHeader("Idempotency-Key"), RequestID: middleware.GetRequestID(c)})
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, value)
}
func (a API) listCampaigns(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	filters := map[string]string{}
	for _, key := range []string{"status", "environment"} {
		if value := c.Query(key); value != "" {
			filters[key] = value
		}
	}
	result, err := a.Campaigns.List(c, domain.ListQuery{Page: page, PerPage: perPage, Sort: c.DefaultQuery("sort", "updated_at:desc"), Filters: filters})
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
func (a API) bindCampaign(c *gin.Context) {
	var input struct {
		PackageID   domain.ID `json:"package_id" binding:"required"`
		PlacementID domain.ID `json:"placement_id" binding:"required"`
		AudienceID  domain.ID `json:"audience_id" binding:"required"`
		Version     int64     `json:"version" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, err)
		return
	}
	result, err := a.Campaigns.Bind(c, application.BindCampaignCommand{CampaignID: domain.ID(c.Param("id")), PackageID: input.PackageID, PlacementID: input.PlacementID, AudienceID: input.AudienceID, ExpectedVersion: input.Version, Actor: middleware.GetActor(c), RequestID: middleware.GetRequestID(c)})
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
func (a API) submitReview(c *gin.Context) {
	version, err := readVersion(c)
	if err != nil {
		fail(c, err)
		return
	}
	result, err := a.Review.Submit(c, domain.ID(c.Param("id")), version, middleware.GetActor(c), middleware.GetRequestID(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
func (a API) reviewCampaign(c *gin.Context) {
	var input struct {
		Decision domain.ApprovalDecision `json:"decision" binding:"required,oneof=approve reject"`
		Comment  string                  `json:"comment" binding:"max=1000"`
		Version  int64                   `json:"version" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, err)
		return
	}
	result, err := a.Review.Decide(c, domain.ID(c.Param("id")), input.Decision, input.Comment, input.Version, middleware.GetActor(c), middleware.GetRequestID(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
func (a API) impact(c *gin.Context) {
	result, err := a.Preview.Analyze(c, domain.ID(c.Param("id")))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
func readVersion(c *gin.Context) (int64, error) {
	var input struct {
		Version int64 `json:"version" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		return 0, err
	}
	return input.Version, nil
}
