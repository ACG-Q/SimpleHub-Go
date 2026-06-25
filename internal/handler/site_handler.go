package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"simplehub-go/internal/crypto"
	"simplehub-go/internal/model"
	"simplehub-go/internal/repository"
	"simplehub-go/internal/service"
)

type SiteHandler struct {
	siteRepo         *repository.SiteRepository
	snapRepo         *repository.SnapshotRepository
	diffRepo         *repository.DiffRepository
	authService      *service.AuthService
	checkService     *service.CheckService
	schedulerService *service.SchedulerService
	encryptionKey    string
}

func NewSiteHandler(
	siteRepo *repository.SiteRepository,
	snapRepo *repository.SnapshotRepository,
	diffRepo *repository.DiffRepository,
	authService *service.AuthService,
	checkService *service.CheckService,
	schedulerService *service.SchedulerService,
	encryptionKey string,
) *SiteHandler {
	return &SiteHandler{
		siteRepo:         siteRepo,
		snapRepo:         snapRepo,
		diffRepo:         diffRepo,
		authService:      authService,
		checkService:     checkService,
		schedulerService: schedulerService,
		encryptionKey:    encryptionKey,
	}
}

type CreateSiteRequest struct {
	Name             string  `json:"name" binding:"required"`
	BaseURL          string  `json:"baseUrl" binding:"required"`
	APIKey           string  `json:"apiKey" binding:"required"`
	APIType          string  `json:"apiType"`
	UserID           *string `json:"userId"`
	BillingURL       *string `json:"billingUrl"`
	BillingAuthType  string  `json:"billingAuthType"`
	BillingAuthValue *string `json:"billingAuthValue"`
	ProxyURL         *string `json:"proxyUrl"`
	BillingLimitField  *string `json:"billingLimitField"`
	BillingUsageField  *string `json:"billingUsageField"`
	UnlimitedQuota   bool    `json:"unlimitedQuota"`
	EnableCheckIn    bool    `json:"enableCheckIn"`
	CheckInMode      string  `json:"checkInMode"`
	ScheduleCron     *string `json:"scheduleCron"`
	Timezone         string  `json:"timezone"`
	Pinned           bool    `json:"pinned"`
	ExcludeFromBatch bool    `json:"excludeFromBatch"`
	CategoryID       *string `json:"categoryId"`
	Extralink        *string `json:"extralink"`
	Remark           *string `json:"remark"`
	SortOrder        int     `json:"sortOrder"`
}

func (h *SiteHandler) List(c *gin.Context) {
	search := c.Query("search")
	sites, err := h.siteRepo.List(search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if search != "" {
		extraIDs, _ := h.snapRepo.FindSiteIDsByModelID(search)
		for _, id := range extraIDs {
			found := false
			for _, s := range sites {
				if s.ID == id {
					found = true
					break
				}
			}
			if !found {
				site, err := h.siteRepo.GetByID(id)
				if err == nil {
					sites = append(sites, *site)
				}
			}
		}
	}

	siteIDs := make([]string, len(sites))
	for i, s := range sites {
		siteIDs[i] = s.ID
	}
	snapMap, _ := h.snapRepo.GetLatestForSites(siteIDs)

	type siteResp struct {
		model.Site
		ModelsJSON any          `json:"modelsJson,omitempty"`
		BillingLimit  *float64 `json:"billingLimit,omitempty"`
		BillingUsage  *float64 `json:"billingUsage,omitempty"`
		CheckInSuccess *bool    `json:"checkInSuccess,omitempty"`
		CheckInQuota   *float64 `json:"checkInQuota,omitempty"`
	}
	resp := make([]siteResp, len(sites))
	for i, s := range sites {
		r := siteResp{Site: s}
		if snap, ok := snapMap[s.ID]; ok {
			r.ModelsJSON = parseModelsJSON(snap.ModelsJSON)
			r.BillingLimit = snap.BillingLimit
			r.BillingUsage = snap.BillingUsage
			r.CheckInSuccess = snap.CheckInSuccess
			r.CheckInQuota = snap.CheckInQuota
		}
		resp[i] = r
	}
	c.JSON(http.StatusOK, resp)
}

func (h *SiteHandler) Get(c *gin.Context) {
	id := c.Param("id")
	site, err := h.siteRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "站点不存在"})
		return
	}
	token, _ := crypto.Decrypt(site.APIKeyEnc, h.encryptionKey)
	var proxyURL *string
	if site.ProxyURLEnc != nil {
		dec, err := crypto.Decrypt(*site.ProxyURLEnc, h.encryptionKey)
		if err == nil {
			proxyURL = &dec
		}
	}
	var billingAV *string
	if site.BillingAuthValue != nil {
		dec, err := crypto.Decrypt(*site.BillingAuthValue, h.encryptionKey)
		if err == nil {
			billingAV = &dec
		}
	}
	resp := model.SiteResponse{
		Site:      *site,
		Token:     token,
		ProxyURL:  proxyURL,
		BillingAV: billingAV,
		Type:      site.APIType,
	}
	c.JSON(http.StatusOK, resp)
}

