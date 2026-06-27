package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"simplehub-go/internal/crypto"
	"simplehub-go/internal/model"
	"simplehub-go/internal/repository"
)

type ExportHandler struct {
	siteRepo      *repository.SiteRepository
	catRepo       *repository.CategoryRepository
	encryptionKey string
}

func NewExportHandler(siteRepo *repository.SiteRepository, catRepo *repository.CategoryRepository, encryptionKey string) *ExportHandler {
	return &ExportHandler{
		siteRepo:      siteRepo,
		catRepo:       catRepo,
		encryptionKey: encryptionKey,
	}
}

type exportSite struct {
	Name         string   `json:"name"`
	BaseURL      string   `json:"baseUrl"`
	APIKey       string   `json:"apiKey"`
	APIType      string   `json:"apiType"`
	UserID       *string  `json:"userId,omitempty"`
	BillingURL       *string `json:"billingUrl,omitempty"`
	BillingAuthType  string  `json:"billingAuthType"`
	BillingAuthValue *string `json:"billingAuthValue,omitempty"`
	ProxyURL         *string `json:"proxyUrl,omitempty"`
	BillingLimitField  *string `json:"billingLimitField,omitempty"`
	BillingUsageField  *string `json:"billingUsageField,omitempty"`
	UnlimitedQuota   bool       `json:"unlimitedQuota"`
	EnableCheckIn    bool       `json:"enableCheckIn"`
	CheckInMode      string     `json:"checkInMode"`
	ScheduleCron     *string    `json:"scheduleCron,omitempty"`
	Timezone         string     `json:"timezone"`
	Pinned           bool       `json:"pinned"`
	ExcludeFromBatch bool       `json:"excludeFromBatch"`
	CategoryName     *string    `json:"categoryName,omitempty"`
	Extralink        *string    `json:"extralink,omitempty"`
	Remark           *string    `json:"remark,omitempty"`
	SortOrder        int        `json:"sortOrder"`
}

func (h *ExportHandler) Export(c *gin.Context) {
	sites, err := h.siteRepo.List("")
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	categories, _ := h.catRepo.List()
	type exportCategory struct {
		Name         string  `json:"name"`
		ScheduleCron *string `json:"scheduleCron,omitempty"`
		Timezone     string  `json:"timezone"`
	}
	exportCats := make([]exportCategory, len(categories))
	for i, cat := range categories {
		exportCats[i] = exportCategory{
			Name:         cat.Name,
			ScheduleCron: cat.ScheduleCron,
			Timezone:     cat.Timezone,
		}
	}

	var exportSites []exportSite
	for _, site := range sites {
		apiKey, err := crypto.Decrypt(site.APIKeyEnc, h.encryptionKey)
		if err != nil {
			continue
		}
		es := exportSite{
			Name:         site.Name,
			BaseURL:      site.BaseURL,
			APIKey:       apiKey,
			APIType:      site.APIType,
			UserID:       site.UserID,
			BillingURL:   site.BillingURL,
			BillingAuthType: site.BillingAuthType,
			BillingLimitField:  site.BillingLimitField,
			BillingUsageField:  site.BillingUsageField,
			UnlimitedQuota:   site.UnlimitedQuota,
			EnableCheckIn:    site.EnableCheckIn,
			CheckInMode:      site.CheckInMode,
			ScheduleCron:     site.ScheduleCron,
			Timezone:         site.Timezone,
			Pinned:           site.Pinned,
			ExcludeFromBatch: site.ExcludeFromBatch,
			Extralink:        site.Extralink,
			Remark:           site.Remark,
			SortOrder:        site.SortOrder,
		}
		if site.Category != nil {
			es.CategoryName = &site.Category.Name
		}
		if site.ProxyURLEnc != nil && *site.ProxyURLEnc != "" {
			if dec, err := crypto.Decrypt(*site.ProxyURLEnc, h.encryptionKey); err == nil {
				es.ProxyURL = &dec
			}
		}
		if site.BillingAuthValue != nil && *site.BillingAuthValue != "" {
			if dec, err := crypto.Decrypt(*site.BillingAuthValue, h.encryptionKey); err == nil {
				es.BillingAuthValue = &dec
			}
		}
		exportSites = append(exportSites, es)
	}

	result := map[string]interface{}{
		"version":    "1.2",
		"exportDate": time.Now(),
		"categories": exportCats,
		"sites":      exportSites,
	}

	Data(c, result)
}

type importSite struct {
	Name         string  `json:"name"`
	BaseURL      string  `json:"baseUrl"`
	APIKey       string  `json:"apiKey"`
	APIType      string  `json:"apiType"`
	UserID       *string `json:"userId,omitempty"`
	BillingURL       *string `json:"billingUrl,omitempty"`
	BillingAuthType  string  `json:"billingAuthType"`
	BillingAuthValue *string `json:"billingAuthValue,omitempty"`
	ProxyURL         *string `json:"proxyUrl,omitempty"`
	BillingLimitField  *string `json:"billingLimitField,omitempty"`
	BillingUsageField  *string `json:"billingUsageField,omitempty"`
	UnlimitedQuota   bool       `json:"unlimitedQuota"`
	EnableCheckIn    bool       `json:"enableCheckIn"`
	CheckInMode      string     `json:"checkInMode"`
	ScheduleCron     *string    `json:"scheduleCron,omitempty"`
	Timezone         string     `json:"timezone"`
	Pinned           bool       `json:"pinned"`
	ExcludeFromBatch bool       `json:"excludeFromBatch"`
	CategoryName     *string    `json:"categoryName,omitempty"`
	Extralink        *string    `json:"extralink,omitempty"`
	Remark           *string    `json:"remark,omitempty"`
	SortOrder        int        `json:"sortOrder"`
}

