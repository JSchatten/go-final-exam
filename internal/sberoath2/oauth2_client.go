package sberoath2

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// OAuth2Response описывает ответ от Sber OAuth2
type OAuth2Response struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"` // В миллисекундах
}

// OAuth2Client управляет жизненным циклом access_token
type OAuth2Client struct {
	clientID     string
	clientSecret string
	scope        string
	tokenURL     string

	// Опциональная функция генерации RqUID
	// По умолчанию — uuid.New().String()
	rquidGenerator func() string

	mu        sync.RWMutex
	token     string
	expiresAt time.Time
}

// RqUIDGenerator — тип функции для генерации RqUID
type RqUIDGenerator func() string

// NewOAuth2Client создаёт новый клиент OAuth2
func NewOAuth2Client(
	clientID, clientSecret, scope, tokenURL string,
	rquidGen RqUIDGenerator,
) *OAuth2Client {
	if rquidGen == nil {
		rquidGen = func() string {
			return uuid.New().String()
		}
	}

	return &OAuth2Client{
		clientID:       clientID,
		clientSecret:   clientSecret,
		scope:          scope,
		tokenURL:       tokenURL,
		rquidGenerator: rquidGen,
	}
}

// GetToken возвращает актуальный access_token (при необходимости обновляет)
func (o *OAuth2Client) GetToken() (string, error) {
	o.mu.RLock()
	shouldRefresh := time.Now().After(o.expiresAt)
	token := o.token
	o.mu.RUnlock()

	if !shouldRefresh && token != "" {
		return token, nil
	}

	// Блокируем на запись
	o.mu.Lock()
	defer o.mu.Unlock()

	// Двойная проверка после захвата мьютекса
	if time.Now().Before(o.expiresAt) && o.token != "" {
		return o.token, nil
	}

	// Запрашиваем новый токен
	err := o.refreshToken()
	if err != nil {
		return "", fmt.Errorf("failed to refresh access token: %w", err)
	}

	return o.token, nil
}

// refreshToken запрашивает новый access_token с уникальным RqUID
func (o *OAuth2Client) refreshToken() error {
	data := url.Values{}
	data.Set("scope", o.scope)

	req, err := http.NewRequest("POST", o.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Генерируем уникальный RqUID для этого запроса
	rquid := o.rquidGenerator()

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("RqUID", rquid)
	req.Header.Set("Authorization", "Basic "+basicAuth(o.clientID, o.clientSecret))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("oauth request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var oauthResp OAuth2Response
	err = json.Unmarshal(body, &oauthResp)
	if err != nil {
		return fmt.Errorf("failed to unmarshal oauth response: %w", err)
	}

	// Преобразуем expires_at из миллисекунд в time.Time
	expTime := time.UnixMilli(oauthResp.ExpiresAt)

	o.token = oauthResp.AccessToken
	o.expiresAt = expTime

	return nil
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
