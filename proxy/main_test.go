package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFastOnAddsTierAndPreservesAuthorization(t *testing.T) {
	var receivedBody []byte
	var receivedAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		receivedBody, _ = io.ReadAll(req.Body)
		receivedAuth = req.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	handler := mustTestHandler(t, upstream.URL, staticMode(true), 1<<20)
	request := httptest.NewRequest(http.MethodPost, "http://proxy/v1/responses", strings.NewReader(`{"model":"gpt-5.6-sol","input":"ping"}`))
	request.Header.Set("Authorization", "Bearer test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if receivedAuth != "Bearer test" {
		t.Fatal("authorization header was not preserved")
	}
	assertTier(t, receivedBody, "fast", true)
}

func TestFastOnPreservesExplicitTier(t *testing.T) {
	for _, tier := range []string{`"priority"`, `"standard"`, `"flex"`, `null`} {
		t.Run(tier, func(t *testing.T) {
			original := []byte(`{"model":"gpt-5.6-sol","service_tier":` + tier + `,"input":"ping"}`)
			received := roundTripBody(t, "/v1/responses", original, staticMode(true), 1<<20, "")
			if !bytes.Equal(received, original) {
				t.Fatalf("body changed\n got: %s\nwant: %s", received, original)
			}
		})
	}
}

func TestFastOffRemovesFastTiers(t *testing.T) {
	for _, tier := range []string{"fast", "priority", "FAST", "PRIORITY"} {
		t.Run(tier, func(t *testing.T) {
			original := []byte(`{"model":"gpt-5.6-sol","service_tier":"` + tier + `","input":"ping"}`)
			received := roundTripBody(t, "/v1/responses", original, staticMode(false), 1<<20, "")
			assertTier(t, received, "", false)
		})
	}
}

func TestFastOffPreservesNonFastTiers(t *testing.T) {
	tests := []string{
		`{"model":"gpt-5.6-sol","input":"ping"}`,
		`{"model":"gpt-5.6-sol","service_tier":"standard","input":"ping"}`,
		`{"model":"gpt-5.6-sol","service_tier":"flex","input":"ping"}`,
		`{"model":"gpt-5.6-sol","service_tier":null,"input":"ping"}`,
	}
	for _, body := range tests {
		received := roundTripBody(t, "/v1/responses", []byte(body), staticMode(false), 1<<20, "")
		if string(received) != body {
			t.Fatalf("body changed\n got: %s\nwant: %s", received, body)
		}
	}
}

func TestModeFileChangesWithoutRestart(t *testing.T) {
	directory := t.TempDir()
	modePath := filepath.Join(directory, "mode")
	writeMode(t, modePath, "on")

	var bodies [][]byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		bodies = append(bodies, body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	handler := mustTestHandler(t, upstream.URL, fileModeReader(modePath), 1<<20)
	requestBody := `{"model":"gpt-5.6-sol","input":"ping"}`
	sendRequest(t, handler, "/v1/responses", []byte(requestBody), "")
	writeMode(t, modePath, "off")
	sendRequest(t, handler, "/v1/responses", []byte(requestBody), "")

	assertTier(t, bodies[0], "fast", true)
	assertTier(t, bodies[1], "", false)
}

func TestMissingOrInvalidModeFailsSafeToOff(t *testing.T) {
	directory := t.TempDir()
	modePath := filepath.Join(directory, "mode")
	reader := fileModeReader(modePath)

	if state := reader(); state.enabled || state.valid {
		t.Fatalf("missing mode state = %+v, want invalid off", state)
	}
	writeMode(t, modePath, "unexpected")
	if state := reader(); state.enabled || state.valid {
		t.Fatalf("invalid mode state = %+v, want invalid off", state)
	}
}

func TestStatusEndpoint(t *testing.T) {
	handler := mustTestHandler(t, "http://127.0.0.1:1", staticMode(true), 1<<20)
	request := httptest.NewRequest(http.MethodGet, "http://proxy"+statusPath, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("status response must disable caching")
	}
	var payload struct {
		Mode  string `json:"mode"`
		Valid bool   `json:"valid"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if payload.Mode != "on" || !payload.Valid {
		t.Fatalf("status payload = %+v", payload)
	}
}

func TestNonMatchingRequestsPassThrough(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{name: "other model", path: "/v1/responses", body: `{"model":"other-model","service_tier":"priority"}`},
		{name: "other path", path: "/v1/chat/completions", body: `{"model":"gpt-5.6-sol","service_tier":"priority"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			received := roundTripBody(t, test.path, []byte(test.body), staticMode(false), 1<<20, "")
			if string(received) != test.body {
				t.Fatalf("body changed\n got: %s\nwant: %s", received, test.body)
			}
		})
	}
}

