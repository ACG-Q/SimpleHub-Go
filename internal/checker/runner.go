package checker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"simplehub-go/internal/proxy"
)

type ModelInfo struct {
	ID            string   `json:"id"`
	Name          string   `json:"name,omitempty"`
	Object        string   `json:"object,omitempty"`
	OwnedBy       string   `json:"owned_by,omitempty"`
	Created       float64  `json:"created,omitempty"`
	Permission    string   `json:"permission,omitempty"`
	Root          string   `json:"root,omitempty"`
	Parent        string   `json:"parent,omitempty"`
	Type          string   `json:"type,omitempty"`
	ChargingType  string   `json:"chargingType,omitempty"`
	InputPrice    *float64 `json:"inputPrice,omitempty"`
	OutputPrice   *float64 `json:"outputPrice,omitempty"`
	SinglePrice   *float64 `json:"singlePrice,omitempty"`
}

type CheckResult struct {
	Models       []ModelInfo `json:"models"`
	Hash         string      `json:"hash"`
	RawResponse  string      `json:"rawResponse,omitempty"`
	StatusCode   int         `json:"statusCode"`
	ResponseTime int         `json:"responseTime"`
	ErrorMessage string      `json:"error,omitempty"`

	BillingLimit *float64 `json:"billingLimit,omitempty"`
	BillingUsage *float64 `json:"billingUsage,omitempty"`
	BillingError string   `json:"billingError,omitempty"`

	CheckInSuccess *bool    `json:"checkInSuccess,omitempty"`
	CheckInMessage string   `json:"checkInMessage,omitempty"`
	CheckInQuota   *float64 `json:"checkInQuota,omitempty"`
	CheckInError   string   `json:"checkInError,omitempty"`
}

type Config struct {
	APIKey            string
	BaseURL           string
	ProxyURL          string
	APIType           string
	UserID            string
	BillingURL        string
	BillingLimitField string
	BillingUsageField string
	BillingAuthType   string
	BillingAuthValue  string
	UnlimitedQuota    bool
	CheckInMode       string
	EnableCheckIn     bool
	IsManual          bool
}

type Runner struct {
	proxyClient *proxy.ProxyClient
}

func NewRunner(pc *proxy.ProxyClient) *Runner {
	return &Runner{proxyClient: pc}
}

func (r *Runner) Run(cfg Config) *CheckResult {
	result := &CheckResult{}

	log.Debug().
		Str("api_type", cfg.APIType).
		Str("base_url", cfg.BaseURL).
		Str("user_id", cfg.UserID).
		Str("checkin_mode", cfg.CheckInMode).
		Bool("enable_checkin", cfg.EnableCheckIn).
		Bool("is_manual", cfg.IsManual).
		Bool("unlimited_quota", cfg.UnlimitedQuota).
		Msg("check run started")

	needCheckIn := cfg.EnableCheckIn && (cfg.APIType == "newapi" || cfg.APIType == "veloera" || cfg.APIType == "voapi")
	if needCheckIn && !cfg.IsManual {
		needCheckIn = cfg.CheckInMode == "" || cfg.CheckInMode == "checkin" || cfg.CheckInMode == "both"
	}
	if needCheckIn && (cfg.APIType == "newapi" || cfg.APIType == "veloera") && cfg.UserID == "" {
		needCheckIn = false
		log.Warn().Str("api_type", cfg.APIType).Msg("checkin skipped: missing user_id")
	}

	if needCheckIn {
		r.performCheckIn(cfg, result)
	}

	needModels := cfg.IsManual || cfg.CheckInMode == "" || cfg.CheckInMode == "model" || cfg.CheckInMode == "both"

	var wg sync.WaitGroup

	if needModels {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.fetchModels(cfg, result)
		}()
	}

	if !cfg.UnlimitedQuota {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.fetchBilling(cfg, result)
		}()
	}

	wg.Wait()

	if needModels && len(result.Models) > 0 {
		sort.Slice(result.Models, func(i, j int) bool {
			return result.Models[i].ID < result.Models[j].ID
		})
		data, _ := json.Marshal(result.Models)
		h := sha256.Sum256(data)
		result.Hash = hex.EncodeToString(h[:])
	}

	log.Debug().
		Str("api_type", cfg.APIType).
		Int("model_count", len(result.Models)).
		Int("status_code", result.StatusCode).
		Int("response_time_ms", result.ResponseTime).
		Str("hash", result.Hash).
		Interface("billing_limit", result.BillingLimit).
		Interface("billing_usage", result.BillingUsage).
		Bool("checkin_success", result.CheckInSuccess != nil && *result.CheckInSuccess).
		Msg("check run completed")

	return result
}

