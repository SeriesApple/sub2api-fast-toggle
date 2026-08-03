package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	defaultListenAddr = ":8080"
	defaultUpstream   = "http://host.docker.internal:8081"
	defaultModeFile   = "/runtime/mode"
	defaultMaxBody    = int64(64 << 20)
	statusPath        = "/_sub2api-fast/status"
)

type modeState struct {
	enabled bool
	valid   bool
}

type modeReader func() modeState

type proxyHandler struct {
	proxy       *httputil.ReverseProxy
	models      map[string]struct{}
	paths       map[string]struct{}
	maxBodySize int64
	mode        modeReader
	fastTier    string
}

type rewriteAction string

const (
	actionNone    rewriteAction = ""
	actionEnabled rewriteAction = "enabled"
	actionRemoved rewriteAction = "removed"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC)

	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		runHealthcheck()
		return
	}

	listenAddr := envOrDefault("LISTEN_ADDR", defaultListenAddr)
	upstream := envOrDefault("UPSTREAM_URL", defaultUpstream)
	modeFile := envOrDefault("FAST_MODE_FILE", defaultModeFile)
	maxBodySize := envInt64OrDefault("MAX_BODY_BYTES", defaultMaxBody)
	models := parseSet(envOrDefault("PRIORITY_MODELS", "gpt-5.6-sol"))
	paths := parseSet(envOrDefault("RESPONSE_PATHS", "/v1/responses,/responses,/backend-api/codex/responses"))
	fastTier := strings.ToLower(envOrDefault("FAST_SERVICE_TIER", "fast"))

	handler, err := newProxyHandler(upstream, models, paths, maxBodySize, fileModeReader(modeFile), fastTier)
	if err != nil {
		log.Fatalf("invalid proxy configuration: %v", err)
	}

	server := &http.Server{
		Addr:              listenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       2 * time.Minute,
		WriteTimeout:      0,
		MaxHeaderBytes:    1 << 20,
	}

	serverErr := make(chan error, 1)
	go func() {
		state := handler.mode()
		log.Printf(
			"fast proxy listening addr=%s upstream=%s mode=%s mode_valid=%t models=%s",
			listenAddr,
			upstream,
			modeName(state),
			state.valid,
			strings.Join(sortedKeys(models), ","),
		)
		serverErr <- server.ListenAndServe()
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("proxy server failed: %v", err)
		}
	case sig := <-signals:
		log.Printf("received signal=%s; draining connections", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	}
}

func newProxyHandler(
	upstream string,
	models map[string]struct{},
	paths map[string]struct{},
	maxBodySize int64,
	mode modeReader,
	fastTier string,
) (*proxyHandler, error) {
	target, err := url.Parse(upstream)
	if err != nil {
		return nil, fmt.Errorf("parse upstream URL: %w", err)
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return nil, fmt.Errorf("unsupported upstream scheme %q", target.Scheme)
	}
	if target.Host == "" {
		return nil, errors.New("upstream URL has no host")
	}
	if len(models) == 0 {
		return nil, errors.New("PRIORITY_MODELS must contain at least one model or *")
	}
	if len(paths) == 0 {
		return nil, errors.New("RESPONSE_PATHS must contain at least one path")
	}
	if maxBodySize <= 0 {
		return nil, errors.New("MAX_BODY_BYTES must be positive")
	}
	if mode == nil {
		return nil, errors.New("mode reader is required")
	}
	if fastTier != "fast" && fastTier != "priority" {
		return nil, errors.New("FAST_SERVICE_TIER must be fast or priority")
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	baseDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		baseDirector(req)
		req.Host = target.Host
	}
	proxy.FlushInterval = -1
	proxy.Transport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DisableCompression:    true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, proxyErr error) {
		log.Printf("upstream request failed method=%s path=%s error=%q", req.Method, req.URL.Path, proxyErr.Error())
		http.Error(w, "upstream gateway unavailable", http.StatusBadGateway)
	}

	return &proxyHandler{
		proxy:       proxy,
		models:      models,
		paths:       paths,
		maxBodySize: maxBodySize,
		mode:        mode,
		fastTier:    fastTier,
	}, nil
}

func (h *proxyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet && req.URL.Path == statusPath {
		h.writeStatus(w)
		return
	}

	if req.Method == http.MethodPost {
		if _, matched := h.paths[req.URL.Path]; matched {
			action, model, status, err := h.rewriteResponseRequest(req)
			if err != nil {
				http.Error(w, err.Error(), status)
				return
			}
			if action != actionNone {
				log.Printf("fast event=%s method=%s path=%s model=%s", action, req.Method, req.URL.Path, model)
			}
		}
	}

	h.proxy.ServeHTTP(w, req)
}

