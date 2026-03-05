package model

import "time"

type UserBinding struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	LinuxDOSubject  string    `gorm:"size:128;not null;uniqueIndex" json:"linuxdo_subject"`
	LinuxDOUsername string    `gorm:"size:128;not null" json:"linuxdo_username"`
	SyntheticEmail  string    `gorm:"size:256;not null;uniqueIndex" json:"synthetic_email"`
	Sub2APIUserID   int64     `gorm:"not null;index" json:"sub2api_user_id"`
	Sub2APIEmail    string    `gorm:"size:256;not null" json:"sub2api_email"`
	Status          string    `gorm:"size:32;not null;default:active" json:"status"`
	FirstLoginAt    time.Time `gorm:"not null" json:"first_login_at"`
	LastLoginAt     time.Time `gorm:"not null" json:"last_login_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CheckinCampaign struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Code           string    `gorm:"size:64;not null;uniqueIndex" json:"code"`
	Name           string    `gorm:"size:128;not null" json:"name"`
	Enabled        bool      `gorm:"not null;default:true" json:"enabled"`
	Timezone       string    `gorm:"size:64;not null" json:"timezone"`
	RewardMode     string    `gorm:"size:32;not null" json:"reward_mode"`
	RewardMin      float64   `gorm:"not null" json:"reward_min"`
	RewardMax      float64   `gorm:"not null" json:"reward_max"`
	RewardScale    int       `gorm:"not null;default:2" json:"reward_scale"`
	MaxPerDay      int       `gorm:"not null;default:1" json:"max_per_day"`
	RiskPolicyJSON string    `gorm:"type:text" json:"risk_policy_json"`
	CreatedBy      string    `gorm:"size:128;not null" json:"created_by"`
	UpdatedBy      string    `gorm:"size:128;not null" json:"updated_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CheckinGrant struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	CampaignID     uint       `gorm:"not null;uniqueIndex:udx_campaign_user_date,priority:1" json:"campaign_id"`
	Sub2APIUserID  int64      `gorm:"not null;index;uniqueIndex:udx_campaign_user_date,priority:2" json:"sub2api_user_id"`
	CheckinDate    string     `gorm:"size:10;not null;uniqueIndex:udx_campaign_user_date,priority:3" json:"checkin_date"`
	Amount         float64    `gorm:"not null" json:"amount"`
	Status         string     `gorm:"size:32;not null;index" json:"status"`
	IdempotencyKey string     `gorm:"size:128;not null;uniqueIndex" json:"idempotency_key"`
	NoteToken      string     `gorm:"size:128;not null;uniqueIndex" json:"note_token"`
	AttemptCount   int        `gorm:"not null;default:1" json:"attempt_count"`
	LastError      string     `gorm:"type:text" json:"last_error"`
	RequestIP      string     `gorm:"size:64" json:"request_ip"`
	UserAgentHash  string     `gorm:"size:128" json:"user_agent_hash"`
	AppliedAt      *time.Time `json:"applied_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type AdminAuditLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	AdminSubject string    `gorm:"size:128;not null" json:"admin_subject"`
	Action       string    `gorm:"size:64;not null;index" json:"action"`
	TargetType   string    `gorm:"size:64;not null" json:"target_type"`
	TargetID     string    `gorm:"size:128;not null" json:"target_id"`
	BeforeJSON   string    `gorm:"type:text" json:"before_json"`
	AfterJSON    string    `gorm:"type:text" json:"after_json"`
	CreatedAt    time.Time `json:"created_at"`
}

type RiskBlock struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	BlockType  string     `gorm:"size:32;not null;uniqueIndex:udx_block_type_value,priority:1" json:"block_type"`
	BlockValue string     `gorm:"size:128;not null;uniqueIndex:udx_block_type_value,priority:2" json:"block_value"`
	Reason     string     `gorm:"size:512;not null" json:"reason"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedBy  string     `gorm:"size:128;not null" json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
}

const (
	CampaignCodeDailyDefault = "daily_checkin_default"

	GrantStatusProcessing = "processing"
	GrantStatusSuccess    = "success"
	GrantStatusFailed     = "failed"
)
