package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"welfare-backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrSub2APIUserNotFound = errors.New("sub2api user not found")

type AuthService struct {
	db                *gorm.DB
	sub2api           *Sub2APIClient
	syntheticDomain   string
	adminSubjectAllow map[string]struct{}
}

func NewAuthService(db *gorm.DB, sub2api *Sub2APIClient, syntheticDomain string, adminSubjectAllow map[string]struct{}) *AuthService {
	return &AuthService{
		db:                db,
		sub2api:           sub2api,
		syntheticDomain:   strings.TrimSpace(syntheticDomain),
		adminSubjectAllow: adminSubjectAllow,
	}
}

func (s *AuthService) SyntheticEmail(subject string) string {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return ""
	}
	return fmt.Sprintf("linuxdo-%s@%s", subject, s.syntheticDomain)
}

func (s *AuthService) ResolveAndBindUser(ctx context.Context, subject, username string) (*model.UserBinding, error) {
	email := s.SyntheticEmail(subject)
	if email == "" {
		return nil, errors.New("invalid linuxdo subject")
	}
	user, err := s.sub2api.FindUserBySyntheticEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrSub2APIUserNotFound
	}
	if strings.TrimSpace(user.Status) != "" && !strings.EqualFold(user.Status, "active") {
		return nil, fmt.Errorf("sub2api user not active: %s", user.Status)
	}

	now := time.Now()
	binding := model.UserBinding{
		LinuxDOSubject:  subject,
		LinuxDOUsername: username,
		SyntheticEmail:  email,
		Sub2APIUserID:   user.ID,
		Sub2APIEmail:    user.Email,
		Status:          "active",
		FirstLoginAt:    now,
		LastLoginAt:     now,
	}

	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "linux_do_subject"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"linux_do_username": username,
			"synthetic_email":   email,
			"sub2_api_user_id":  user.ID,
			"sub2_api_email":    user.Email,
			"status":            "active",
			"last_login_at":     now,
			"updated_at":        now,
		}),
	}).Create(&binding).Error; err != nil {
		return nil, err
	}

	var out model.UserBinding
	if err := s.db.WithContext(ctx).Where("linux_do_subject = ?", subject).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *AuthService) IsAdminSubject(subject string) bool {
	_, ok := s.adminSubjectAllow[strings.TrimSpace(subject)]
	return ok
}