func (h *SiteHandler) Create(c *gin.Context) {
	var req CreateSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供必填字段"})
		return
	}

	apiKeyEnc, err := crypto.Encrypt(req.APIKey, h.encryptionKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加密失败"})
		return
	}

	apiType := nonZero(req.APIType, "other")
	checkInMode := nonZero(req.CheckInMode, "both")
	timezone := nonZero(req.Timezone, "UTC")

	site := &model.Site{
		ID:   newID(),
		Name: req.Name, BaseURL: req.BaseURL,
		APIKeyEnc: apiKeyEnc, APIType: apiType,
		UserID: req.UserID, BillingURL: req.BillingURL,
		BillingAuthType: nonZero(req.BillingAuthType, "token"),
		BillingLimitField: req.BillingLimitField,
		BillingUsageField: req.BillingUsageField,
		UnlimitedQuota: req.UnlimitedQuota,
		EnableCheckIn: req.EnableCheckIn,
		CheckInMode: checkInMode, ScheduleCron: req.ScheduleCron,
		Timezone: timezone, Pinned: req.Pinned,
		ExcludeFromBatch: req.ExcludeFromBatch,
		CategoryID: req.CategoryID, Extralink: req.Extralink,
		Remark: req.Remark, SortOrder: req.SortOrder,
	}
	if req.BillingAuthValue != nil && *req.BillingAuthValue != "" {
		encBA, errBA := crypto.Encrypt(*req.BillingAuthValue, h.encryptionKey)
		if errBA == nil {
			site.BillingAuthValue = &encBA
		}
	}
	if req.ProxyURL != nil && *req.ProxyURL != "" {
		normalized := normalizeProxyURL(*req.ProxyURL)
		encPU, errPU := crypto.Encrypt(normalized, h.encryptionKey)
		if errPU == nil {
			site.ProxyURLEnc = &encPU
		}
	}

	if err := h.siteRepo.Create(site); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.schedulerService != nil {
		h.schedulerService.ScheduleAll()
	}

	resp := model.SiteResponse{
		Site:      *site,
		Token:     req.APIKey,
		ProxyURL:  req.ProxyURL,
		BillingAV: req.BillingAuthValue,
		Type:      apiType,
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *SiteHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	updates := buildUpdates(body, h.encryptionKey)
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有需要更新的字段"})
		return
	}

	if err := h.siteRepo.Update(id, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	site, err := h.siteRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	token, _ := crypto.Decrypt(site.APIKeyEnc, h.encryptionKey)
	var proxyURL *string
	if site.ProxyURLEnc != nil {
		dec, err := crypto.Decrypt(*site.ProxyURLEnc, h.encryptionKey)
		if err == nil {
			proxyURL = &dec
		}
	}
	var billingAV *string
	if site.BillingAuthValue != nil {
		dec, err := crypto.Decrypt(*site.BillingAuthValue, h.encryptionKey)
		if err == nil {
			billingAV = &dec
		}
	}

	if h.schedulerService != nil {
		h.schedulerService.ScheduleAll()
	}

	resp := model.SiteResponse{
		Site:      *site,
		Token:     token,
		ProxyURL:  proxyURL,
		BillingAV: billingAV,
		Type:      site.APIType,
	}
	c.JSON(http.StatusOK, resp)
}

