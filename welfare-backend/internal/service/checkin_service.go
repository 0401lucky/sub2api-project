package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"welfare-backend/internal/model"
	"welfare-backend/internal/util"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrCheckinDisabled = errors.New("checkin disabled")
	ErrAlreadyChecked  = errors.New("already checked in today")
	ErrCheckinBlocked  = errors.New("checkin blocked")
	ErrCheckinBusy     = errors.New("checkin processing")
)

type CheckinService struct {
	db      *gorm.DB
	sub2api *Sub2APIClient
}

type CheckinActor struct {
	Sub2APIUserID  int64
	LinuxDOSubject string
}

type CheckinStatus struct {
	Date        string          `json:"date"`
	CanCheckin  bool            `json:"can_checkin"`
	CheckedIn   bool            `json:"checked_in"`
	GrantStatus string          `json:"grant_status,omitempty"`
	Amount      float64         `json:"amount,omitempty"`
	Campaign    CheckinCampaign `json:"campaign"`
	Blocked     bool            `json:"blocked"`
}

type CheckinCampaign struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Enabled     bool    `json:"enabled"`
	Timezone    string  `json:"timezone"`
	RewardMode  string  `json:"reward_mode"`
	RewardMin   float64 `json:"reward_min"`
	RewardMax   float64 `json:"reward_max"`
	RewardScale int     `json:"reward_scale"`
	MaxPerDay   int     `json:"max_per_day"`
}

type CheckinResult struct {
	Date      string  `json:"date"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	GrantID   uint    `json:"grant_id"`
	Message   string  `json:"message"`
	NoteToken string  `json:"-"`
}

type CheckinHistoryItem struct {
	ID          uint       `json:"id"`
	CheckinDate string     `json:"checkin_date"`
	Amount      float64    `json:"amount"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	AppliedAt   *time.Time `json:"applied_at,omitempty"`
}

type CheckinRecordFilter struct {
	Page     int
	PageSize int
	Status   string
	Date     string
	UserID   int64
}

type UpdateCampaignInput struct {
	Enabled     *bool    `json:"enabled"`
	RewardMin   *float64 `json:"reward_min"`
	RewardMax   *float64 `json:"reward_max"`
	RewardScale *int     `json:"reward_scale"`
	Timezone    *string  `json:"timezone"`
}

type RiskBlockInput struct {
	BlockType  string     `json:"block_type"`
	BlockValue string     `json:"block_value"`
	Reason     string     `json:"reason"`
	ExpiresAt  *time.Time `json:"expires_at"`
}

func NewCheckinService(db *gorm.DB, sub2api *Sub2APIClient) *CheckinService {
	return &CheckinService{db: db, sub2api: sub2api}
}

func (s *CheckinService) GetStatus(ctx context.Context, actor CheckinActor, ip string) (*CheckinStatus, error) {
	campaign, err := s.getCampaign(ctx)
	if err != nil {
		return nil, err
	}
	date, _, err := s.currentDate(campaign.Timezone)
	if err != nil {
		return nil, err
	}
	blocked, _, err := s.isBlocked(ctx, actor, ip)
	if err != nil {
		return nil, err
	}
	out := &CheckinStatus{
		Date:       date,
		CanCheckin: campaign.Enabled && !blocked,
		CheckedIn:  false,
		Blocked:    blocked,
		Campaign:   toCampaignDTO(campaign),
	}

	var grant model.CheckinGrant
	err = s.db.WithContext(ctx).
		Where("campaign_id = ? AND sub2_api_user_id = ? AND checkin_date = ?", campaign.ID, actor.Sub2APIUserID, date).
		First(&grant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return out, nil
		}
		return nil, err
	}
	if grant.Status != model.GrantStatusSuccess {
		reconciled, err := s.reconcileGrantFromSub2API(ctx, &grant)
		if err != nil {
			return nil, err
		}
		if reconciled {
			out.GrantStatus = model.GrantStatusSuccess
			out.Amount = grant.Amount
			out.CheckedIn = true
			out.CanCheckin = false
			return out, nil
		}
	}
	out.GrantStatus = grant.Status
	out.Amount = grant.Amount
	out.CheckedIn = grant.Status == model.GrantStatusSuccess
	out.CanCheckin = out.CanCheckin && !out.CheckedIn && grant.Status != model.GrantStatusProcessing
	return out, nil
}

