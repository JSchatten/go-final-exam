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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v2/oauth", r.URL.Path)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.NotEmpty(t, r.Header.Get("RqUID"))
		_, err := uuid.Parse(r.Header.Get("RqUID"))
		assert.NoError(t, err)

		auth := r.Header.Get("Authorization")
		assert.True(t, strings.HasPrefix(auth, "Basic "))

		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), "scope=SALUTE_SPEECH_PERS")

		resp := OAuth2Response{
			AccessToken: "test-access-token",
			ExpiresAt:   time.Now().Add(10 * time.Minute).UnixMilli(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Подготавливаем корректный base64-encoded clientSecret
	validAuthKey := base64.StdEncoding.EncodeToString([]byte("client-id:client-secret"))

	client, err := NewOAuth2Client(
		"client-id",
		validAuthKey,
		"SALUTE_SPEECH_PERS",
		server.URL+"/api/v2/oauth",
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, client)

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
			ExpiresAt:   time.Now().Add(5 * time.Second).UnixMilli(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	validAuthKey := base64.StdEncoding.EncodeToString([]byte("id:secret"))

	client, err := NewOAuth2Client("id", validAuthKey, "scope", server.URL+"/api/v2/oauth", nil)
	require.NoError(t, err)

	token1, err := client.GetToken()
	require.NoError(t, err)
	assert.Equal(t, "token-1", token1)
	assert.Equal(t, 1, handlerCalls)

	token2, err := client.GetToken()
	require.NoError(t, err)
	assert.Equal(t, "token-1", token2)
	assert.Equal(t, 1, handlerCalls)

	time.Sleep(6 * time.Second)

	token3, err := client.GetToken()
	require.NoError(t, err)
	assert.Equal(t, "token-2", token3)
	assert.Equal(t, 2, handlerCalls)
}

func TestOAuth2Client_InvalidResponse_Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid credentials"}`))
	}))
	defer server.Close()

	validAuthKey := base64.StdEncoding.EncodeToString([]byte("id:secret"))

	client, err := NewOAuth2Client("id", validAuthKey, "scope", server.URL+"/api/v2/oauth", nil)
	require.NoError(t, err)

	_, err = client.GetToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed with status 401")
}

func TestOAuth2Client_InvalidResponse_Body(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	validAuthKey := base64.StdEncoding.EncodeToString([]byte("id:secret"))

	client, err := NewOAuth2Client("id", validAuthKey, "scope", server.URL+"/api/v2/oauth", nil)
	require.NoError(t, err)

	_, err = client.GetToken()
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
	validAuthKey := base64.StdEncoding.EncodeToString([]byte("id:secret"))

	client, err := NewOAuth2Client("id", validAuthKey, "scope", server.URL+"/api/v2/oauth", generator)
	require.NoError(t, err)

	_, err = client.GetToken()
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

	validAuthKey := base64.StdEncoding.EncodeToString([]byte("user:pass"))

	client, err := NewOAuth2Client("user", validAuthKey, "scope", server.URL+"/api/v2/oauth", nil)
	require.NoError(t, err)

	_, err = client.GetToken()
	assert.NoError(t, err)
}

// - Новые тесты -

func TestOAuth2Client_InvalidAuthKey_Base64(t *testing.T) {
	_, err := NewOAuth2Client(
		"client-id",
		"!!!invalid-base64!!!",
		"scope",
		"https://example.com/oauth",
		nil,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode AUTHORIZATION_KEY")
}

func TestOAuth2Client_InvalidAuthKey_Format(t *testing.T) {
	invalidFormat := base64.StdEncoding.EncodeToString([]byte("invalid-format-without-colon"))
	_, err := NewOAuth2Client(
		"client-id",
		invalidFormat,
		"scope",
		"https://example.com/oauth",
		nil,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid format: expected client_id:client_secret")
}

func TestOAuth2Client_ClientID_Mismatch(t *testing.T) {
	// Кодируем: wrong-id:secret
	authKey := base64.StdEncoding.EncodeToString([]byte("wrong-id:secret"))

	_, err := NewOAuth2Client(
		"correct-id",
		authKey,
		"scope",
		"https://example.com/oauth",
		nil,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "clientID mismatch")
}
