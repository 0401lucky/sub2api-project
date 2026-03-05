package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"welfare-backend/internal/middleware"
	"welfare-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type CheckinHandler struct {
	checkin *service.CheckinService
}

func NewCheckinHandler(checkin *service.CheckinService) *CheckinHandler {
	return &CheckinHandler{checkin: checkin}
}

func (h *CheckinHandler) Status(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	status, err := h.checkin.GetStatus(c.Request.Context(), service.CheckinActor{
		Sub2APIUserID:  claims.Sub2APIUserID,
		LinuxDOSubject: claims.LinuxDOSubject,
	}, c.ClientIP())
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	Success(c, status)
}

func (h *CheckinHandler) Daily(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	result, err := h.checkin.Checkin(c.Request.Context(), service.CheckinActor{
		Sub2APIUserID:  claims.Sub2APIUserID,
		LinuxDOSubject: claims.LinuxDOSubject,
	}, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCheckinDisabled):
			Error(c, http.StatusForbidden, "签到暂未开放")
		case errors.Is(err, service.ErrCheckinBlocked):
			Error(c, http.StatusForbidden, err.Error())
		case errors.Is(err, service.ErrAlreadyChecked):
			Success(c, result)
		case errors.Is(err, service.ErrCheckinBusy):
			Error(c, http.StatusConflict, "签到处理中，请稍后重试")
		default:
			log.Printf("checkin add balance failed: user_id=%d subject=%s err=%v", claims.Sub2APIUserID, claims.LinuxDOSubject, err)
			Error(c, http.StatusBadGateway, "发放额度失败: "+err.Error())
		}
		return
	}
	Success(c, result)
}

func (h *CheckinHandler) History(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	page := parseInt(c.DefaultQuery("page", "1"), 1)
	pageSize := parseInt(c.DefaultQuery("page_size", "20"), 20)
	items, total, err := h.checkin.ListUserHistory(c.Request.Context(), claims.Sub2APIUserID, page, pageSize)
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	Success(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func parseInt(raw string, fallback int) int {
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
