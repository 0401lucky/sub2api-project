package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthClaims struct {
	LinuxDOSubject string `json:"linuxdo_subject"`
	LinuxDOName    string `json:"linuxdo_name"`
	Sub2APIUserID  int64  `json:"sub2api_user_id"`
	Sub2APIEmail   string `json:"sub2api_email"`
	IsAdmin        bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

type JWTService struct {
	secret []byte
	expire time.Duration
}

func NewJWTService(secret string, expire time.Duration) *JWTService {
	return &JWTService{secret: []byte(secret), expire: expire}
}

func (s *JWTService) Sign(claims AuthClaims) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(s.expire)
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Subject:   claims.LinuxDOSubject,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		NotBefore: jwt.NewNumericDate(now),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	raw, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return raw, expiresAt, nil
}

func (s *JWTService) Parse(token string) (*AuthClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &AuthClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*AuthClaims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
