package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"welfare-backend/internal/middleware"
	"welfare-backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
		log.Printf("admin get checkin config failed: err=%v", err)
		Error(c, http.StatusInternalServerError, "获取签到配置失败")
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
		log.Printf("admin update checkin config failed: admin=%s err=%v", claims.LinuxDOSubject, err)
		if isBadRequestError(err) {
			Error(c, http.StatusBadRequest, "配置参数不合法")
			return
		}
		Error(c, http.StatusInternalServerError, "保存配置失败")
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
		log.Printf("admin list checkin records failed: err=%v", err)
		Error(c, http.StatusInternalServerError, "获取签到记录失败")
		return
	}
	Success(c, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}

func (h *AdminHandler) ListRiskBlocks(c *gin.Context) {
	items, err := h.checkin.AdminListBlocks(c.Request.Context())
	if err != nil {
		log.Printf("admin list risk blocks failed: err=%v", err)
		Error(c, http.StatusInternalServerError, "获取封禁列表失败")
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
		log.Printf("admin create risk block failed: admin=%s err=%v", claims.LinuxDOSubject, err)
		if isBadRequestError(err) {
			Error(c, http.StatusBadRequest, "封禁参数不合法")
			return
		}
		if strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			Error(c, http.StatusConflict, "该封禁规则已存在")
			return
		}
		Error(c, http.StatusInternalServerError, "新增封禁失败")
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
		log.Printf("admin delete risk block failed: admin=%s id=%d err=%v", claims.LinuxDOSubject, id64, err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Error(c, http.StatusNotFound, "封禁记录不存在")
			return
		}
		Error(c, http.StatusBadRequest, "删除封禁失败")
		return
	}
	Success(c, gin.H{"ok": true})
}

func isBadRequestError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalid") ||
		strings.Contains(msg, "must be") ||
		strings.Contains(msg, "required") ||
		strings.Contains(msg, "timezone")
}