func (r *Runner) performCheckIn(cfg Config, result *CheckResult) {
	start := time.Now()

	client := proxy.NewDirectClient()
	if cfg.ProxyURL != "" {
		c, err := r.proxyClient.GetClient(cfg.ProxyURL)
		if err == nil {
			client = &proxy.DirectClient{Client: c}
		}
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")

	var checkinURL string
	var headers map[string]string

	switch cfg.APIType {
	case "newapi":
		checkinURL = baseURL + "/api/user/checkin"
		headers = map[string]string{
			"Authorization":           "Bearer " + cfg.APIKey,
			"new-api-user":            cfg.UserID,
			"Content-Type":            "application/json",
			"Accept":                  "application/json",
			"Cache-Control":           "no-store",
			"User-Agent":              "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		}
	case "veloera":
		checkinURL = baseURL + "/api/user/check_in"
		headers = map[string]string{
			"Authorization":           "Bearer " + cfg.APIKey,
			"veloera-user":            cfg.UserID,
			"Content-Type":            "application/json",
			"Accept":                  "application/json",
			"Cache-Control":           "no-store",
			"User-Agent":              "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		}
	case "voapi":
		checkinURL = baseURL + "/api/check_in"
		headers = map[string]string{
			"Authorization":           cfg.APIKey,
			"Accept":                  "application/json, text/plain, */*",
			"Cache-Control":           "no-store",
			"User-Agent":              "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		}
	}

	log.Debug().
		Str("api_type", cfg.APIType).
		Str("url", checkinURL).
		Bool("has_proxy", cfg.ProxyURL != "").
		Msg("performCheckIn request")

	req, _ := http.NewRequest("POST", checkinURL, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	body, statusCode, err := doRequest(client.Client, req, 15*time.Second)
	duration := int(time.Since(start).Milliseconds())
	if err != nil {
		log.Error().Err(err).Str("api_type", cfg.APIType).Msg("performCheckIn failed")
		result.CheckInSuccess = boolPtr(false)
		result.CheckInMessage = "签到异常"
		result.CheckInError = err.Error()
		return
	}

	bodyPreview := string(body)
	if len(bodyPreview) > 200 {
		bodyPreview = bodyPreview[:200] + "..."
	}

	if statusCode != 200 {
		log.Warn().
			Str("api_type", cfg.APIType).
			Int("status_code", statusCode).
			Int("duration_ms", duration).
			Str("body_preview", bodyPreview).
			Msg("performCheckIn non-200")
		result.CheckInSuccess = boolPtr(false)
		result.CheckInMessage = fmt.Sprintf("签到HTTP %d", statusCode)
		result.CheckInError = fmt.Sprintf("HTTP %d", statusCode)
		return
	}

	var parsed struct {
		Success bool `json:"success"`
		Code    int  `json:"code"`
		Message string `json:"message"`
		Msg     string `json:"msg"`
		Error   string `json:"error"`
		Data    *struct {
			Quota  *float64 `json:"quota"`
			Amount *float64 `json:"amount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		log.Error().Str("api_type", cfg.APIType).Str("body_preview", bodyPreview).Msg("performCheckIn parse failed")
		result.CheckInSuccess = boolPtr(false)
		result.CheckInMessage = "签到响应解析失败"
		result.CheckInError = fmt.Sprintf("invalid JSON: %s", string(body[:min(len(body), 100)]))
		return
	}

	errorMsg := parsed.Message
	if errorMsg == "" {
		errorMsg = parsed.Msg
	}
	if errorMsg == "" {
		errorMsg = parsed.Error
	}

	if cfg.APIType == "voapi" && parsed.Code == 0 {
		var quota *float64
		if parsed.Data != nil && parsed.Data.Amount != nil {
			q := *parsed.Data.Amount
			quota = &q
		}
		msg := errorMsg
		if msg == "" {
			msg = "签到成功"
		}
		result.CheckInSuccess = boolPtr(true)
		result.CheckInMessage = msg
		result.CheckInQuota = quota
		log.Info().Str("api_type", cfg.APIType).Str("msg", msg).Interface("quota", quota).Msg("performCheckIn success")
		return
	}

	if cfg.APIType != "voapi" && parsed.Success {
		var quota *float64
		if parsed.Data != nil && parsed.Data.Quota != nil {
			q := *parsed.Data.Quota
			quota = &q
		}
		msg := errorMsg
		if msg == "" {
			msg = "签到成功"
		}
		result.CheckInSuccess = boolPtr(true)
		result.CheckInMessage = msg
		result.CheckInQuota = quota
		log.Info().Str("api_type", cfg.APIType).Str("msg", msg).Interface("quota", quota).Msg("performCheckIn success")
		return
	}

	if errorMsg == "" {
		errorMsg = "签到失败"
	}
	result.CheckInSuccess = boolPtr(false)
	result.CheckInMessage = errorMsg
	result.CheckInError = errorMsg
	log.Warn().Str("api_type", cfg.APIType).Str("msg", errorMsg).Msg("performCheckIn failed")
}

func (r *Runner) fetchModels(cfg Config, result *CheckResult) {
	start := time.Now()

	client := proxy.NewDirectClient()
	if cfg.ProxyURL != "" {
		c, err := r.proxyClient.GetClient(cfg.ProxyURL)
		if err == nil {
			client = &proxy.DirectClient{Client: c}
		}
	}

	var modelsURL string
	var headers map[string]string

	switch cfg.APIType {
	case "newapi", "veloera":
		modelsURL = strings.TrimRight(cfg.BaseURL, "/") + "/api/user/models"
		headers = map[string]string{
			"Authorization": "Bearer " + cfg.APIKey,
			"Content-Type":  "application/json",
			"User-Agent":    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			"Accept":        "application/json",
		}
		if cfg.UserID != "" {
			userHeader := "New-Api-User"
			if cfg.APIType == "veloera" {
				userHeader = "Veloera-User"
			}
			headers[userHeader] = cfg.UserID
		}

	case "donehub":
		modelsURL = strings.TrimRight(cfg.BaseURL, "/") + "/api/available_model"
		headers = map[string]string{
			"Authorization": "Bearer " + cfg.APIKey,
			"Content-Type":  "application/json",
			"User-Agent":    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			"Accept":        "application/json",
		}

	case "voapi":
		modelsURL = strings.TrimRight(cfg.BaseURL, "/") + "/api/models"
		headers = map[string]string{
			"Authorization": cfg.APIKey,
			"Content-Type":  "application/json",
			"Accept":        "application/json, text/plain, */*",
		}

	default:
		modelsURL = strings.TrimRight(cfg.BaseURL, "/") + "/v1/models"
		headers = map[string]string{
			"Authorization": "Bearer " + cfg.APIKey,
			"Content-Type":  "application/json",
		}
	}

	req, err := http.NewRequest("GET", modelsURL, nil)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("create request failed: %v", err)
		return
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	log.Debug().
		Str("api_type", cfg.APIType).
		Str("url", modelsURL).
		Str("user_id", cfg.UserID).
		Bool("has_proxy", cfg.ProxyURL != "").
		Msg("fetchModels request")

	body, statusCode, err := doRequest(client.Client, req, 15*time.Second)
	duration := int(time.Since(start).Milliseconds())
	if err != nil {
		log.Error().
			Str("api_type", cfg.APIType).
			Str("url", modelsURL).
			Err(err).
			Msg("fetchModels failed")
		result.ErrorMessage = fmt.Sprintf("HTTP request failed: %v", err)
		result.StatusCode = statusCode
		result.ResponseTime = duration
		return
	}

	result.StatusCode = statusCode
	result.RawResponse = string(body)
	result.ResponseTime = duration

	bodyPreview := string(body)
	if len(bodyPreview) > 200 {
		bodyPreview = bodyPreview[:200] + "..."
	}

	if statusCode != 200 {
		log.Warn().
			Str("api_type", cfg.APIType).
			Str("url", modelsURL).
			Int("status_code", statusCode).
			Int("duration_ms", duration).
			Str("body_preview", bodyPreview).
			Msg("fetchModels non-200 response")
		result.ErrorMessage = fmt.Sprintf("HTTP %d: %s", statusCode, string(body))
		return
	}

	models, parseErr := parseModels(cfg.APIType, body)
	if parseErr != nil {
		log.Error().
			Str("api_type", cfg.APIType).
			Err(parseErr).
			Msg("fetchModels parse error")
		result.ErrorMessage = fmt.Sprintf("JSON parse error: %v", parseErr)
		return
	}
	result.Models = models

	log.Debug().
		Str("api_type", cfg.APIType).
		Int("model_count", len(models)).
		Int("status_code", statusCode).
		Int("duration_ms", duration).
		Str("body_preview", bodyPreview).
		Msg("fetchModels success")
}

func (r *Runner) fetchBilling(cfg Config, result *CheckResult) {
	log.Debug().
		Str("api_type", cfg.APIType).
		Msg("fetchBilling routing")
	switch cfg.APIType {
	case "newapi", "veloera":
		r.fetchBillingNewapi(cfg, result)
	case "donehub":
		r.fetchBillingDonehub(cfg, result)
	case "voapi":
		r.fetchBillingVoapi(cfg, result)
	default:
		r.fetchBillingOther(cfg, result)
	}
}

func (r *Runner) fetchBillingNewapi(cfg Config, result *CheckResult) {
	start := time.Now()
	client := proxy.NewDirectClient()
	if cfg.ProxyURL != "" {
		c, err := r.proxyClient.GetClient(cfg.ProxyURL)
		if err == nil {
			client = &proxy.DirectClient{Client: c}
		}
	}

	url := strings.TrimRight(cfg.BaseURL, "/") + "/api/user/self"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	userHeader := ""
	if cfg.UserID != "" {
		userHeader = "New-Api-User"
		if cfg.APIType == "veloera" {
			userHeader = "Veloera-User"
		}
		req.Header.Set(userHeader, cfg.UserID)
	}

	log.Debug().
		Str("api_type", cfg.APIType).
		Str("url", url).
		Str("user_header", userHeader).
		Bool("has_proxy", cfg.ProxyURL != "").
		Msg("fetchBillingNewapi request")

	body, statusCode, err := doRequest(client.Client, req, 10*time.Second)
	duration := int(time.Since(start).Milliseconds())
	if err != nil {
		log.Error().
			Str("api_type", cfg.APIType).
			Str("url", url).
			Err(err).
			Msg("fetchBillingNewapi failed")
		result.BillingError = fmt.Sprintf("billing request failed: %v", err)
		return
	}

	bodyPreview := string(body)
	if len(bodyPreview) > 200 {
		bodyPreview = bodyPreview[:200] + "..."
	}

	if statusCode != 200 {
		log.Warn().
			Str("api_type", cfg.APIType).
			Int("status_code", statusCode).
			Int("duration_ms", duration).
			Str("body_preview", bodyPreview).
			Msg("fetchBillingNewapi non-200 response")
		result.BillingError = fmt.Sprintf("billing HTTP %d", statusCode)
		return
	}

	var resp struct {
		Success bool `json:"success"`
		Data    *struct {
			Quota     float64 `json:"quota"`
			UsedQuota float64 `json:"used_quota"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || !resp.Success || resp.Data == nil {
		result.BillingError = fmt.Sprintf("invalid billing response: %s", string(body))
		log.Error().
			Str("api_type", cfg.APIType).
			Str("body_preview", bodyPreview).
			Msg("fetchBillingNewapi invalid response")
		return
	}

	const ratio = 500000.0
	total := (resp.Data.Quota + resp.Data.UsedQuota) / ratio
	used := resp.Data.UsedQuota / ratio
	result.BillingLimit = &total
	result.BillingUsage = &used

	log.Debug().
		Str("api_type", cfg.APIType).
		Int("status_code", statusCode).
		Int("duration_ms", duration).
		Float64("quota_raw", resp.Data.Quota).
		Float64("used_quota_raw", resp.Data.UsedQuota).
		Float64("limit", total).
		Float64("usage", used).
		Msg("fetchBillingNewapi success")
}

func (r *Runner) fetchBillingDonehub(cfg Config, result *CheckResult) {
	start := time.Now()
	client := proxy.NewDirectClient()
	if cfg.ProxyURL != "" {
		c, err := r.proxyClient.GetClient(cfg.ProxyURL)
		if err == nil {
			client = &proxy.DirectClient{Client: c}
		}
	}

	url := strings.TrimRight(cfg.BaseURL, "/") + "/api/user/self"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	log.Debug().
		Str("api_type", cfg.APIType).
		Str("url", url).
		Bool("has_proxy", cfg.ProxyURL != "").
		Msg("fetchBillingDonehub request")

	body, statusCode, err := doRequest(client.Client, req, 10*time.Second)
	duration := int(time.Since(start).Milliseconds())
	if err != nil {
		log.Error().
			Str("api_type", cfg.APIType).
			Err(err).
			Msg("fetchBillingDonehub failed")
		result.BillingError = fmt.Sprintf("billing request failed: %v", err)
		return
	}

	bodyPreview := string(body)
	if len(bodyPreview) > 200 {
		bodyPreview = bodyPreview[:200] + "..."
	}

	if statusCode != 200 {
		log.Warn().
			Str("api_type", cfg.APIType).
			Int("status_code", statusCode).
			Int("duration_ms", duration).
			Str("body_preview", bodyPreview).
			Msg("fetchBillingDonehub non-200 response")
		result.BillingError = fmt.Sprintf("billing HTTP %d", statusCode)
		return
	}

	var resp struct {
		Success bool `json:"success"`
		Data    *struct {
			Quota     float64 `json:"quota"`
			UsedQuota float64 `json:"used_quota"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || !resp.Success || resp.Data == nil {
		result.BillingError = fmt.Sprintf("invalid billing response: %s", string(body))
		log.Error().
			Str("api_type", cfg.APIType).
			Str("body_preview", bodyPreview).
			Msg("fetchBillingDonehub invalid response")
		return
	}

	const ratio = 500000.0
	total := (resp.Data.Quota + resp.Data.UsedQuota) / ratio
	used := resp.Data.UsedQuota / ratio
	result.BillingLimit = &total
	result.BillingUsage = &used

	log.Debug().
		Str("api_type", cfg.APIType).
		Int("status_code", statusCode).
		Int("duration_ms", duration).
		Float64("quota_raw", resp.Data.Quota).
		Float64("used_quota_raw", resp.Data.UsedQuota).
		Float64("limit", total).
		Float64("usage", used).
		Msg("fetchBillingDonehub success")
}

func (r *Runner) fetchBillingVoapi(cfg Config, result *CheckResult) {
	start := time.Now()
	client := proxy.NewDirectClient()
	if cfg.ProxyURL != "" {
		c, err := r.proxyClient.GetClient(cfg.ProxyURL)
		if err == nil {
			client = &proxy.DirectClient{Client: c}
		}
	}

	url := strings.TrimRight(cfg.BaseURL, "/") + "/api/user/info"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	log.Debug().
		Str("api_type", cfg.APIType).
		Str("url", url).
		Bool("has_proxy", cfg.ProxyURL != "").
		Msg("fetchBillingVoapi request")

	body, statusCode, err := doRequest(client.Client, req, 10*time.Second)
	duration := int(time.Since(start).Milliseconds())
	if err != nil {
		log.Error().
			Str("api_type", cfg.APIType).
			Err(err).
			Msg("fetchBillingVoapi failed")
		result.BillingError = fmt.Sprintf("billing request failed: %v", err)
		return
	}

	bodyPreview := string(body)
	if len(bodyPreview) > 200 {
		bodyPreview = bodyPreview[:200] + "..."
	}

	if statusCode != 200 {
		log.Warn().
			Str("api_type", cfg.APIType).
			Int("status_code", statusCode).
			Int("duration_ms", duration).
			Str("body_preview", bodyPreview).
			Msg("fetchBillingVoapi non-200 response")
		result.BillingError = fmt.Sprintf("billing HTTP %d", statusCode)
		return
	}

	var resp struct {
		Code int `json:"code"`
		Data *struct {
			BindBalance      float64 `json:"bindBalance"`
			BasicBalance     float64 `json:"basicBalance"`
			UsedBindBalance  float64 `json:"usedBindBalance"`
			UsedBasicBalance float64 `json:"usedBasicBalance"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || resp.Code != 0 || resp.Data == nil {
		result.BillingError = fmt.Sprintf("invalid billing response: %s", string(body))
		log.Error().
			Str("api_type", cfg.APIType).
			Str("body_preview", bodyPreview).
			Msg("fetchBillingVoapi invalid response")
		return
	}

	total := resp.Data.BindBalance + resp.Data.BasicBalance
	used := resp.Data.UsedBindBalance + resp.Data.UsedBasicBalance
	result.BillingLimit = &total
	result.BillingUsage = &used

	log.Debug().
		Str("api_type", cfg.APIType).
		Int("status_code", statusCode).
		Int("duration_ms", duration).
		Float64("bind_balance", resp.Data.BindBalance).
		Float64("basic_balance", resp.Data.BasicBalance).
		Float64("used_bind", resp.Data.UsedBindBalance).
		Float64("used_basic", resp.Data.UsedBasicBalance).
		Float64("limit", total).
		Float64("usage", used).
		Msg("fetchBillingVoapi success")
}

func (r *Runner) fetchBillingOther(cfg Config, result *CheckResult) {
	if cfg.BillingURL != "" {
		log.Debug().Str("api_type", cfg.APIType).Str("billing_url", cfg.BillingURL).Msg("fetchBillingOther: custom billing")
		r.fetchCustomBilling(cfg, result)
	} else {
		log.Debug().Str("api_type", cfg.APIType).Msg("fetchBillingOther: OpenAI-style billing")
		r.fetchOpenAIBilling(cfg, result)
	}
}

func (r *Runner) fetchCustomBilling(cfg Config, result *CheckResult) {
	start := time.Now()
	client := proxy.NewDirectClient()
	if cfg.ProxyURL != "" {
		c, err := r.proxyClient.GetClient(cfg.ProxyURL)
		if err == nil {
			client = &proxy.DirectClient{Client: c}
		}
	}

	req, _ := http.NewRequest("GET", cfg.BillingURL, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	if cfg.BillingAuthValue != "" {
		if cfg.BillingAuthType == "token" {
			if strings.HasPrefix(cfg.BillingAuthValue, "Bearer ") {
				req.Header.Set("Authorization", cfg.BillingAuthValue)
			} else {
				req.Header.Set("Authorization", "Bearer "+cfg.BillingAuthValue)
			}
		} else if cfg.BillingAuthType == "cookie" {
			req.Header.Set("Cookie", cfg.BillingAuthValue)
		}
	}

	log.Debug().
		Str("api_type", cfg.APIType).
		Str("url", cfg.BillingURL).
		Str("auth_type", cfg.BillingAuthType).
		Bool("has_auth_value", cfg.BillingAuthValue != "").
		Str("limit_field", cfg.BillingLimitField).
		Str("usage_field", cfg.BillingUsageField).
		Bool("has_proxy", cfg.ProxyURL != "").
		Msg("fetchCustomBilling request")

	body, statusCode, err := doRequest(client.Client, req, 10*time.Second)
	duration := int(time.Since(start).Milliseconds())
	if err != nil {
		log.Error().
			Str("api_type", cfg.APIType).
			Str("url", cfg.BillingURL).
			Err(err).
			Msg("fetchCustomBilling failed")
		result.BillingError = fmt.Sprintf("billing request failed: %v", err)
		return
	}

	bodyPreview := string(body)
	if len(bodyPreview) > 200 {
		bodyPreview = bodyPreview[:200] + "..."
	}

	if statusCode != 200 {
		log.Warn().
			Str("api_type", cfg.APIType).
			Int("status_code", statusCode).
			Int("duration_ms", duration).
			Str("body_preview", bodyPreview).
			Msg("fetchCustomBilling non-200 response")
		result.BillingError = fmt.Sprintf("billing HTTP %d", statusCode)
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		result.BillingError = fmt.Sprintf("billing JSON error: %v", err)
		log.Error().
			Str("api_type", cfg.APIType).
			Err(err).
			Msg("fetchCustomBilling JSON error")
		return
	}

	if cfg.BillingLimitField != "" && cfg.BillingUsageField != "" {
		if v, ok := getNestedFloat(data, cfg.BillingLimitField); ok {
			result.BillingLimit = &v
		}
		if v, ok := getNestedFloat(data, cfg.BillingUsageField); ok {
			result.BillingUsage = &v
		}
	} else {
		if v, ok := getNestedFloat(data, "limit"); ok {
			result.BillingLimit = &v
		}
		if v, ok := getNestedFloat(data, "usage"); ok {
			result.BillingUsage = &v
		}
		if v, ok := getNestedFloat(data, "quota"); ok {
			result.BillingLimit = &v
		}
		if v, ok := getNestedFloat(data, "used"); ok {
			result.BillingUsage = &v
		}
		if v, ok := getNestedFloat(data, "balance"); ok {
			result.BillingLimit = &v
		}
		if v, ok := getNestedFloat(data, "consumed"); ok {
			result.BillingUsage = &v
		}
		if v, ok := getNestedFloat(data, "system_hard_limit_usd"); ok {
			result.BillingLimit = &v
		}
		if v, ok := getNestedFloat(data, "total_usage"); ok {
			u := v * 0.01
			result.BillingUsage = &u
		}
	}

	log.Debug().
		Str("api_type", cfg.APIType).
		Int("status_code", statusCode).
		Int("duration_ms", duration).
		Interface("limit", result.BillingLimit).
		Interface("usage", result.BillingUsage).
		Str("body_preview", bodyPreview).
		Msg("fetchCustomBilling success")
}

func (r *Runner) fetchOpenAIBilling(cfg Config, result *CheckResult) {
	client := proxy.NewDirectClient()
	if cfg.ProxyURL != "" {
		c, err := r.proxyClient.GetClient(cfg.ProxyURL)
		if err == nil {
			client = &proxy.DirectClient{Client: c}
		}
	}

	subURL := strings.TrimRight(cfg.BaseURL, "/") + "/v1/dashboard/billing/subscription"
	subReq, _ := http.NewRequest("GET", subURL, nil)
	subReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	subReq.Header.Set("Content-Type", "application/json")
	subReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	subReq.Header.Set("Accept", "application/json, text/plain, */*")
	subReq.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	subReq.Header.Set("Cache-Control", "no-cache")
	subReq.Header.Set("Pragma", "no-cache")

	usageURL := strings.TrimRight(cfg.BaseURL, "/") + "/v1/dashboard/billing/usage"
	usageReq, _ := http.NewRequest("GET", usageURL, nil)
	usageReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	usageReq.Header.Set("Content-Type", "application/json")
	usageReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	usageReq.Header.Set("Accept", "application/json, text/plain, */*")
	usageReq.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	usageReq.Header.Set("Cache-Control", "no-cache")
	usageReq.Header.Set("Pragma", "no-cache")

	log.Debug().
		Str("api_type", cfg.APIType).
		Str("sub_url", subURL).
		Str("usage_url", usageURL).
		Bool("has_proxy", cfg.ProxyURL != "").
		Msg("fetchOpenAIBilling request")

	subStart := time.Now()
	subBody, subStatus, subErr := doRequest(client.Client, subReq, 10*time.Second)
	subDuration := int(time.Since(subStart).Milliseconds())

	if subErr != nil {
		log.Error().Str("api_type", cfg.APIType).Str("url", subURL).Err(subErr).Msg("fetchOpenAIBilling subscription failed")
	} else {
		subPreview := string(subBody)
		if len(subPreview) > 200 {
			subPreview = subPreview[:200] + "..."
		}
		log.Debug().
			Str("api_type", cfg.APIType).
			Int("status_code", subStatus).
			Int("duration_ms", subDuration).
			Str("body_preview", subPreview).
			Msg("fetchOpenAIBilling subscription response")
	}

	if subErr == nil && subStatus == 200 {
		var subData struct {
			HardLimitUsd *float64 `json:"system_hard_limit_usd"`
		}
		if err := json.Unmarshal(subBody, &subData); err == nil && subData.HardLimitUsd != nil {
			result.BillingLimit = subData.HardLimitUsd
			log.Debug().
				Str("api_type", cfg.APIType).
				Float64("hard_limit_usd", *subData.HardLimitUsd).
				Msg("fetchOpenAIBilling parsed subscription")
		}
	}

	usageStart := time.Now()
	usageBody, usageStatus, usageErr := doRequest(client.Client, usageReq, 10*time.Second)
	usageDuration := int(time.Since(usageStart).Milliseconds())

	if usageErr != nil {
		log.Error().Str("api_type", cfg.APIType).Str("url", usageURL).Err(usageErr).Msg("fetchOpenAIBilling usage failed")
	} else {
		usagePreview := string(usageBody)
		if len(usagePreview) > 200 {
			usagePreview = usagePreview[:200] + "..."
		}
		log.Debug().
			Str("api_type", cfg.APIType).
			Int("status_code", usageStatus).
			Int("duration_ms", usageDuration).
			Str("body_preview", usagePreview).
			Msg("fetchOpenAIBilling usage response")
	}

	if usageErr == nil && usageStatus == 200 {
		var usageData struct {
			TotalUsage *float64 `json:"total_usage"`
		}
		if err := json.Unmarshal(usageBody, &usageData); err == nil && usageData.TotalUsage != nil {
			u := *usageData.TotalUsage * 0.01
			result.BillingUsage = &u
			log.Debug().
				Str("api_type", cfg.APIType).
				Float64("total_usage_raw", *usageData.TotalUsage).
				Float64("usage", u).
				Msg("fetchOpenAIBilling parsed usage")
		}
	}

	var errs []string
	if subErr != nil {
		errs = append(errs, "subscription: "+subErr.Error())
	} else if subStatus != 200 {
		errs = append(errs, fmt.Sprintf("subscription HTTP %d", subStatus))
	}
	if usageErr != nil {
		errs = append(errs, "usage: "+usageErr.Error())
	} else if usageStatus != 200 {
		errs = append(errs, fmt.Sprintf("usage HTTP %d", usageStatus))
	}
	if len(errs) > 0 {
		result.BillingError = strings.Join(errs, "; ")
	}
}

func doRequest(client *http.Client, req *http.Request, timeout time.Duration) ([]byte, int, error) {
	client.Timeout = timeout
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read body failed: %w", err)
	}
	return body, resp.StatusCode, nil
}

func parseModels(apiType string, body []byte) ([]ModelInfo, error) {
	switch apiType {
	case "newapi", "veloera":
		return parseNewapiModels(body, apiType)
	case "donehub":
		return parseDonehubModels(body)
	case "voapi":
		return parseVoapiModels(body)
	default:
		return parseOpenaiModels(body)
	}
}

func parseNewapiModels(body []byte, apiType string) ([]ModelInfo, error) {
	var resp struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("api returned success=false")
	}

	ownedBy := "new-api"
	if apiType == "veloera" {
		ownedBy = "veloera"
	}

	var objModels []struct {
		ID      string  `json:"id"`
		Object  string  `json:"object"`
		OwnedBy string  `json:"owned_by"`
		Created float64 `json:"created"`
	}
	if err := json.Unmarshal(resp.Data, &objModels); err == nil {
		var models []ModelInfo
		for _, m := range objModels {
			ob := m.OwnedBy
			if ob == "" {
				ob = ownedBy
			}
			obj := m.Object
			if obj == "" {
				obj = "model"
			}
			cr := m.Created
			if cr == 0 {
				cr = float64(time.Now().Unix())
			}
			models = append(models, ModelInfo{
				ID:      m.ID,
				Name:    m.ID,
				Object:  obj,
				OwnedBy: ob,
				Created: cr,
			})
		}
		return models, nil
	}

	var strIDs []string
	if err := json.Unmarshal(resp.Data, &strIDs); err != nil {
		return nil, fmt.Errorf("unexpected data format")
	}
	now := float64(time.Now().Unix())
	var models []ModelInfo
	for _, id := range strIDs {
		models = append(models, ModelInfo{
			ID:      id,
			Name:    id,
			Object:  "model",
			OwnedBy: ownedBy,
			Created: now,
		})
	}
	return models, nil
}

func parseDonehubModels(body []byte) ([]ModelInfo, error) {
	var resp struct {
		Data map[string]struct {
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	now := float64(time.Now().Unix())
	var models []ModelInfo
	for id, v := range resp.Data {
		ob := v.OwnedBy
		if ob == "" {
			ob = "unknown"
		}
		models = append(models, ModelInfo{
			ID:      id,
			Name:    id,
			Object:  "model",
			OwnedBy: ob,
			Created: now,
		})
	}
	return models, nil
}

func parseVoapiModels(body []byte) ([]ModelInfo, error) {
	var resp struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("api returned code=%d", resp.Code)
	}
	if resp.Data == nil {
		return nil, fmt.Errorf("no data in response")
	}

	var rawModels []json.RawMessage

	var withModels struct {
		Models []json.RawMessage `json:"models"`
	}
	if err := json.Unmarshal(resp.Data, &withModels); err == nil && len(withModels.Models) > 0 {
		rawModels = withModels.Models
	} else {
		var arr []json.RawMessage
		if err := json.Unmarshal(resp.Data, &arr); err == nil {
			rawModels = arr
		}
	}

	var models []ModelInfo
	for _, raw := range rawModels {
		var m struct {
			IDKey        string   `json:"idKey"`
			Model        string   `json:"model"`
			ID           string   `json:"id"`
			FirmIDKey    string   `json:"firmIdKey"`
			Firm         string   `json:"firm"`
			Provider     string   `json:"provider"`
			Object       string   `json:"object"`
			Created      float64  `json:"created"`
			ChargingType string   `json:"chargingType"`
			InputPrice   *float64 `json:"inputPrice"`
			OutputPrice  *float64 `json:"outputPrice"`
			SinglePrice  *float64 `json:"singlePrice"`
		}
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		id := m.IDKey
		if id == "" {
			id = m.Model
		}
		if id == "" {
			id = m.ID
		}
		if id == "" {
			continue
		}
		ownedBy := m.FirmIDKey
		if ownedBy == "" {
			ownedBy = m.Firm
		}
		if ownedBy == "" {
			ownedBy = m.Provider
		}
		if ownedBy == "" {
			ownedBy = "voapi"
		}
		obj := m.Object
		if obj == "" {
			obj = "model"
		}
		cr := m.Created
		if cr == 0 {
			cr = float64(time.Now().Unix())
		} else if cr > 1e12 {
			cr = cr / 1000
		}
		models = append(models, ModelInfo{
			ID:           id,
			Name:         id,
			Object:       obj,
			OwnedBy:      ownedBy,
			Created:      cr,
			ChargingType: m.ChargingType,
			InputPrice:   m.InputPrice,
			OutputPrice:  m.OutputPrice,
			SinglePrice:  m.SinglePrice,
		})
	}
	return models, nil
}

func parseOpenaiModels(body []byte) ([]ModelInfo, error) {
	var resp struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, fmt.Errorf("no data in response")
	}

	var objModels []struct {
		ID         string  `json:"id"`
		Object     string  `json:"object"`
		OwnedBy    string  `json:"owned_by"`
		Created    float64 `json:"created"`
		Permission string  `json:"permission"`
		Root       string  `json:"root"`
		Parent     string  `json:"parent"`
		Type       string  `json:"type"`
	}
	if err := json.Unmarshal(resp.Data, &objModels); err == nil {
		now := float64(time.Now().Unix())
		var models []ModelInfo
		for _, m := range objModels {
			ob := m.OwnedBy
			if ob == "" {
				ob = "unknown"
			}
			obj := m.Object
			if obj == "" {
				obj = "model"
			}
			cr := m.Created
			if cr == 0 {
				cr = now
			}
			models = append(models, ModelInfo{
				ID:         m.ID,
				Name:       m.ID,
				Object:     obj,
				OwnedBy:    ob,
				Created:    cr,
				Permission: m.Permission,
				Root:       m.Root,
				Parent:     m.Parent,
				Type:       m.Type,
			})
		}
		return models, nil
	}

	var strIDs []string
	if err := json.Unmarshal(resp.Data, &strIDs); err != nil {
		return nil, fmt.Errorf("unexpected data format")
	}
	now := float64(time.Now().Unix())
	var models []ModelInfo
	for _, id := range strIDs {
		models = append(models, ModelInfo{
			ID:      id,
			Name:    id,
			Object:  "model",
			OwnedBy: "unknown",
			Created: now,
		})
	}
	return models, nil
}

func getNestedFloat(data map[string]interface{}, path string) (float64, bool) {
	parts := strings.Split(path, ".")
	current := interface{}(data)
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return 0, false
		}
		current = m[part]
	}
	switch v := current.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case json.Number:
		f, _ := v.Float64()
		return f, true
	}
	return 0, false
}

func hashStrings(ss []string) string {
	input := strings.Join(ss, ",")
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

func boolPtr(b bool) *bool {
	return &b
}