func (h *ExportHandler) Import(c *gin.Context) {
	var req struct {
		Version    string `json:"version"`
		Categories []struct {
			Name         string  `json:"name"`
			ScheduleCron *string `json:"scheduleCron,omitempty"`
			Timezone     string  `json:"timezone"`
		} `json:"categories,omitempty"`
		Sites []importSite `json:"sites"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "无效的导入数据")
		return
	}

	if len(req.Sites) == 0 {
		Fail(c, http.StatusBadRequest, "没有需要导入的站点")
		return
	}

	catNameMap := make(map[string]*model.Category)
	existingCats, _ := h.catRepo.List()
	for _, ec := range existingCats {
		catNameMap[ec.Name] = &ec
	}
	for _, sc := range req.Categories {
		if _, exists := catNameMap[sc.Name]; exists {
			continue
		}
		timezone := sc.Timezone
		if timezone == "" {
			timezone = "Asia/Shanghai"
		}
		cat := &model.Category{
			ID:           newID(),
			Name:         sc.Name,
			ScheduleCron: sc.ScheduleCron,
			Timezone:     timezone,
		}
		if err := h.catRepo.Create(cat); err == nil {
			catNameMap[sc.Name] = cat
		}
	}

	type importError struct {
		Index int    `json:"index"`
		Name  string `json:"name"`
		Error string `json:"error"`
	}
	var errors []importError
	imported := 0
	total := len(req.Sites)

	for idx, s := range req.Sites {
		if s.Name == "" {
			errors = append(errors, importError{Index: idx, Name: s.Name, Error: "缺少站点名称"})
			continue
		}
		if s.BaseURL == "" {
			errors = append(errors, importError{Index: idx, Name: s.Name, Error: "缺少接口地址"})
			continue
		}
		if s.APIKey == "" {
			errors = append(errors, importError{Index: idx, Name: s.Name, Error: "缺少API密钥"})
			continue
		}

		apiKeyEnc, err := crypto.Encrypt(s.APIKey, h.encryptionKey)
		if err != nil {
			errors = append(errors, importError{Index: idx, Name: s.Name, Error: "加密失败: " + err.Error()})
			continue
		}

		apiType := s.APIType
		if apiType == "" {
			apiType = "other"
		}
		timezone := s.Timezone
		if timezone == "" {
			timezone = "UTC"
		}
		checkInMode := s.CheckInMode
		if checkInMode == "" {
			checkInMode = "both"
		}
		billingAuthType := s.BillingAuthType
		if billingAuthType == "" {
			billingAuthType = "token"
		}

		site := &model.Site{
			ID:               newID(),
			Name:             s.Name,
			BaseURL:          s.BaseURL,
			APIKeyEnc:        apiKeyEnc,
			APIType:          apiType,
			UserID:           s.UserID,
			BillingURL:       s.BillingURL,
			BillingAuthType:  billingAuthType,
			BillingLimitField:  s.BillingLimitField,
			BillingUsageField:  s.BillingUsageField,
			UnlimitedQuota:   s.UnlimitedQuota,
			EnableCheckIn:    s.EnableCheckIn,
			CheckInMode:      checkInMode,
			ScheduleCron:     s.ScheduleCron,
			Timezone:         timezone,
			Pinned:           s.Pinned,
			ExcludeFromBatch: s.ExcludeFromBatch,
			Extralink:        s.Extralink,
			Remark:           s.Remark,
			SortOrder:        s.SortOrder,
		}
		if s.ProxyURL != nil && *s.ProxyURL != "" {
			normalized := normalizeProxyURL(*s.ProxyURL)
			if enc, err := crypto.Encrypt(normalized, h.encryptionKey); err == nil {
				site.ProxyURLEnc = &enc
			}
		}
		if s.BillingAuthValue != nil && *s.BillingAuthValue != "" {
			if enc, err := crypto.Encrypt(*s.BillingAuthValue, h.encryptionKey); err == nil {
				site.BillingAuthValue = &enc
			}
		}
		if s.CategoryName != nil && *s.CategoryName != "" {
			if cat, ok := catNameMap[*s.CategoryName]; ok {
				site.CategoryID = &cat.ID
			}
		}
		if err := h.siteRepo.Create(site); err != nil {
			errors = append(errors, importError{Index: idx, Name: s.Name, Error: "创建失败: " + err.Error()})
			continue
		}
		imported++
	}

	resp := gin.H{
		"imported": imported,
		"total":    total,
	}
	if len(errors) > 0 {
		resp["errors"] = errors
	}
	Data(c, resp)
}