func buildUpdates(body map[string]interface{}, encKey string) map[string]interface{} {
	updates := make(map[string]interface{})

	if val, ok := body["apiKey"]; ok {
		if s, ok2 := val.(string); ok2 && s != "" {
			if enc, err := crypto.Encrypt(s, encKey); err == nil {
				updates["api_key_enc"] = enc
			}
		}
	}
	if val, ok := body["proxyUrl"]; ok {
		if s, ok2 := val.(string); ok2 && s != "" {
			body["proxyUrl"] = normalizeProxyURL(s)
		}
	}
	encryptOrNull(body, "proxyUrl", "proxy_url_enc", encKey, updates)
	encryptOrNull(body, "billingAuthValue", "billing_auth_value", encKey, updates)

	simpleFields := []string{
		"name", "baseUrl", "apiType",
		"billingAuthType", "checkInMode", "timezone",
	}
	for _, field := range simpleFields {
		if val, ok := body[field]; ok {
			updates[camelToSnake(field)] = val
		}
	}

	nullableFields := []string{
		"userId", "billingUrl", "billingLimitField", "billingUsageField",
		"scheduleCron", "extralink", "remark", "categoryId",
	}
	for _, field := range nullableFields {
		if val, ok := body[field]; ok {
			if val == nil {
				updates[camelToSnake(field)] = gorm.Expr("NULL")
			} else {
				updates[camelToSnake(field)] = val
			}
		}
	}

	boolFields := []string{"unlimitedQuota", "enableCheckIn", "pinned", "excludeFromBatch"}
	for _, field := range boolFields {
		if v, ok := body[field]; ok {
			if b, ok2 := v.(bool); ok2 {
				updates[camelToSnake(field)] = b
			}
		}
	}

	if val, ok := body["sortOrder"]; ok {
		switch v := val.(type) {
		case float64:
			updates["sort_order"] = int(v)
		}
	}

	return updates
}

func encryptOrNull(body map[string]interface{}, key, column, encKey string, updates map[string]interface{}) {
	if val, ok := body[key]; ok {
		if val == nil {
			updates[column] = gorm.Expr("NULL")
		} else if s, ok2 := val.(string); ok2 {
			if s == "" {
				updates[column] = gorm.Expr("NULL")
			} else if enc, err := crypto.Encrypt(s, encKey); err == nil {
				updates[column] = enc
			}
		}
	}
}

func camelToSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 && s[i-1] != '_' {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func nonZero(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func (h *SiteHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.siteRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if h.schedulerService != nil {
		h.schedulerService.RemoveSiteTask(id)
		h.schedulerService.ScheduleAll()
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *SiteHandler) Check(c *gin.Context) {
	id := c.Param("id")
	skipNotif := c.Query("skipNotification") == "true"

	result, err := h.checkService.CheckSite(id, skipNotif, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检测失败：" + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"message": "检测成功",
		"result":  result,
	})
}

func (h *SiteHandler) Reorder(c *gin.Context) {
	var req struct {
		Orders []struct {
			ID        string `json:"id"`
			SortOrder int    `json:"sortOrder"`
		} `json:"orders"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}
	for _, o := range req.Orders {
		if err := h.siteRepo.Update(o.ID, map[string]interface{}{"sort_order": o.SortOrder}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func parseModelsJSON(s string) any {
	var arr []any
	if json.Unmarshal([]byte(s), &arr) == nil {
		return arr
	}
	return []any{}
}

func (h *SiteHandler) GetSnapshots(c *gin.Context) {
	id := c.Param("id")
	limit := 1
	if l := c.Query("limit"); l != "" {
		if v, err := parseInt(l); err == nil && v > 0 {
			limit = v
		}
	}
	snaps, err := h.snapRepo.List(id, limit)
	if err != nil {
		c.JSON(http.StatusOK, []model.ModelSnapshot{})
		return
	}
	type resp struct {
		model.ModelSnapshot
		ModelsJSON any `json:"modelsJson"`
	}
	result := make([]resp, len(snaps))
	for i, s := range snaps {
		result[i] = resp{ModelSnapshot: s, ModelsJSON: parseModelsJSON(s.ModelsJSON)}
	}
	c.JSON(http.StatusOK, result)
}

func (h *SiteHandler) GetLatestSnapshot(c *gin.Context) {
	id := c.Param("id")
	snap, err := h.snapRepo.GetLatestWithError(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "无快照数据"})
		return
	}
	type resp struct {
		model.ModelSnapshot
		ModelsJSON any `json:"modelsJson"`
	}
	c.JSON(http.StatusOK, resp{ModelSnapshot: *snap, ModelsJSON: parseModelsJSON(snap.ModelsJSON)})
}

func (h *SiteHandler) GetDiffs(c *gin.Context) {
	id := c.Param("id")
	limit := 50
	if l := c.Query("limit"); l != "" {
		if v, err := parseInt(l); err == nil && v > 0 {
			limit = v
		}
	}
	diffs, err := h.diffRepo.List(id, limit)
	if err != nil {
		c.JSON(http.StatusOK, []model.ModelDiff{})
		return
	}
	type diffResp struct {
		model.ModelDiff
		AddedJSON   any `json:"addedJson"`
		RemovedJSON any `json:"removedJson"`
		ChangedJSON any `json:"changedJson"`
	}
	result := make([]diffResp, len(diffs))
	for i, d := range diffs {
		result[i] = diffResp{
			ModelDiff:   d,
			AddedJSON:   parseModelsJSON(d.AddedJSON),
			RemovedJSON: parseModelsJSON(d.RemovedJSON),
			ChangedJSON: parseModelsJSON(d.ChangedJSON),
		}
	}
	c.JSON(http.StatusOK, result)
}

type proxyRequest struct {
	pathByType      map[string]string
	skipAuth        bool   // skip Authorization header (e.g., newapi pricing)
	extraHeaders    map[string]string
	transformRequest func([]byte) []byte // optional request body transform (before sending)
	transform       func([]byte) []byte // optional response body transform
	verbose         bool   // log full request details
}

func pickPath(apiType string, pathByType map[string]string) string {
	if p, ok := pathByType[apiType]; ok {
		return p
	}
	for _, p := range pathByType {
		return p
	}
	return ""
}

func (h *SiteHandler) doProxy(c *gin.Context, site *model.Site, pr proxyRequest) {
	start := time.Now()

	apiKey, err := crypto.Decrypt(site.APIKeyEnc, h.encryptionKey)
	if err != nil {
		c.JSON(500, gin.H{"error": "解密失败"})
		return
	}

	upstreamPath := pickPath(site.APIType, pr.pathByType)
	upstreamURL := strings.TrimRight(site.BaseURL, "/") + upstreamPath

	bodyBytes, _ := io.ReadAll(c.Request.Body)
	if pr.transformRequest != nil {
		bodyBytes = pr.transformRequest(bodyBytes)
	}
	req, _ := http.NewRequest(c.Request.Method, upstreamURL, bytes.NewReader(bodyBytes))

	authType := "none"
	if !pr.skipAuth {
		if site.APIType == "voapi" {
			authType = "voapi_raw"
			req.Header.Set("Authorization", apiKey)
		} else {
			authType = "bearer"
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
	}

	var userHeader string
	if site.UserID != nil && *site.UserID != "" {
		switch site.APIType {
		case "newapi":
			userHeader = "New-Api-User"
			req.Header.Set("New-Api-User", *site.UserID)
		case "veloera":
			userHeader = "Veloera-User"
			req.Header.Set("Veloera-User", *site.UserID)
		}
	}

	for k, v := range pr.extraHeaders {
		req.Header.Set(k, v)
	}

	if pr.verbose {
		allHeaders := make(map[string]string)
		for k := range req.Header {
			allHeaders[k] = req.Header.Get(k)
		}
		log.Info().
			Str("site_id", site.ID).
			Str("site_name", site.Name).
			Str("api_type", site.APIType).
			Str("method", c.Request.Method).
			Str("upstream_url", upstreamURL).
			Str("auth_type", authType).
			Str("user_header", userHeader).
			Interface("all_headers", allHeaders).
			Msg("pricing proxy request")
	} else {
		log.Debug().
			Str("site_id", site.ID).
			Str("site_name", site.Name).
			Str("api_type", site.APIType).
			Str("method", c.Request.Method).
			Str("upstream_url", upstreamURL).
			Str("auth_type", authType).
			Str("user_header", userHeader).
			Interface("extra_headers", pr.extraHeaders).
			Msg("proxy request")
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().
			Str("site_id", site.ID).
			Str("upstream_url", upstreamURL).
			Err(err).
			Msg("proxy upstream request failed")
		c.JSON(502, gin.H{"error": "上游请求失败: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	bodyPreview := string(body)
	if len(bodyPreview) > 200 {
		bodyPreview = bodyPreview[:200] + "..."
	}

	log.Debug().
		Str("site_id", site.ID).
		Int("status_code", resp.StatusCode).
		Str("content_type", resp.Header.Get("Content-Type")).
		Int("duration_ms", int(time.Since(start).Milliseconds())).
		Int("body_size", len(body)).
		Str("body_preview", bodyPreview).
		Msg("proxy response")

	if pr.transform != nil {
		body = pr.transform(body)
	}

	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}

func transformNewapiCreateRequest(body []byte) []byte {
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		return body
	}

	camelToSnake := map[string]string{
		"expiredTime":         "expired_time",
		"unlimitedQuota":      "unlimited_quota",
		"remainQuota":         "remain_quota",
		"modelLimitsEnabled":  "model_limits_enabled",
		"modelLimits":         "model_limits",
		"allowIps":            "allow_ips",
	}

	out := make(map[string]interface{})
	for k, v := range req {
		if snake, ok := camelToSnake[k]; ok {
			out[snake] = v
		} else {
			out[k] = v
		}
	}
	if _, ok := out["remain_amount"]; !ok {
		out["remain_amount"] = 0
	}
	if _, ok := out["cross_group_retry"]; !ok {
		out["cross_group_retry"] = false
	}

	result, _ := json.Marshal(out)
	return result
}

func transformNewapiTokens(body []byte) []byte {
	var raw struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil || !raw.Success || raw.Data == nil {
		return body
	}

	var items []map[string]interface{}

	var asItems struct {
		Items []map[string]interface{} `json:"items"`
	}
	if err := json.Unmarshal(raw.Data, &asItems); err == nil && len(asItems.Items) > 0 {
		items = asItems.Items
	} else {
		var arr []map[string]interface{}
		if err := json.Unmarshal(raw.Data, &arr); err == nil {
			items = arr
		}
	}
	if items == nil {
		return body
	}

	for i := range items {
		if key, ok := items[i]["key"].(string); ok && key != "" {
			items[i]["key"] = normalizeManagedTokenKey(key)
		}
	}

	out, _ := json.Marshal(gin.H{"success": true, "data": items})
	return out
}

func transformNewapiTokenKey(body []byte) []byte {
	var raw struct {
		Data *struct {
			Key string `json:"key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil || raw.Data == nil || raw.Data.Key == "" {
		return body
	}
	raw.Data.Key = normalizeManagedTokenKey(raw.Data.Key)
	out, _ := json.Marshal(raw)
	return out
}

func transformVoapiTokens(body []byte) []byte {
	var raw struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    *struct {
			Records []map[string]interface{} `json:"records"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil || raw.Code != 0 || raw.Data == nil {
		return body
	}

	type tokenItem struct {
		ID            string  `json:"id"`
		Name          string  `json:"name"`
		Key           string  `json:"key"`
		Group         string  `json:"group"`
		ExpiredTime   int64   `json:"expired_time"`
		UnlimitedQuota bool   `json:"unlimited_quota"`
		RemainQuota   int64   `json:"remain_quota"`
		UsedQuota     int64   `json:"used_quota"`
		Status        int     `json:"status"`
		CreatedTime   int64   `json:"created_time"`
		AccessedTime  int64   `json:"accessed_time"`
		UID           float64 `json:"uid,omitempty"`
	}

	items := make([]tokenItem, 0, len(raw.Data.Records))
	for _, r := range raw.Data.Records {
		id, _ := r["id"].(string)
		name, _ := r["name"].(string)
		token, _ := r["token"].(string)
		key, _ := r["key"].(string)
		if key == "" {
			key = token
		}

		groups, _ := r["groups"].([]interface{})
		var group string
		if len(groups) > 0 {
			group = fmt.Sprintf("%v", groups[0])
		}

		expireTime, _ := r["expireTime"].(float64)
		var expiredTime int64
		if expireTime == 4102329600000 {
			expiredTime = -1
		} else {
			expiredTime = int64(expireTime) / 1000
		}

		boundless, _ := r["boundlessAmount"].(bool)
		amount, _ := r["amount"].(string)
		used, _ := r["used"].(string)
		enable, _ := r["enable"].(bool)

		status := 0
		if enable {
			status = 1
		}

		var remainQ int64
		if !boundless {
			if a, err := strconv.ParseFloat(amount, 64); err == nil {
				remainQ = int64(a * 500000)
			}
		}
		var usedQ int64
		if u, err := strconv.ParseFloat(used, 64); err == nil {
			usedQ = int64(u * 500000)
		}

		created, _ := r["created"].(float64)
		updated, _ := r["updated"].(float64)
		uid, _ := r["uid"].(float64)

		items = append(items, tokenItem{
			ID:            id,
			Name:          name,
			Key:           normalizeManagedTokenKey(key),
			Group:         group,
			ExpiredTime:   expiredTime,
			UnlimitedQuota: boundless,
			RemainQuota:   remainQ,
			UsedQuota:     usedQ,
			Status:        status,
			CreatedTime:   int64(created) / 1000,
			AccessedTime:  int64(updated) / 1000,
			UID:           uid,
		})
	}

	out, _ := json.Marshal(gin.H{"success": true, "data": items})
	return out
}

func transformVoapiCreateRequest(body []byte) []byte {
	var req struct {
		Name           string   `json:"name"`
		RemainQuota    *float64 `json:"remainQuota"`
		UnlimitedQuota *bool    `json:"unlimitedQuota"`
		ExpiredTime    *float64 `json:"expiredTime"`
		Groups         []int    `json:"groups"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return body
	}

	payload := map[string]interface{}{
		"name":           req.Name,
		"amount":         "0",
		"boundlessAmount": false,
		"enable":         true,
		"expireTime":     int64(4102329600000),
		"genCount":       1,
		"groups":         []int{1},
	}
	if req.Name == "" {
		payload["name"] = "key-" + itoa(time.Now().UnixMilli())
	}
	if req.RemainQuota != nil && *req.RemainQuota > 0 {
		payload["amount"] = itoa(int64(*req.RemainQuota / 500000))
	}
	if req.UnlimitedQuota != nil {
		payload["boundlessAmount"] = *req.UnlimitedQuota
	}
	if req.ExpiredTime != nil {
		if *req.ExpiredTime == -1 {
			payload["expireTime"] = int64(4102329600000)
		} else {
			payload["expireTime"] = int64(*req.ExpiredTime * 1000)
		}
	}
	if len(req.Groups) > 0 {
		payload["groups"] = req.Groups
	}
	out, _ := json.Marshal(payload)
	return out
}

func transformVoapiCreateResponse(body []byte) []byte {
	var raw struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return body
	}
	if raw.Code == 0 {
		out, _ := json.Marshal(gin.H{"success": true, "message": "创建成功"})
		return out
	}
	out, _ := json.Marshal(gin.H{"success": false, "error": raw.Message})
	return out
}

func transformVoapiUpdateRequest(body []byte) []byte {
	var req struct {
		ID            string  `json:"id"`
		Name          string  `json:"name"`
		Key           string  `json:"key"`
		UnlimitedQuota bool   `json:"unlimited_quota"`
		RemainQuota   float64 `json:"remain_quota"`
		ExpiredTime   float64 `json:"expired_time"`
		Group         string  `json:"group"`
		UID           float64 `json:"uid"`
		UsedQuota     float64 `json:"used_quota"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return body
	}

	amount := "0"
	if !req.UnlimitedQuota {
		amount = itoa(int64(req.RemainQuota / 500000))
	}
	groups := []int{1}
	if req.Group != "" {
		if g, err := strconv.Atoi(req.Group); err == nil {
			groups = []int{g}
		}
	}
	expireTime := int64(4102329600000)
	if req.ExpiredTime != -1 {
		expireTime = int64(req.ExpiredTime * 1000)
	}
	used := "0"
	if req.UsedQuota > 0 {
		used = itoa(int64(req.UsedQuota / 500000))
	}

	payload := map[string]interface{}{
		"name":            req.Name,
		"amount":          amount,
		"boundlessAmount": req.UnlimitedQuota,
		"enable":          true,
		"expireTime":      expireTime,
		"groups":          groups,
		"token":           serializeManagedTokenKey(req.Key),
		"uid":             int64(req.UID),
		"used":            used,
	}
	out, _ := json.Marshal(payload)
	return out
}

func transformVoapiUpdateResponse(body []byte) []byte {
	var raw struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return body
	}
	if raw.Code == 0 {
		out, _ := json.Marshal(gin.H{"success": true, "message": "更新成功"})
		return out
	}
	out, _ := json.Marshal(gin.H{"success": false, "error": raw.Message})
	return out
}

func transformVoapiDeleteResponse(body []byte) []byte {
	var raw struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return body
	}
	if raw.Code == 0 {
		out, _ := json.Marshal(gin.H{"success": true, "message": "删除成功"})
		return out
	}
	out, _ := json.Marshal(gin.H{"success": false, "error": raw.Message})
	return out
}

func transformVoapiGroups(body []byte) []byte {
	var raw struct {
		Code int `json:"code"`
		Data *struct {
			Groups []map[string]interface{} `json:"groups"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil || raw.Code != 0 || raw.Data == nil {
		return body
	}

	normalized := make(map[string]gin.H)
	for _, g := range raw.Data.Groups {
		id, _ := g["id"].(float64)
		name, _ := g["name"].(string)
		if name == "" {
			name = fmt.Sprintf("分组%.0f", id)
		}
		idStr := itoa(int64(id))
		normalized[idStr] = gin.H{"name": name, "desc": name}
	}

	out, _ := json.Marshal(gin.H{"success": true, "data": normalized})
	return out
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}

func normalizeManagedTokenKey(key string) string {
	if !strings.HasPrefix(key, "sk-") {
		return "sk-" + key
	}
	return key
}

func serializeManagedTokenKey(key string) string {
	return strings.TrimPrefix(key, "sk-")
}

func normalizeProxyURL(url string) string {
	return strings.TrimRight(url, "/")
}

func (h *SiteHandler) ListTokens(c *gin.Context) {
	site, err := h.siteRepo.GetByID(c.Param("id"))
	if err != nil {
		c.JSON(404, gin.H{"error": "站点不存在"})
		return
	}
	pr := proxyRequest{
		pathByType: map[string]string{
			"newapi":  "/api/token/",
			"veloera": "/api/token/",
			"donehub": "/api/token/",
			"voapi":   "/api/keys",
		},
	}
	if site.APIType == "voapi" {
		pr.transform = transformVoapiTokens
	} else {
		pr.transform = transformNewapiTokens
	}
	h.doProxy(c, site, pr)
}

func (h *SiteHandler) CreateToken(c *gin.Context) {
	site, err := h.siteRepo.GetByID(c.Param("id"))
	if err != nil {
		c.JSON(404, gin.H{"error": "站点不存在"})
		return
	}
	pr := proxyRequest{
		pathByType: map[string]string{
			"newapi":  "/api/token/",
			"veloera": "/api/token/",
			"donehub": "/api/token/",
			"voapi":   "/api/keys",
		},
	}
	if site.APIType == "voapi" {
		pr.transformRequest = transformVoapiCreateRequest
		pr.transform = transformVoapiCreateResponse
	} else {
		pr.transformRequest = transformNewapiCreateRequest
	}
	h.doProxy(c, site, pr)
}

func (h *SiteHandler) UpdateToken(c *gin.Context) {
	site, err := h.siteRepo.GetByID(c.Param("id"))
	if err != nil {
		c.JSON(404, gin.H{"error": "站点不存在"})
		return
	}

	if site.APIType == "voapi" {
		bodyBytes, _ := io.ReadAll(c.Request.Body)
		var body struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(bodyBytes, &body); err != nil || body.ID == "" {
			c.JSON(400, gin.H{"error": "缺少令牌ID"})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(transformVoapiUpdateRequest(bodyBytes)))
		upstreamURL := strings.TrimRight(site.BaseURL, "/") + "/api/keys/" + body.ID
		apiKey, err := crypto.Decrypt(site.APIKeyEnc, h.encryptionKey)
		if err != nil {
			c.JSON(500, gin.H{"error": "解密失败"})
			return
		}
		req, _ := http.NewRequest("PUT", upstreamURL, c.Request.Body)
		req.Header.Set("Authorization", apiKey)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(502, gin.H{"error": "上游请求失败: " + err.Error()})
			return
		}
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, "application/json", transformVoapiUpdateResponse(respBody))
		return
	}

	h.doProxy(c, site, proxyRequest{
		pathByType: map[string]string{
			"newapi":  "/api/token/",
			"veloera": "/api/token/",
			"donehub": "/api/token/",
		},
	})
}