func (s *CheckinService) Checkin(ctx context.Context, actor CheckinActor, ip, userAgent string) (*CheckinResult, error) {
	campaign, err := s.getCampaign(ctx)
	if err != nil {
		return nil, err
	}
	if !campaign.Enabled {
		return nil, ErrCheckinDisabled
	}
	blocked, reason, err := s.isBlocked(ctx, actor, ip)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, fmt.Errorf("%w: %s", ErrCheckinBlocked, reason)
	}
	date, now, err := s.currentDate(campaign.Timezone)
	if err != nil {
		return nil, err
	}
	var preGrant model.CheckinGrant
	preErr := s.db.WithContext(ctx).
		Where("campaign_id = ? AND sub2_api_user_id = ? AND checkin_date = ?", campaign.ID, actor.Sub2APIUserID, date).
		First(&preGrant).Error
	if preErr == nil {
		if preGrant.Status == model.GrantStatusSuccess {
			return &CheckinResult{
				Date:    date,
				Amount:  preGrant.Amount,
				Status:  model.GrantStatusSuccess,
				GrantID: preGrant.ID,
				Message: "今日已签到",
			}, ErrAlreadyChecked
		}
		reconciled, err := s.reconcileGrantFromSub2API(ctx, &preGrant)
		if err != nil {
			return nil, err
		}
		if reconciled {
			return &CheckinResult{
				Date:    date,
				Amount:  preGrant.Amount,
				Status:  model.GrantStatusSuccess,
				GrantID: preGrant.ID,
				Message: "今日已签到",
			}, ErrAlreadyChecked
		}
	}

	var reserved model.CheckinGrant
	reserveErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.CheckinGrant
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("campaign_id = ? AND sub2_api_user_id = ? AND checkin_date = ?", campaign.ID, actor.Sub2APIUserID, date).
			First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		amount, err := util.RandomDecimalInRange(campaign.RewardMin, campaign.RewardMax, campaign.RewardScale)
		if err != nil {
			return err
		}
		idempotencyKey, err := util.RandomToken(24)
		if err != nil {
			return err
		}
		noteToken := buildDailyNoteToken(campaign.Code, actor.Sub2APIUserID, date)
		hash := util.SHA256String(userAgent)
		if len(hash) > 64 {
			hash = hash[:64]
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			reserved = model.CheckinGrant{
				CampaignID:     campaign.ID,
				Sub2APIUserID:  actor.Sub2APIUserID,
				CheckinDate:    date,
				Amount:         amount,
				Status:         model.GrantStatusProcessing,
				IdempotencyKey: idempotencyKey,
				NoteToken:      noteToken,
				AttemptCount:   1,
				RequestIP:      ip,
				UserAgentHash:  hash,
			}
			if err := tx.Create(&reserved).Error; err != nil {
				return err
			}
			return nil
		}

		switch existing.Status {
		case model.GrantStatusSuccess:
			reserved = existing
			return ErrAlreadyChecked
		case model.GrantStatusProcessing:
			if time.Since(existing.UpdatedAt) <= 2*time.Minute {
				reserved = existing
				return ErrCheckinBusy
			}
		}

		existing.Amount = amount
		existing.Status = model.GrantStatusProcessing
		existing.IdempotencyKey = idempotencyKey
		if strings.TrimSpace(existing.NoteToken) == "" {
			existing.NoteToken = noteToken
		}
		existing.AttemptCount = existing.AttemptCount + 1
		existing.RequestIP = ip
		existing.UserAgentHash = hash
		existing.LastError = ""
		existing.AppliedAt = nil
		existing.UpdatedAt = now
		if err := tx.Save(&existing).Error; err != nil {
			return err
		}
		reserved = existing
		return nil
	})
	if reserveErr != nil {
		if errors.Is(reserveErr, ErrAlreadyChecked) {
			return &CheckinResult{
				Date:    date,
				Amount:  reserved.Amount,
				Status:  model.GrantStatusSuccess,
				GrantID: reserved.ID,
				Message: "今日已签到",
			}, ErrAlreadyChecked
		}
		if errors.Is(reserveErr, ErrCheckinBusy) {
			return &CheckinResult{Date: date, Status: model.GrantStatusProcessing, GrantID: reserved.ID, Message: "签到处理中"}, ErrCheckinBusy
		}
		return nil, reserveErr
	}

	noteText := fmt.Sprintf("welfare_checkin:%s", reserved.NoteToken)
	found, err := s.sub2api.HasBalanceRecordByNoteToken(ctx, actor.Sub2APIUserID, reserved.NoteToken)
	if err == nil && found {
		appliedAt := time.Now()
		if err := s.db.WithContext(ctx).Model(&model.CheckinGrant{}).
			Where("id = ?", reserved.ID).
			Updates(map[string]interface{}{"status": model.GrantStatusSuccess, "applied_at": &appliedAt, "last_error": "", "updated_at": time.Now()}).Error; err != nil {
			return nil, err
		}
		return &CheckinResult{Date: date, Amount: reserved.Amount, Status: model.GrantStatusSuccess, GrantID: reserved.ID, Message: "今日已签到"}, ErrAlreadyChecked
	}
	addErr := s.sub2api.AddBalance(ctx, actor.Sub2APIUserID, reserved.Amount, noteText)
	if addErr != nil {
		reconciled := false
		found, err := s.sub2api.HasBalanceRecordByNoteToken(ctx, actor.Sub2APIUserID, reserved.NoteToken)
		if err == nil && found {
			reconciled = true
		}
		if reconciled {
			appliedAt := time.Now()
			_ = s.db.WithContext(ctx).Model(&model.CheckinGrant{}).
				Where("id = ?", reserved.ID).
				Updates(map[string]interface{}{"status": model.GrantStatusSuccess, "applied_at": &appliedAt, "last_error": "", "updated_at": time.Now()}).Error
			return &CheckinResult{Date: date, Amount: reserved.Amount, Status: model.GrantStatusSuccess, GrantID: reserved.ID, Message: "签到成功"}, nil
		}

		_ = s.db.WithContext(ctx).Model(&model.CheckinGrant{}).
			Where("id = ?", reserved.ID).
			Updates(map[string]interface{}{"status": model.GrantStatusFailed, "last_error": addErr.Error(), "updated_at": time.Now()}).Error
		return nil, addErr
	}

	appliedAt := time.Now()
	if err := s.db.WithContext(ctx).Model(&model.CheckinGrant{}).
		Where("id = ?", reserved.ID).
		Updates(map[string]interface{}{"status": model.GrantStatusSuccess, "applied_at": &appliedAt, "last_error": "", "updated_at": time.Now()}).Error; err != nil {
		return nil, err
	}
	return &CheckinResult{Date: date, Amount: reserved.Amount, Status: model.GrantStatusSuccess, GrantID: reserved.ID, Message: "签到成功"}, nil
}

