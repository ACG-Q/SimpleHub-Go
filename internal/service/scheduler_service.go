package service

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"

	"simplehub-go/internal/model"
	"simplehub-go/internal/repository"
)

type SchedulerService struct {
	cron            *cron.Cron
	siteRepo        *repository.SiteRepository
	catRepo         *repository.CategoryRepository
	schedRepo       *repository.ScheduleConfigRepository
	checkService    *CheckService
	notifService    *NotificationService
	jobs            map[string]cron.EntryID
}

func NewSchedulerService(
	siteRepo *repository.SiteRepository,
	catRepo *repository.CategoryRepository,
	schedRepo *repository.ScheduleConfigRepository,
	checkService *CheckService,
	notifService *NotificationService,
) *SchedulerService {
	return &SchedulerService{
		cron:         cron.New(cron.WithSeconds()),
		siteRepo:     siteRepo,
		catRepo:      catRepo,
		schedRepo:    schedRepo,
		checkService: checkService,
		notifService: notifService,
		jobs:         make(map[string]cron.EntryID),
	}
}

func (s *SchedulerService) Start() {
	s.cron.Start()
	log.Info().Msg("scheduler started")
}

func (s *SchedulerService) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Info().Msg("scheduler stopped")
}

func (s *SchedulerService) ScheduleAll() {
	schedCfg, err := s.schedRepo.GetOrCreate()
	if err != nil {
		log.Error().Err(err).Msg("failed to get schedule config")
		return
	}

	if schedCfg.Enabled {
		s.scheduleGlobalTask(schedCfg)
	}

	sites, err := s.siteRepo.List("")
	if err != nil {
		log.Error().Err(err).Msg("failed to list sites for scheduling")
		return
	}
	for _, site := range sites {
		if site.ScheduleCron != nil && *site.ScheduleCron != "" {
			s.scheduleSiteTask(&site)
		}
	}

	categories, err := s.catRepo.List()
	if err != nil {
		log.Error().Err(err).Msg("failed to list categories for scheduling")
		return
	}
	for _, cat := range categories {
		if cat.ScheduleCron != nil && *cat.ScheduleCron != "" {
			s.scheduleCategoryTask(&cat)
		}
	}

	log.Info().Int("jobs", len(s.jobs)).Msg("all tasks scheduled")
}

func (s *SchedulerService) scheduleGlobalTask(cfg *model.ScheduleConfig) {
	expr := fmt.Sprintf("0 %d %d * * *", cfg.Minute, cfg.Hour)
	entryID, err := s.cron.AddFunc(expr, func() {
		s.runGlobalCheck(cfg)
	})
	if err != nil {
		log.Error().Err(err).Str("expr", expr).Msg("failed to schedule global task")
		return
	}
	s.jobs["global"] = entryID
	log.Info().Str("expr", expr).Msg("global task scheduled")
}

func (s *SchedulerService) scheduleSiteTask(site *model.Site) {
	if site.ScheduleCron == nil || *site.ScheduleCron == "" {
		return
	}
	expr := *site.ScheduleCron
	if expr[0] != '0' {
		expr = "0 " + expr
	}
	siteID := site.ID
	entryID, err := s.cron.AddFunc(expr, func() {
		log.Info().Str("site", siteID).Msg("running scheduled site check")
		if _, err := s.checkService.CheckSite(siteID, false); err != nil {
			log.Error().Err(err).Str("site", siteID).Msg("scheduled site check failed")
		}
	})
	if err != nil {
		log.Error().Err(err).Str("site", siteID).Msg("failed to schedule site task")
		return
	}
	s.jobs["site:"+siteID] = entryID
}

func (s *SchedulerService) scheduleCategoryTask(cat *model.Category) {
	if cat.ScheduleCron == nil || *cat.ScheduleCron == "" {
		return
	}
	expr := *cat.ScheduleCron
	if expr[0] != '0' {
		expr = "0 " + expr
	}
	catID := cat.ID
	entryID, err := s.cron.AddFunc(expr, func() {
		s.runCategoryCheck(catID)
	})
	if err != nil {
		log.Error().Err(err).Str("category", catID).Msg("failed to schedule category task")
		return
	}
	s.jobs["cat:"+catID] = entryID
}

// TEST: 手动触发全局检测
func (s *SchedulerService) RunGlobalCheck() {
	cfg, err := s.schedRepo.GetOrCreate()
	if err != nil {
		log.Error().Err(err).Msg("global check: failed to get config")
		return
	}
	s.runGlobalCheck(cfg)
}

func (s *SchedulerService) runGlobalCheck(cfg *model.ScheduleConfig) {
	log.Info().Msg("starting global check")
	sites, err := s.siteRepo.List("")
	if err != nil {
		log.Error().Err(err).Msg("global check: failed to list sites")
		return
	}

	var reports []SiteCheckReport
	for i, site := range sites {
		if site.ExcludeFromBatch {
			continue
		}
		if !cfg.OverrideIndividual && site.ScheduleCron != nil && *site.ScheduleCron != "" {
			continue
		}

		report := SiteCheckReport{Name: site.Name}
		sr, err := s.checkService.CheckSite(site.ID, true)
		if err != nil {
			report.Error = err.Error()
		} else {
			report.ModelCount = len(sr.Result.Models)
			if sr.Result.CheckInSuccess != nil && *sr.Result.CheckInSuccess {
				report.CheckInOK = true
			}
			if sr.Result.CheckInMessage != "" {
				report.CheckInMsg = sr.Result.CheckInMessage
			}
			report.CheckInQuota = sr.Result.CheckInQuota
			report.BillingLimit = sr.Result.BillingLimit
			report.BillingUsage = sr.Result.BillingUsage
		}
		reports = append(reports, report)

		if i < len(sites)-1 && cfg.Interval > 0 {
			time.Sleep(time.Duration(cfg.Interval) * time.Second)
		}
	}

	if s.notifService != nil {
		s.notifService.SendAggregatedNotification(reports)
	}

	s.schedRepo.Update(cfg.ID, map[string]interface{}{
		"last_run": time.Now(),
	})
	log.Info().Int("checked", len(sites)).Msg("global check completed")
}

func (s *SchedulerService) runCategoryCheck(catID string) {
	log.Info().Str("category", catID).Msg("starting category check")
	cat, err := s.catRepo.GetByID(catID)
	if err != nil {
		log.Error().Err(err).Str("category", catID).Msg("category check: failed to get category")
		return
	}

	for i, site := range cat.Sites {
		result, err := s.checkService.CheckSite(site.ID, true)
		if err != nil {
			log.Error().Err(err).Str("site", site.ID).Msg("category check site failed")
			continue
		}
		_ = result

		if i < len(cat.Sites)-1 {
			time.Sleep(5 * time.Second)
		}
	}
	log.Info().Str("category", catID).Int("sites", len(cat.Sites)).Msg("category check completed")
}

func (s *SchedulerService) ScheduleNowForSite(siteID string) {
	s.checkService.CheckSite(siteID, true)
}

func (s *SchedulerService) RemoveSiteTask(siteID string) {
	if entryID, ok := s.jobs["site:"+siteID]; ok {
		s.cron.Remove(entryID)
		delete(s.jobs, "site:"+siteID)
	}
}
