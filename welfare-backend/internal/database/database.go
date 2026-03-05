package database

import (
	"errors"
	"fmt"
	"time"

	"welfare-backend/internal/config"
	"welfare-backend/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

func New(cfg *config.Config) (*gorm.DB, error) {
	var (
		db  *gorm.DB
		err error
	)

	gcfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)}
	switch cfg.DatabaseDriver {
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(cfg.DatabaseDSN), gcfg)
	case "postgres":
		db, err = gorm.Open(postgres.Open(cfg.DatabaseDSN), gcfg)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.DatabaseDriver)
	}
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.AutoMigrate(
		&model.UserBinding{},
		&model.CheckinCampaign{},
		&model.CheckinGrant{},
		&model.AdminAuditLog{},
		&model.RiskBlock{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	if err := ensureDefaultCampaign(db, cfg.CheckinTimezone); err != nil {
		return nil, err
	}
	return db, nil
}

func ensureDefaultCampaign(db *gorm.DB, tz string) error {
	var existing model.CheckinCampaign
	err := db.Where("code = ?", model.CampaignCodeDailyDefault).First(&existing).Error
	if err == nil {
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("check default campaign: %w", err)
	}

	now := time.Now()
	campaign := model.CheckinCampaign{
		Code:           model.CampaignCodeDailyDefault,
		Name:           "每日签到",
		Enabled:        true,
		Timezone:       tz,
		RewardMode:     "uniform_decimal",
		RewardMin:      1,
		RewardMax:      5,
		RewardScale:    2,
		MaxPerDay:      1,
		RiskPolicyJSON: `{"version":1,"ip_rate_limit":"5/min"}`,
		CreatedBy:      "system",
		UpdatedBy:      "system",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoNothing: true,
	}).Create(&campaign).Error; err != nil {
		return fmt.Errorf("create default campaign: %w", err)
	}
	return nil
}
