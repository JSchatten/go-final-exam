package sberoath2

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuth2Client_GetToken_Success(t *testing.T) {
	// Создаем мок-сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем метод и заголовки
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v2/oauth", r.URL.Path)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.NotEmpty(t, r.Header.Get("RqUID"))
		_, err := uuid.Parse(r.Header.Get("RqUID"))
		assert.NoError(t, err, "RqUID must be valid UUID")

		auth := r.Header.Get("Authorization")
		assert.True(t, strings.HasPrefix(auth, "Basic "))

		// Читаем тело
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), "scope=SALUTE_SPEECH_PERS")

		// Отдаем успешный ответ
		resp := OAuth2Response{
			AccessToken: "test-access-token",
			ExpiresAt:   time.Now().Add(10 * time.Minute).UnixMilli(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOAuth2Client(
		"client-id",
		"client-secret",
		"SALUTE_SPEECH_PERS",
		server.URL+"/api/v2/oauth",
		nil,
	)

	token, err := client.GetToken()
	require.NoError(t, err)
	assert.Equal(t, "test-access-token", token)
}

func TestOAuth2Client_TokenCaching(t *testing.T) {
	var handlerCalls int
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		handlerCalls++
		mu.Unlock()

		resp := OAuth2Response{
			AccessToken: fmt.Sprintf("token-%d", handlerCalls),
			ExpiresAt:   time.Now().Add(5 * time.Second).UnixMilli(), // Действует 5 секунд
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOAuth2Client("id", "secret", "scope", server.URL+"/api/v2/oauth", nil)

	// Первый вызов — должен запросить токен
	token1, err := client.GetToken()
	require.NoError(t, err)
	assert.Equal(t, "token-1", token1)

	// Второй вызов сразу — должен вернуть из кэша
	token2, err := client.GetToken()
	require.NoError(t, err)
	assert.Equal(t, "token-1", token2)
	assert.Equal(t, 1, handlerCalls) // Запрос не ушёл на сервер

	// Ждём 6 секунд — токен просрочен
	time.Sleep(6 * time.Second)

	// Третий вызов — должен обновиться
	token3, err := client.GetToken()
	require.NoError(t, err)
	assert.Equal(t, "token-2", token3)
	assert.Equal(t, 2, handlerCalls) // Новый запрос
}

func TestOAuth2Client_InvalidResponse_Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid credentials"}`))
	}))
	defer server.Close()

	client := NewOAuth2Client("id", "secret", "scope", server.URL+"/api/v2/oauth", nil)

	_, err := client.GetToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed with status 401")
}

func TestOAuth2Client_InvalidResponse_Body(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json}`)) // Невалидный JSON
	}))
	defer server.Close()

	client := NewOAuth2Client("id", "secret", "scope", server.URL+"/api/v2/oauth", nil)

	_, err := client.GetToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal oauth response")
}

func TestOAuth2Client_CustomRqUIDGenerator(t *testing.T) {
	const expectedRqUID = "custom-rquid"

	var capturedRqUID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRqUID = r.Header.Get("RqUID")

		resp := OAuth2Response{
			AccessToken: "test-token",
			ExpiresAt:   time.Now().Add(time.Minute).UnixMilli(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	generator := func() string { return expectedRqUID }
	client := NewOAuth2Client("id", "secret", "scope", server.URL+"/api/v2/oauth", generator)

	_, err := client.GetToken()
	require.NoError(t, err)
	assert.Equal(t, expectedRqUID, capturedRqUID)
}

func TestOAuth2Client_BasicAuthEncoding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		assert.True(t, strings.HasPrefix(auth, "Basic "))
		decoded, _ := base64.StdEncoding.DecodeString(auth[6:])
		assert.Equal(t, "user:pass", string(decoded))

		resp := OAuth2Response{
			AccessToken: "token",
			ExpiresAt:   time.Now().Add(time.Minute).UnixMilli(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOAuth2Client("user", "pass", "scope", server.URL+"/api/v2/oauth", nil)
	_, err := client.GetToken()
	assert.NoError(t, err)
}