func (h *SiteHandler) DeleteToken(c *gin.Context) {
	site, err := h.siteRepo.GetByID(c.Param("id"))
	if err != nil {
		c.JSON(404, gin.H{"error": "站点不存在"})
		return
	}
	tokenID := c.Param("tokenId")
	pr := proxyRequest{
		pathByType: map[string]string{
			"newapi":  "/api/token/" + tokenID,
			"veloera": "/api/token/" + tokenID,
			"donehub": "/api/token/" + tokenID,
			"voapi":   "/api/keys/" + tokenID,
		},
	}
	if site.APIType == "voapi" {
		pr.transform = transformVoapiDeleteResponse
	}
	h.doProxy(c, site, pr)
}

func (h *SiteHandler) GetTokenKey(c *gin.Context) {
	site, err := h.siteRepo.GetByID(c.Param("id"))
	if err != nil {
		c.JSON(404, gin.H{"error": "站点不存在"})
		return
	}
	if site.APIType == "voapi" {
		c.JSON(400, gin.H{"error": "当前站点类型不需要单独获取完整令牌"})
		return
	}

	tokenID := c.Param("tokenId")
	h.doProxy(c, site, proxyRequest{
		pathByType: map[string]string{
			"newapi":  "/api/token/" + tokenID + "/key",
			"veloera": "/api/token/" + tokenID + "/key",
			"donehub": "/api/token/" + tokenID + "/key",
		},
		transform: transformNewapiTokenKey,
		extraHeaders: map[string]string{
			"Origin":        strings.TrimRight(site.BaseURL, "/"),
			"Referer":       strings.TrimRight(site.BaseURL, "/") + "/console/token",
			"Cache-Control": "no-store",
			"Accept":        "application/json, text/plain, */*",
		},
	})
}

