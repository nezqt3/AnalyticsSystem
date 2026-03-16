package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const SessionCookieName = "analytics_admin_session"

type Manager struct {
	email         string
	password      string
	sessionSecret []byte
	ttl           time.Duration
}

type User struct {
	Email string `json:"email"`
}

func NewManager(email string, password string, sessionSecret string) *Manager {
	return &Manager{
		email:         strings.TrimSpace(email),
		password:      password,
		sessionSecret: []byte(strings.TrimSpace(sessionSecret)),
		ttl:           24 * time.Hour,
	}
}

func (m *Manager) Enabled() bool {
	return m != nil && m.email != "" && m.password != "" && len(m.sessionSecret) > 0
}

func (m *Manager) ValidateCredentials(email string, password string) bool {
	if !m.Enabled() {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(email)), []byte(m.email)) != 1 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(password), []byte(m.password)) == 1
}

func (m *Manager) CreateToken() (string, error) {
	if !m.Enabled() {
		return "", errors.New("auth manager disabled")
	}
	expiresAt := time.Now().UTC().Add(m.ttl).Unix()
	payload := fmt.Sprintf("%s|%d", m.email, expiresAt)
	signature := m.sign(payload)
	return base64.RawURLEncoding.EncodeToString([]byte(payload + "|" + signature)), nil
}

func (m *Manager) ParseToken(token string) (User, error) {
	if !m.Enabled() {
		return User{}, errors.New("auth manager disabled")
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return User{}, err
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 {
		return User{}, errors.New("invalid token format")
	}
	payload := strings.Join(parts[:2], "|")
	if subtle.ConstantTimeCompare([]byte(parts[2]), []byte(m.sign(payload))) != 1 {
		return User{}, errors.New("invalid token signature")
	}
	if subtle.ConstantTimeCompare([]byte(parts[0]), []byte(m.email)) != 1 {
		return User{}, errors.New("invalid token user")
	}
	expiresAt, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return User{}, err
	}
	if time.Now().UTC().Unix() > expiresAt {
		return User{}, errors.New("token expired")
	}
	return User{Email: m.email}, nil
}

func (m *Manager) sign(payload string) string {
	h := hmac.New(sha256.New, m.sessionSecret)
	_, _ = h.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
