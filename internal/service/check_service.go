package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	"simplehub-go/internal/checker"
	"simplehub-go/internal/crypto"
	"simplehub-go/internal/model"
	"simplehub-go/internal/proxy"
	"simplehub-go/internal/repository"
)

type CheckService struct {
	siteRepo      *repository.SiteRepository
	snapRepo      *repository.SnapshotRepository
	diffRepo      *repository.DiffRepository
	proxyClient   *proxy.ProxyClient
	encryptionKey string
	notifService  *NotificationService
}

func NewCheckService(
	siteRepo *repository.SiteRepository,
	snapRepo *repository.SnapshotRepository,
	diffRepo *repository.DiffRepository,
	proxyClient *proxy.ProxyClient,
	encryptionKey string,
	notifService *NotificationService,
) *CheckService {
	return &CheckService{
		siteRepo:      siteRepo,
		snapRepo:      snapRepo,
		diffRepo:      diffRepo,
		proxyClient:   proxyClient,
		encryptionKey: encryptionKey,
		notifService:  notifService,
	}
}

func (s *CheckService) CheckSite(siteID string, skipNotification bool, isManual ...bool) (*checker.CheckResult, error) {
	manual := false
	if len(isManual) > 0 {
		manual = isManual[0]
	}
	site, err := s.siteRepo.GetByID(siteID)
	if err != nil {
		return nil, err
	}

	apiKey, err := crypto.Decrypt(site.APIKeyEnc, s.encryptionKey)
	if err != nil {
		return nil, err
	}

	var proxyURL string
	if site.ProxyURLEnc != nil && *site.ProxyURLEnc != "" {
		dec, err := crypto.Decrypt(*site.ProxyURLEnc, s.encryptionKey)
		if err == nil {
			proxyURL = dec
		}
	}

	var userID string
	if site.UserID != nil {
		userID = *site.UserID
	}

	var billingLimitF, billingUsageF string
	if site.BillingLimitField != nil {
		billingLimitF = *site.BillingLimitField
	}
	if site.BillingUsageField != nil {
		billingUsageF = *site.BillingUsageField
	}

	var billingURL string
	if site.BillingURL != nil {
		billingURL = *site.BillingURL
	}

	var billingAuthValue string
	if site.BillingAuthValue != nil && *site.BillingAuthValue != "" {
		dec, err := crypto.Decrypt(*site.BillingAuthValue, s.encryptionKey)
		if err == nil {
			billingAuthValue = dec
		}
	}

	runner := checker.NewRunner(s.proxyClient)
	result := runner.Run(checker.Config{
		APIKey:            apiKey,
		BaseURL:           site.BaseURL,
		ProxyURL:          proxyURL,
		APIType:           site.APIType,
		UserID:            userID,
		BillingURL:        billingURL,
		BillingLimitField: billingLimitF,
		BillingUsageField: billingUsageF,
		BillingAuthType:   site.BillingAuthType,
		BillingAuthValue:  billingAuthValue,
		UnlimitedQuota:    site.UnlimitedQuota,
		CheckInMode:       site.CheckInMode,
		EnableCheckIn:     site.EnableCheckIn,
		IsManual:          manual,
	})

	prevSnap, _ := s.snapRepo.GetLatest(siteID)

	modelsJSON := modelsToJSON(result.Models)
	snapHash := computeHash(result.Models)

	snap := &model.ModelSnapshot{
		ID:           newID(),
		SiteID:       siteID,
		FetchedAt:    time.Now(),
		ModelsJSON:   modelsJSON,
		Hash:         snapHash,
		RawResponse:  strPtr(result.RawResponse),
		ErrorMessage: strPtrOrNil(result.ErrorMessage),
		StatusCode:   intPtr(result.StatusCode),
		ResponseTime: intPtr(result.ResponseTime),
		BillingLimit: result.BillingLimit,
		BillingUsage: result.BillingUsage,
		BillingError: strPtrOrNil(result.BillingError),
		CheckInSuccess: result.CheckInSuccess,
		CheckInMessage: strPtrOrNil(result.CheckInMessage),
		CheckInQuota:   result.CheckInQuota,
		CheckInError:   strPtrOrNil(result.CheckInError),
	}
	if err := s.snapRepo.Create(snap); err != nil {
		return nil, err
	}

	hasChanges := prevSnap == nil || prevSnap.Hash != snapHash
	if hasChanges && prevSnap != nil {
		diff := computeDiff(prevSnap, result.Models)
		diffRec := &model.ModelDiff{
			ID:             newID(),
			SiteID:         siteID,
			DiffAt:         time.Now(),
			AddedJSON:      toJSON(diff.Added),
			RemovedJSON:    toJSON(diff.Removed),
			ChangedJSON:    toJSON(diff.Changed),
			SnapshotToID:   &snap.ID,
			SnapshotFromID: &prevSnap.ID,
		}
		if err := s.diffRepo.Create(diffRec); err != nil {
			return nil, err
		}

		if !skipNotification && s.notifService != nil {
			s.notifService.SendModelChangeNotification(site.Name, diff, nil)
		}
	}

	s.siteRepo.Update(siteID, map[string]interface{}{
		"last_checked_at": time.Now(),
	})

	return result, nil
}

type DiffResult struct {
	Added   []checker.ModelInfo `json:"added"`
	Removed []checker.ModelInfo `json:"removed"`
	Changed []checker.ModelInfo `json:"changed"`
}

func computeDiff(prev *model.ModelSnapshot, currModels []checker.ModelInfo) DiffResult {
	if prev == nil || prev.ModelsJSON == "" {
		return DiffResult{}
	}

	var prevModels []checker.ModelInfo
	if err := json.Unmarshal([]byte(prev.ModelsJSON), &prevModels); err != nil {
		return DiffResult{}
	}

	prevMap := make(map[string]checker.ModelInfo)
	currMap := make(map[string]checker.ModelInfo)

	for _, m := range prevModels {
		prevMap[m.ID] = m
	}
	for _, m := range currModels {
		currMap[m.ID] = m
	}

	var diff DiffResult
	for id, m := range currMap {
		if _, ok := prevMap[id]; !ok {
			diff.Added = append(diff.Added, m)
		}
	}
	for id, m := range prevMap {
		if _, ok := currMap[id]; !ok {
			diff.Removed = append(diff.Removed, m)
		}
	}
	return diff
}

func modelsToJSON(models []checker.ModelInfo) string {
	if models == nil {
		return "[]"
	}
	b, _ := json.Marshal(models)
	return string(b)
}

func toJSON(v interface{}) string {
	if v == nil {
		return "[]"
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func computeHash(models []checker.ModelInfo) string {
	sorted := make([]checker.ModelInfo, len(models))
	copy(sorted, models)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})
	data, _ := json.Marshal(sorted)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func intPtr(i int) *int {
	return &i
}