func (s *CheckinService) ListUserHistory(ctx context.Context, sub2apiUserID int64, page, pageSize int) ([]CheckinHistoryItem, int64, error) {
	page = normalizePage(page)
	pageSize = normalizePageSize(pageSize)
	query := s.db.WithContext(ctx).Model(&model.CheckinGrant{}).Where("sub2_api_user_id = ?", sub2apiUserID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.CheckinGrant
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]CheckinHistoryItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, CheckinHistoryItem{
			ID:          row.ID,
			CheckinDate: row.CheckinDate,
			Amount:      row.Amount,
			Status:      row.Status,
			CreatedAt:   row.CreatedAt,
			AppliedAt:   row.AppliedAt,
		})
	}
	return out, total, nil
}

func (s *CheckinService) AdminGetCampaign(ctx context.Context) (*model.CheckinCampaign, error) {
	return s.getCampaign(ctx)
}

func (s *CheckinService) AdminUpdateCampaign(ctx context.Context, adminSubject string, in UpdateCampaignInput) (*model.CheckinCampaign, error) {
	campaign, err := s.getCampaign(ctx)
	if err != nil {
		return nil, err
	}
	before := fmt.Sprintf("enabled=%v,min=%.2f,max=%.2f,scale=%d,tz=%s", campaign.Enabled, campaign.RewardMin, campaign.RewardMax, campaign.RewardScale, campaign.Timezone)

	if in.Enabled != nil {
		campaign.Enabled = *in.Enabled
	}
	if in.RewardMin != nil {
		campaign.RewardMin = *in.RewardMin
	}
	if in.RewardMax != nil {
		campaign.RewardMax = *in.RewardMax
	}
	if in.RewardScale != nil {
		campaign.RewardScale = *in.RewardScale
	}
	if in.Timezone != nil {
		campaign.Timezone = strings.TrimSpace(*in.Timezone)
	}
	if campaign.RewardMin < 0 || campaign.RewardMax < campaign.RewardMin {
		return nil, errors.New("invalid reward range")
	}
	if campaign.RewardScale < 0 || campaign.RewardScale > 4 {
		return nil, errors.New("reward_scale must between 0 and 4")
	}
	if _, err := time.LoadLocation(campaign.Timezone); err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}
	campaign.UpdatedBy = adminSubject
	if err := s.db.WithContext(ctx).Save(campaign).Error; err != nil {
		return nil, err
	}
	after := fmt.Sprintf("enabled=%v,min=%.2f,max=%.2f,scale=%d,tz=%s", campaign.Enabled, campaign.RewardMin, campaign.RewardMax, campaign.RewardScale, campaign.Timezone)
	_ = s.writeAudit(ctx, adminSubject, "update_campaign", "checkin_campaign", fmt.Sprintf("%d", campaign.ID), before, after)
	return campaign, nil
}

