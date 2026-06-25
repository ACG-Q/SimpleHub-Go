package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"simplehub-go/internal/config"
	"simplehub-go/internal/handler"
	"simplehub-go/internal/model"
	"simplehub-go/internal/proxy"
	"simplehub-go/internal/repository"
	"simplehub-go/internal/router"
	"simplehub-go/internal/service"
)

func main() {
	dbPath := flag.String("db", "data/db.sqlite", "database file path")
	flag.Parse()

	if !filepath.IsAbs(*dbPath) {
		exe, err := os.Executable()
		if err == nil {
			*dbPath = filepath.Join(filepath.Dir(exe), *dbPath)
		}
	}

	reset := false
	args := flag.Args()
	if len(args) > 0 && strings.ToLower(args[0]) == "reset" {
		reset = true
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	db, err := model.InitDatabase(*dbPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize database")
	}

	setupSvc := service.NewSetupService(db)

	var result *service.InitResult
	if reset {
		log.Warn().Msg("resetting admin configuration...")
		result, err = setupSvc.Reset()
	} else {
		result, err = setupSvc.InitApp()
	}
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize app")
	}

	if result.IsFirst {
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("  SimpleHub")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("  端口:          %d\n", result.Port)
		fmt.Printf("  安全入口:      http://localhost:%d/%s\n", result.Port, result.Entry)
		fmt.Printf("  管理员账号:    %s\n", result.Username)
		fmt.Printf("  管理员密码:    %s\n", result.Password)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("  请立即记录以上信息。JWT 密钥已自动生成。")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
	}

	cfg := &config.Config{
		Port:          result.Port,
		JWTSecret:     setupSvc.GetJWTSecret(),
		EncryptionKey: setupSvc.GetEncryptionKey(),
		SkipAuth:      false,
		LogLevel:      "info",
	}

	log.Info().Int("port", cfg.Port).Msg("starting SimpleHub Go server")

	authService := service.NewAuthService(db, cfg.JWTSecret)

	authHandler := handler.NewAuthHandler(authService)

	siteRepo := repository.NewSiteRepository(db)
	snapRepo := repository.NewSnapshotRepository(db)
	diffRepo := repository.NewDiffRepository(db)

	proxyClient := proxy.NewProxyClient()

	catRepo := repository.NewCategoryRepository(db)
	emailRepo := repository.NewEmailConfigRepository(db)
	schedRepo := repository.NewScheduleConfigRepository(db)

	notifService := service.NewNotificationService(emailRepo, cfg.EncryptionKey)
	checkService := service.NewCheckService(siteRepo, snapRepo, diffRepo, proxyClient, cfg.EncryptionKey, notifService)
	schedulerService := service.NewSchedulerService(siteRepo, catRepo, schedRepo, checkService, notifService)

	siteHandler := handler.NewSiteHandler(siteRepo, snapRepo, diffRepo, authService, checkService, schedulerService, cfg.EncryptionKey)
	catHandler := handler.NewCategoryHandler(catRepo, checkService, schedulerService)
	emailHandler := handler.NewEmailHandler(emailRepo, cfg.EncryptionKey, notifService)
	scheduleHandler := handler.NewScheduleHandler(schedRepo, schedulerService)
	exportHandler := handler.NewExportHandler(siteRepo, catRepo, cfg.EncryptionKey)

	schedulerService.ScheduleAll()
	defer schedulerService.Stop()

	distFS := getDistFS()
	r := router.Setup(cfg, authService, authHandler, siteHandler, catHandler, emailHandler, scheduleHandler, exportHandler, distFS)

	srv := &http.Server{
		Addr:    ":" + itoa(cfg.Port),
		Handler: r,
	}

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Info().Msg("shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal().Err(err).Msg("server forced to shutdown")
		}
	}()

	log.Info().Int("port", cfg.Port).Msg("server listening")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("server failed")
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [12]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