func (h *proxyHandler) rewriteResponseRequest(req *http.Request) (rewriteAction, string, int, error) {
	if !isIdentityEncoding(req.Header.Get("Content-Encoding")) {
		return actionNone, "", http.StatusUnsupportedMediaType, errors.New("compressed Responses requests are not supported")
	}
	if req.Body == nil {
		return actionNone, "", http.StatusBadRequest, errors.New("Responses request body is required")
	}
	if req.ContentLength > h.maxBodySize {
		return actionNone, "", http.StatusRequestEntityTooLarge, errors.New("Responses request body is too large")
	}

	originalBody := req.Body
	body, err := io.ReadAll(io.LimitReader(originalBody, h.maxBodySize+1))
	_ = originalBody.Close()
	if err != nil {
		return actionNone, "", http.StatusBadRequest, errors.New("could not read Responses request body")
	}
	if int64(len(body)) > h.maxBodySize {
		return actionNone, "", http.StatusRequestEntityTooLarge, errors.New("Responses request body is too large")
	}
	restoreBody(req, body)

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil || payload == nil {
		return actionNone, "", http.StatusBadRequest, errors.New("Responses request body must be a JSON object")
	}

	var model string
	if err := json.Unmarshal(payload["model"], &model); err != nil || model == "" {
		return actionNone, "", 0, nil
	}
	if !setMatches(h.models, model) {
		return actionNone, model, 0, nil
	}

	state := h.mode()
	if state.enabled {
		if _, exists := payload["service_tier"]; exists {
			return actionNone, model, 0, nil
		}
		tierJSON, _ := json.Marshal(h.fastTier)
		payload["service_tier"] = tierJSON
		if err := replaceJSONBody(req, payload); err != nil {
			return actionNone, model, http.StatusInternalServerError, errors.New("could not enable Fast mode")
		}
		return actionEnabled, model, 0, nil
	}

	rawTier, exists := payload["service_tier"]
	if !exists {
		return actionNone, model, 0, nil
	}
	var tier string
	if err := json.Unmarshal(rawTier, &tier); err != nil {
		return actionNone, model, 0, nil
	}
	tier = strings.ToLower(strings.TrimSpace(tier))
	if tier != "fast" && tier != "priority" {
		return actionNone, model, 0, nil
	}
	delete(payload, "service_tier")
	if err := replaceJSONBody(req, payload); err != nil {
		return actionNone, model, http.StatusInternalServerError, errors.New("could not disable Fast mode")
	}
	return actionRemoved, model, 0, nil
}

func (h *proxyHandler) writeStatus(w http.ResponseWriter) {
	state := h.mode()
	payload := struct {
		Mode            string   `json:"mode"`
		Valid           bool     `json:"valid"`
		FastServiceTier string   `json:"fast_service_tier"`
		Models          []string `json:"models"`
		Paths           []string `json:"paths"`
	}{
		Mode:            modeName(state),
		Valid:           state.valid,
		FastServiceTier: h.fastTier,
		Models:          sortedKeys(h.models),
		Paths:           sortedKeys(h.paths),
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func replaceJSONBody(req *http.Request, payload map[string]json.RawMessage) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	restoreBody(req, body)
	return nil
}

func restoreBody(req *http.Request, body []byte) {
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	req.TransferEncoding = nil
	req.Header.Set("Content-Length", strconv.Itoa(len(body)))
	req.Header.Del("Transfer-Encoding")
	req.Header.Del("Expect")
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
}

func fileModeReader(path string) modeReader {
	return func() modeState {
		data, err := os.ReadFile(path)
		if err != nil {
			return modeState{enabled: false, valid: false}
		}
		switch strings.ToLower(strings.TrimSpace(string(data))) {
		case "on":
			return modeState{enabled: true, valid: true}
		case "off":
			return modeState{enabled: false, valid: true}
		default:
			return modeState{enabled: false, valid: false}
		}
	}
}

func modeName(state modeState) string {
	if state.enabled {
		return "on"
	}
	return "off"
}

func isIdentityEncoding(value string) bool {
	value = strings.TrimSpace(value)
	return value == "" || strings.EqualFold(value, "identity")
}

func parseSet(value string) map[string]struct{} {
	items := make(map[string]struct{})
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items[item] = struct{}{}
		}
	}
	return items
}

func setMatches(items map[string]struct{}, value string) bool {
	if _, all := items["*"]; all {
		return true
	}
	_, matched := items[value]
	return matched
}

func sortedKeys(items map[string]struct{}) []string {
	keys := make([]string, 0, len(items))
	for item := range items {
		keys = append(keys, item)
	}
	sort.Strings(keys)
	return keys
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envInt64OrDefault(name string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		log.Fatalf("invalid %s: %v", name, err)
	}
	return parsed
}

func runHealthcheck() {
	client := &http.Client{Timeout: 4 * time.Second}
	response, err := client.Get("http://127.0.0.1:8080" + statusPath)
	if err != nil {
		log.Printf("healthcheck failed: %v", err)
		os.Exit(1)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		log.Printf("healthcheck returned status=%d", response.StatusCode)
		os.Exit(1)
	}
}