func (s *CheckinService) AdminListRecords(ctx context.Context, filter CheckinRecordFilter) ([]model.CheckinGrant, int64, error) {
	filter.Page = normalizePage(filter.Page)
	filter.PageSize = normalizePageSize(filter.PageSize)

	query := s.db.WithContext(ctx).Model(&model.CheckinGrant{})
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Date != "" {
		query = query.Where("checkin_date = ?", filter.Date)
	}
	if filter.UserID > 0 {
		query = query.Where("sub2_api_user_id = ?", filter.UserID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.CheckinGrant
	if err := query.Order("id DESC").Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *CheckinService) AdminListBlocks(ctx context.Context) ([]model.RiskBlock, error) {
	var rows []model.RiskBlock
	if err := s.db.WithContext(ctx).Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *CheckinService) AdminCreateBlock(ctx context.Context, adminSubject string, in RiskBlockInput) (*model.RiskBlock, error) {
	in.BlockType = strings.TrimSpace(in.BlockType)
	in.BlockValue = strings.TrimSpace(in.BlockValue)
	in.Reason = strings.TrimSpace(in.Reason)
	if in.BlockType == "" || in.BlockValue == "" || in.Reason == "" {
		return nil, errors.New("block_type, block_value, reason are required")
	}
	if in.BlockType != "user" && in.BlockType != "subject" && in.BlockType != "ip" {
		return nil, errors.New("block_type must be user/subject/ip")
	}
	record := model.RiskBlock{
		BlockType:  in.BlockType,
		BlockValue: in.BlockValue,
		Reason:     in.Reason,
		ExpiresAt:  in.ExpiresAt,
		CreatedBy:  adminSubject,
	}
	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, err
	}
	_ = s.writeAudit(ctx, adminSubject, "create_block", "risk_block", fmt.Sprintf("%d", record.ID), "", fmt.Sprintf("%s:%s", record.BlockType, record.BlockValue))
	return &record, nil
}

func (s *CheckinService) AdminDeleteBlock(ctx context.Context, adminSubject string, id uint) error {
	var block model.RiskBlock
	if err := s.db.WithContext(ctx).First(&block, id).Error; err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Delete(&model.RiskBlock{}, id).Error; err != nil {
		return err
	}
	_ = s.writeAudit(ctx, adminSubject, "delete_block", "risk_block", fmt.Sprintf("%d", id), fmt.Sprintf("%s:%s", block.BlockType, block.BlockValue), "")
	return nil
}

func (s *CheckinService) getCampaign(ctx context.Context) (*model.CheckinCampaign, error) {
	var campaign model.CheckinCampaign
	if err := s.db.WithContext(ctx).Where("code = ?", model.CampaignCodeDailyDefault).First(&campaign).Error; err != nil {
		return nil, err
	}
	return &campaign, nil
}

func (s *CheckinService) currentDate(tz string) (string, time.Time, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return "", time.Time{}, err
	}
	now := time.Now().In(loc)
	return now.Format("2006-01-02"), now, nil
}