func (h *SiteHandler) ListGroups(c *gin.Context) {
	site, err := h.siteRepo.GetByID(c.Param("id"))
	if err != nil {
		c.JSON(404, gin.H{"error": "站点不存在"})
		return
	}
	pr := proxyRequest{
		pathByType: map[string]string{
			"newapi":  "/api/user/self/groups",
			"veloera": "/api/user/self/groups",
			"donehub": "/api/user_group_map",
			"voapi":   "/api/models",
		},
	}
	if site.APIType == "voapi" {
		pr.transform = transformVoapiGroups
	}
	h.doProxy(c, site, pr)
}

func (h *SiteHandler) GetPricing(c *gin.Context) {
	site, err := h.siteRepo.GetByID(c.Param("id"))
	if err != nil {
		c.JSON(404, gin.H{"error": "站点不存在"})
		return
	}

	if site.APIType != "newapi" && site.APIType != "veloera" && site.APIType != "donehub" && site.APIType != "voapi" {
		c.JSON(400, gin.H{"error": "此站点类型不支持pricing接口"})
		return
	}

	extraHeaders := map[string]string{}
	if site.APIType == "voapi" {
		extraHeaders["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
		extraHeaders["Accept"] = "application/json"
	}

	h.doProxy(c, site, proxyRequest{
		pathByType: map[string]string{
			"newapi":  "/api/pricing",
			"veloera": "/api/pricing",
			"donehub": "/api/available_model",
			"voapi":   "/api/models",
		},
		skipAuth:     false,
		extraHeaders: extraHeaders,
		verbose:      true,
	})
}

func (h *SiteHandler) Redeem(c *gin.Context) {
	site, err := h.siteRepo.GetByID(c.Param("id"))
	if err != nil {
		c.JSON(404, gin.H{"error": "站点不存在"})
		return
	}
	h.doProxy(c, site, proxyRequest{
		pathByType: map[string]string{
			"newapi":  "/api/user/topup",
			"veloera": "/api/user/topup",
			"donehub": "/api/user/topup",
			"voapi":   "/api/user/topup",
		},
	})
}

func newID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 25)
	t := time.Now().UnixNano()
	for i := 0; i < 25; i++ {
		b[i] = chars[(t>>(i*2))%36]
	}
	return string(b)
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}
