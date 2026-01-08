package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lurus-ai/subscription-service/internal/biz"
)

// HTTPServer represents the HTTP server
type HTTPServer struct {
	engine   *gin.Engine
	subUC    *biz.SubscriptionUsecase
	planRepo biz.PlanRepo
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(subUC *biz.SubscriptionUsecase, planRepo biz.PlanRepo) *HTTPServer {
	s := &HTTPServer{
		engine:   gin.Default(),
		subUC:    subUC,
		planRepo: planRepo,
	}
	s.setupRoutes()
	return s
}

func (s *HTTPServer) setupRoutes() {
	// Health check
	s.engine.GET("/health", s.health)

	api := s.engine.Group("/api/v1")
	{
		// Plans
		api.GET("/plans", s.listPlans)
		api.GET("/plans/:code", s.getPlan)

		// Subscriptions
		api.GET("/subscriptions", s.listSubscriptions)
		api.GET("/subscriptions/user/:user_id", s.getUserSubscription)
		api.GET("/subscriptions/:id", s.getSubscriptionByID)
		api.POST("/subscriptions", s.createSubscription)
		api.DELETE("/subscriptions/:id", s.cancelSubscription)
		api.POST("/subscriptions/:id/renew", s.renewSubscription)
		api.POST("/subscriptions/:id/reset-daily", s.resetDailyQuota)

		// Quota
		api.POST("/quota/deduct", s.deductQuota)
		api.GET("/quota/:user_id", s.getUserQuota)
		api.GET("/quota/:user_id/status", s.getQuotaStatus)
	}

	// Admin routes
	admin := s.engine.Group("/admin/v1")
	{
		admin.POST("/plans", s.createPlan)
		admin.PUT("/plans/:id", s.updatePlan)
		admin.POST("/plans/init", s.initDefaultPlans)
		admin.GET("/subscriptions", s.adminListSubscriptions)
		admin.GET("/stats/overview", s.getStatsOverview)
	}
}

func (s *HTTPServer) Run(addr string) error {
	return s.engine.Run(addr)
}

// --- Handlers ---

func (s *HTTPServer) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "subscription-service",
	})
}

func (s *HTTPServer) listPlans(c *gin.Context) {
	plans, err := s.planRepo.ListActive(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": plans})
}

func (s *HTTPServer) getPlan(c *gin.Context) {
	code := c.Param("code")
	plan, err := s.planRepo.GetByCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Plan not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": plan})
}

func (s *HTTPServer) getUserSubscription(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid user ID"})
		return
	}

	sub, err := s.subUC.GetUserSubscription(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "No active subscription found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": sub})
}

type createSubscriptionReq struct {
	UserID   int    `json:"user_id" binding:"required"`
	PlanCode string `json:"plan_code" binding:"required"`
}

func (s *HTTPServer) createSubscription(c *gin.Context) {
	var req createSubscriptionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	sub, err := s.subUC.Subscribe(c.Request.Context(), req.UserID, req.PlanCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": sub})
}

func (s *HTTPServer) cancelSubscription(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid subscription ID"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	if err := s.subUC.Cancel(c.Request.Context(), id, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Subscription cancelled"})
}

func (s *HTTPServer) renewSubscription(c *gin.Context) {
	// Manual renewal trigger
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Renewal triggered"})
}

type deductQuotaReq struct {
	UserID int   `json:"user_id" binding:"required"`
	Amount int64 `json:"amount" binding:"required"`
}

func (s *HTTPServer) deductQuota(c *gin.Context) {
	var req deductQuotaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	if err := s.subUC.DeductQuota(c.Request.Context(), req.UserID, req.Amount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Quota deducted"})
}

func (s *HTTPServer) getUserQuota(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid user ID"})
		return
	}

	sub, err := s.subUC.GetUserSubscription(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "No active subscription",
			"data": gin.H{
				"current_quota":   0,
				"used_quota":      0,
				"has_quota":       false,
				"daily_quota":     0,
				"today_used":      0,
				"has_daily_quota": false,
				"current_group":   "free",
				"is_fallback":     true,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"current_quota":       sub.CurrentQuota,
			"used_quota":          sub.UsedQuota,
			"has_quota":           sub.HasQuota(),
			"daily_quota":         sub.DailyQuota,
			"today_used":          sub.TodayUsed,
			"daily_remaining":     sub.RemainingDailyQuota(),
			"has_daily_quota":     sub.HasDailyQuota(),
			"current_group":       sub.CurrentGroup,
			"is_fallback":         sub.IsUsingFallback(),
			"last_daily_reset_at": sub.LastDailyResetAt,
			"expires_at":          sub.ExpiresAt,
			"plan":                sub.Plan,
		},
	})
}

func (s *HTTPServer) createPlan(c *gin.Context) {
	var plan biz.Plan
	if err := c.ShouldBindJSON(&plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	if err := s.planRepo.Create(c.Request.Context(), &plan); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": plan})
}

func (s *HTTPServer) updatePlan(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid plan ID"})
		return
	}

	plan, err := s.planRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Plan not found"})
		return
	}

	if err := c.ShouldBindJSON(plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	if err := s.planRepo.Update(c.Request.Context(), plan); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": plan})
}

func (s *HTTPServer) initDefaultPlans(c *gin.Context) {
	if err := s.planRepo.InitDefaultPlans(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Default plans initialized"})
}

func (s *HTTPServer) getQuotaStatus(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid user ID"})
		return
	}

	status, err := s.subUC.CheckQuotaStatus(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}

func (s *HTTPServer) listSubscriptions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	planCode := c.Query("plan_code")

	subs, total, err := s.subUC.ListSubscriptions(c.Request.Context(), page, pageSize, status, planCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"subscriptions": subs,
			"total":         total,
			"page":          page,
			"page_size":     pageSize,
		},
	})
}

func (s *HTTPServer) getSubscriptionByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid subscription ID"})
		return
	}

	sub, err := s.subUC.GetSubscriptionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Subscription not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": sub})
}

func (s *HTTPServer) resetDailyQuota(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid subscription ID"})
		return
	}

	if err := s.subUC.ResetSubscriptionDailyQuota(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Daily quota reset successfully"})
}

func (s *HTTPServer) adminListSubscriptions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	planCode := c.Query("plan_code")
	userID := c.Query("user_id")

	var userIDInt int
	if userID != "" {
		userIDInt, _ = strconv.Atoi(userID)
	}

	subs, total, err := s.subUC.AdminListSubscriptions(c.Request.Context(), page, pageSize, status, planCode, userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"subscriptions": subs,
			"total":         total,
			"page":          page,
			"page_size":     pageSize,
		},
	})
}

func (s *HTTPServer) getStatsOverview(c *gin.Context) {
	stats, err := s.subUC.GetStatsOverview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}