func TestRejectsUninspectableResponseRequests(t *testing.T) {
	t.Run("compressed", func(t *testing.T) {
		status := sendRejectedRequest(t, []byte(`{"model":"gpt-5.6-sol"}`), 1<<20, "gzip")
		if status != http.StatusUnsupportedMediaType {
			t.Fatalf("status = %d, want 415", status)
		}
	})

	t.Run("oversized", func(t *testing.T) {
		status := sendRejectedRequest(t, []byte(strings.Repeat("x", 128)), 64, "")
		if status != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want 413", status)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		status := sendRejectedRequest(t, []byte(`{"model":`), 1<<20, "")
		if status != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", status)
		}
	})
}

func TestStreamsResponseWithoutWaitingForCompletion(t *testing.T) {
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		_, _ = io.WriteString(w, "data: first\n\n")
		flusher.Flush()
		<-release
		_, _ = io.WriteString(w, "data: second\n\n")
		flusher.Flush()
	}))
	defer upstream.Close()

	proxyServer := httptest.NewServer(mustTestHandler(t, upstream.URL, staticMode(true), 1<<20))
	defer proxyServer.Close()

	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Post(proxyServer.URL+"/v1/responses", "application/json", strings.NewReader(`{"model":"gpt-5.6-sol"}`))
	if err != nil {
		t.Fatalf("start streaming request: %v", err)
	}
	defer response.Body.Close()

	first := make([]byte, len("data: first\n\n"))
	if _, err := io.ReadFull(response.Body, first); err != nil {
		t.Fatalf("read first event: %v", err)
	}
	if string(first) != "data: first\n\n" {
		t.Fatalf("first event = %q", first)
	}
	close(release)
}

func roundTripBody(t *testing.T, path string, body []byte, mode modeReader, maxBody int64, contentEncoding string) []byte {
	t.Helper()
	var received []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		received, _ = io.ReadAll(req.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	handler := mustTestHandler(t, upstream.URL, mode, maxBody)
	sendRequest(t, handler, path, body, contentEncoding)
	return received
}

func sendRequest(t *testing.T, handler http.Handler, path string, body []byte, contentEncoding string) {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "http://proxy"+path, bytes.NewReader(body))
	if contentEncoding != "" {
		request.Header.Set("Content-Encoding", contentEncoding)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", response.Code, response.Body.String())
	}
}

func sendRejectedRequest(t *testing.T, body []byte, maxBody int64, contentEncoding string) int {
	t.Helper()
	handler := mustTestHandler(t, "http://127.0.0.1:1", staticMode(false), maxBody)
	request := httptest.NewRequest(http.MethodPost, "http://proxy/v1/responses", bytes.NewReader(body))
	if contentEncoding != "" {
		request.Header.Set("Content-Encoding", contentEncoding)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response.Code
}

func mustTestHandler(t *testing.T, upstream string, mode modeReader, maxBody int64) http.Handler {
	t.Helper()
	handler, err := newProxyHandler(
		upstream,
		map[string]struct{}{"gpt-5.6-sol": {}},
		map[string]struct{}{`/v1/responses`: {}},
		maxBody,
		mode,
		"fast",
	)
	if err != nil {
		t.Fatalf("newProxyHandler: %v", err)
	}
	return handler
}

func staticMode(enabled bool) modeReader {
	return func() modeState { return modeState{enabled: enabled, valid: true} }
}

func writeMode(t *testing.T, path, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value+"\n"), 0o600); err != nil {
		t.Fatalf("write mode: %v", err)
	}
}

func assertTier(t *testing.T, body []byte, expected string, shouldExist bool) {
	t.Helper()
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	raw, exists := payload["service_tier"]
	if exists != shouldExist {
		t.Fatalf("service_tier exists = %t, want %t; body=%s", exists, shouldExist, body)
	}
	if !exists {
		return
	}
	var actual string
	if err := json.Unmarshal(raw, &actual); err != nil {
		t.Fatalf("decode tier: %v", err)
	}
	if actual != expected {
		t.Fatalf("service_tier = %q, want %q", actual, expected)
	}
}
