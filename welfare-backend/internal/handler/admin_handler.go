package handler

import (
	"net/http"
	"strconv"
	"time"

	"welfare-backend/internal/middleware"
	"welfare-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	checkin *service.CheckinService
}

func NewAdminHandler(checkin *service.CheckinService) *AdminHandler {
	return &AdminHandler{checkin: checkin}
}

func (h *AdminHandler) GetCheckinConfig(c *gin.Context) {
	cfg, err := h.checkin.AdminGetCampaign(c.Request.Context())
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	Success(c, cfg)
}

func (h *AdminHandler) UpdateCheckinConfig(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req service.UpdateCampaignInput
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	updated, err := h.checkin.AdminUpdateCampaign(c.Request.Context(), claims.LinuxDOSubject, req)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}
	Success(c, updated)
}

func (h *AdminHandler) ListCheckinRecords(c *gin.Context) {
	page := parseInt(c.DefaultQuery("page", "1"), 1)
	pageSize := parseInt(c.DefaultQuery("page_size", "20"), 20)
	userID := int64(parseInt(c.DefaultQuery("user", "0"), 0))
	items, total, err := h.checkin.AdminListRecords(c.Request.Context(), service.CheckinRecordFilter{
		Page:     page,
		PageSize: pageSize,
		Status:   c.Query("status"),
		Date:     c.Query("date"),
		UserID:   userID,
	})
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	Success(c, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}

func (h *AdminHandler) ListRiskBlocks(c *gin.Context) {
	items, err := h.checkin.AdminListBlocks(c.Request.Context())
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	Success(c, items)
}

func (h *AdminHandler) CreateRiskBlock(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		BlockType  string  `json:"block_type"`
		BlockValue string  `json:"block_value"`
		Reason     string  `json:"reason"`
		ExpiresAt  *string `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			Error(c, http.StatusBadRequest, "invalid expires_at")
			return
		}
		expiresAt = &t
	}
	created, err := h.checkin.AdminCreateBlock(c.Request.Context(), claims.LinuxDOSubject, service.RiskBlockInput{
		BlockType:  req.BlockType,
		BlockValue: req.BlockValue,
		Reason:     req.Reason,
		ExpiresAt:  expiresAt,
	})
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}
	Success(c, created)
}

func (h *AdminHandler) DeleteRiskBlock(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.checkin.AdminDeleteBlock(c.Request.Context(), claims.LinuxDOSubject, uint(id64)); err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}
	Success(c, gin.H{"ok": true})
}