func (s *CheckinService) isBlocked(ctx context.Context, actor CheckinActor, ip string) (bool, string, error) {
	candidates := [][2]string{
		{"user", fmt.Sprintf("%d", actor.Sub2APIUserID)},
		{"subject", actor.LinuxDOSubject},
		{"ip", ip},
	}
	now := time.Now()
	for _, item := range candidates {
		if strings.TrimSpace(item[1]) == "" {
			continue
		}
		var block model.RiskBlock
		err := s.db.WithContext(ctx).
			Where("block_type = ? AND block_value = ?", item[0], item[1]).
			Where("expires_at IS NULL OR expires_at > ?", now).
			First(&block).Error
		if err == nil {
			return true, block.Reason, nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return false, "", err
		}
	}
	return false, "", nil
}

func (s *CheckinService) writeAudit(ctx context.Context, adminSubject, action, targetType, targetID, before, after string) error {
	row := model.AdminAuditLog{
		AdminSubject: adminSubject,
		Action:       action,
		TargetType:   targetType,
		TargetID:     targetID,
		BeforeJSON:   before,
		AfterJSON:    after,
	}
	return s.db.WithContext(ctx).Create(&row).Error
}

func toCampaignDTO(c *model.CheckinCampaign) CheckinCampaign {
	return CheckinCampaign{
		Code:        c.Code,
		Name:        c.Name,
		Enabled:     c.Enabled,
		Timezone:    c.Timezone,
		RewardMode:  c.RewardMode,
		RewardMin:   c.RewardMin,
		RewardMax:   c.RewardMax,
		RewardScale: c.RewardScale,
		MaxPerDay:   c.MaxPerDay,
	}
}

func normalizePage(page int) int {
	if page <= 0 {
		return 1
	}
	return page
}

func normalizePageSize(size int) int {
	if size <= 0 {
		return 20
	}
	if size > 200 {
		return 200
	}
	return size
}

func isTimeoutLike(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(strings.ToLower(err.Error()), "timeout") {
		return true
	}
	var nerr net.Error
	return errors.As(err, &nerr) && nerr.Timeout()
}

func buildDailyNoteToken(campaignCode string, userID int64, date string) string {
	return fmt.Sprintf("wfck:%s:%d:%s", strings.TrimSpace(campaignCode), userID, strings.TrimSpace(date))
}

func (s *CheckinService) reconcileGrantFromSub2API(ctx context.Context, grant *model.CheckinGrant) (bool, error) {
	if grant == nil || strings.TrimSpace(grant.NoteToken) == "" || grant.Status == model.GrantStatusSuccess {
		return false, nil
	}
	found, err := s.sub2api.HasBalanceRecordByNoteToken(ctx, grant.Sub2APIUserID, grant.NoteToken)
	if err != nil || !found {
		return false, nil
	}
	appliedAt := time.Now()
	if err := s.db.WithContext(ctx).Model(&model.CheckinGrant{}).
		Where("id = ?", grant.ID).
		Updates(map[string]interface{}{"status": model.GrantStatusSuccess, "applied_at": &appliedAt, "last_error": "", "updated_at": time.Now()}).Error; err != nil {
		return false, err
	}
	grant.Status = model.GrantStatusSuccess
	grant.AppliedAt = &appliedAt
	grant.LastError = ""
	return true, nil
}
