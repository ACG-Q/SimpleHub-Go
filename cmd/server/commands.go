package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"simplehub-go/internal/config"
	"simplehub-go/internal/crypto"
	"simplehub-go/internal/handler"
	"simplehub-go/internal/model"
	"simplehub-go/internal/proxy"
	"simplehub-go/internal/repository"
	"simplehub-go/internal/router"
	"simplehub-go/internal/service"
)

func printHelp() {
	fmt.Println(`SimpleHub - AI Relay Model Monitor

Usage:
  server.exe                    Start the server
  server.exe help               Show this help
  server.exe info               Show configuration info
  server.exe version            Show version
  server.exe reset              Reset admin credentials
  server.exe re-encrypt         Re-encrypt all sensitive data with current key
  server.exe db backup [path]   Backup SQLite database
  server.exe db vacuum          Reclaim SQLite disk space (VACUUM)
  server.exe db set <path>      Set persistent database path override
  server.exe db set             Reset to default database path`)
}

func resolveDBPath(exeDir string) string {
	defaultPath := filepath.Join(exeDir, "data", "db.sqlite")

	if _, err := os.Stat(defaultPath); err != nil {
		return defaultPath
	}

	db, err := gorm.Open(sqlite.Open(defaultPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return defaultPath
	}
	sqlDB, _ := db.DB()
	cfgPath := model.GetConfig(db, "db_path")
	if sqlDB != nil {
		sqlDB.Close()
	}

	if cfgPath == "" {
		return defaultPath
	}
	if filepath.IsAbs(cfgPath) {
		return cfgPath
	}
	return filepath.Join(exeDir, cfgPath)
}

func openDB(exeDir string) *gorm.DB {
	dbPath := resolveDBPath(exeDir)
	db, err := model.InitDatabase(dbPath)
	if err != nil {
		log.Fatal().Err(err).Str("path", dbPath).Msg("failed to open database")
	}
	return db
}

func startServer(exeDir string) {
	dbPath := resolveDBPath(exeDir)

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	db, err := model.InitDatabase(dbPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize database")
	}

	setupSvc := service.NewSetupService(db)
	result, err := setupSvc.InitApp()
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
	dashRepo := repository.NewDashboardRepository(db)
	dashboardHandler := handler.NewDashboardHandler(dashRepo)

	schedulerService.ScheduleAll()
	defer schedulerService.Stop()

	distFS := getDistFS()
	r := router.Setup(cfg, authService, authHandler, siteHandler, catHandler, emailHandler, scheduleHandler, exportHandler, dashboardHandler, distFS)

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

func cmdInfo(exeDir string) {
	db := openDB(exeDir)

	port := model.GetConfig(db, "port")
	entry := model.GetConfig(db, "admin_entry")
	encKey := model.GetConfig(db, "encryption_key")

	var siteCount int64
	db.Model(&model.Site{}).Count(&siteCount)

	var userCount int64
	db.Model(&model.User{}).Count(&userCount)

	var catCount int64
	db.Model(&model.Category{}).Count(&catCount)

	dbPath := resolveDBPath(exeDir)

	fmt.Println("SimpleHub Configuration")
	fmt.Println("─────────────────────────────")
	fmt.Printf("  Database:     %s\n", dbPath)
	fmt.Printf("  Port:         %s\n", port)
	fmt.Printf("  Admin Entry:  %s\n", entry)
	fmt.Printf("  Encryption:   %s\n", map[bool]string{true: "configured", false: "MISSING"}[encKey != ""])
	fmt.Printf("  Users:        %d\n", userCount)
	fmt.Printf("  Sites:        %d\n", siteCount)
	fmt.Printf("  Categories:   %d\n", catCount)
}

func cmdReset(exeDir string) {
	db := openDB(exeDir)
	setupSvc := service.NewSetupService(db)

	result, err := setupSvc.Reset()
	if err != nil {
		log.Fatal().Err(err).Msg("reset failed")
	}

	fmt.Println("Reset complete. Admin credentials updated:")
	fmt.Printf("  Port:          %d\n", result.Port)
	fmt.Printf("  Admin Entry:   %s\n", result.Entry)
	fmt.Printf("  Admin Account: %s\n", result.Username)
	fmt.Printf("  Admin Password:%s\n", result.Password)
	fmt.Println()
	fmt.Println("Encryption key was preserved. Encrypted data remains readable.")
}

func cmdReEncrypt(exeDir string) {
	db := openDB(exeDir)
	encKey := model.GetConfig(db, "encryption_key")
	if encKey == "" {
		log.Fatal().Msg("encryption_key not found. Has the server been initialized?")
	}

	var sites []model.Site
	db.Find(&sites)
	for _, s := range sites {
		if s.APIKeyEnc != "" {
			plain, err := crypto.Decrypt(s.APIKeyEnc, encKey)
			if err != nil {
				log.Error().Err(err).Str("site", s.ID).Msg("failed to decrypt api_key, skipping")
				continue
			}
			re, err := crypto.Encrypt(plain, encKey)
			if err != nil {
				log.Error().Err(err).Str("site", s.ID).Msg("failed to re-encrypt api_key")
				continue
			}
			db.Model(&model.Site{}).Where("id = ?", s.ID).Update("api_key_enc", re)
		}
		if s.ProxyURLEnc != nil && *s.ProxyURLEnc != "" {
			plain, err := crypto.Decrypt(*s.ProxyURLEnc, encKey)
			if err != nil {
				log.Error().Err(err).Str("site", s.ID).Msg("failed to decrypt proxy_url, skipping")
				continue
			}
			re, err := crypto.Encrypt(plain, encKey)
			if err != nil {
				log.Error().Err(err).Str("site", s.ID).Msg("failed to re-encrypt proxy_url")
				continue
			}
			db.Model(&model.Site{}).Where("id = ?", s.ID).Update("proxy_url_enc", re)
		}
		if s.BillingAuthValue != nil && *s.BillingAuthValue != "" {
			plain, err := crypto.Decrypt(*s.BillingAuthValue, encKey)
			if err != nil {
				log.Error().Err(err).Str("site", s.ID).Msg("failed to decrypt billing_auth, skipping")
				continue
			}
			re, err := crypto.Encrypt(plain, encKey)
			if err != nil {
				log.Error().Err(err).Str("site", s.ID).Msg("failed to re-encrypt billing_auth")
				continue
			}
			db.Model(&model.Site{}).Where("id = ?", s.ID).Update("billing_auth_value", re)
		}
	}

	var emails []model.EmailConfig
	db.Find(&emails)
	for _, e := range emails {
		if e.ResendAPIKeyEnc != "" {
			plain, err := crypto.Decrypt(e.ResendAPIKeyEnc, encKey)
			if err != nil {
				log.Error().Err(err).Str("id", e.ID).Msg("failed to decrypt resend_api_key, skipping")
				continue
			}
			re, err := crypto.Encrypt(plain, encKey)
			if err != nil {
				log.Error().Err(err).Str("id", e.ID).Msg("failed to re-encrypt resend_api_key")
				continue
			}
			db.Model(&model.EmailConfig{}).Where("id = ?", e.ID).Update("resend_api_key_enc", re)
		}
	}

	fmt.Println("Re-encryption complete.")
}

func runDBCommand(exeDir string, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: server.exe db <backup|vacuum|set>")
		fmt.Fprintln(os.Stderr, "Run 'server.exe help' for details")
		os.Exit(1)
	}

	switch args[0] {
	case "backup":
		backupPath := ""
		if len(args) > 1 {
			backupPath = args[1]
		}
		cmdDBBackup(exeDir, backupPath)
	case "vacuum":
		cmdDBVacuum(exeDir)
	case "set":
		path := ""
		if len(args) > 1 {
			path = args[1]
		}
		cmdDBSet(exeDir, path)
	default:
		fmt.Fprintf(os.Stderr, "Unknown db subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Run 'server.exe help' for details")
		os.Exit(1)
	}
}

func cmdDBBackup(exeDir, backupPath string) {
	src := resolveDBPath(exeDir)

	if _, err := os.Stat(src); err != nil {
		log.Fatal().Err(err).Str("path", src).Msg("database file not found")
	}

	if backupPath == "" {
		backupPath = src + ".backup"
	} else if !filepath.IsAbs(backupPath) {
		backupPath = filepath.Join(exeDir, backupPath)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open source database")
	}
	defer srcFile.Close()

	dstFile, err := os.Create(backupPath)
	if err != nil {
		log.Fatal().Err(err).Str("path", backupPath).Msg("failed to create backup file")
	}
	defer dstFile.Close()

	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		log.Fatal().Err(err).Msg("backup copy failed")
	}

	fmt.Printf("Backup saved to: %s (%d bytes)\n", backupPath, written)
}

func cmdDBVacuum(exeDir string) {
	db := openDB(exeDir)
	if err := db.Exec("VACUUM").Error; err != nil {
		log.Fatal().Err(err).Msg("VACUUM failed")
	}
	fmt.Println("Database VACUUM complete.")
}

func cmdDBSet(exeDir, path string) {
	defaultPath := filepath.Join(exeDir, "data", "db.sqlite")

	os.MkdirAll(filepath.Dir(defaultPath), 0755)
	db, err := model.InitDatabase(defaultPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open default database")
	}

	if path == "" {
		model.SetConfig(db, "db_path", "")
		db.Delete(&model.AppConfig{}, "key = ?", "db_path")
		fmt.Println("Database path override cleared. Will use default.")
	} else {
		model.SetConfig(db, "db_path", path)
		resolved := path
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(exeDir, resolved)
		}
		absDefault := defaultPath
		if abs, err := filepath.Abs(defaultPath); err == nil {
			absDefault = abs
		}
		fmt.Printf("Database path set to: %s\n", resolved)
		fmt.Printf("  (stored in: %s)\n", absDefault)
		fmt.Println("Restart the server for the change to take effect.")
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
