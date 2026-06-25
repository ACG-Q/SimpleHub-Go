package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simplehub-go/internal/config"
	"simplehub-go/internal/handler"
	"simplehub-go/internal/middleware"
	"simplehub-go/internal/service"
)

func Setup(cfg *config.Config, authService *service.AuthService,
	authHandler *handler.AuthHandler,
	siteHandler *handler.SiteHandler,
	catHandler *handler.CategoryHandler,
	emailHandler *handler.EmailHandler,
	scheduleHandler *handler.ScheduleHandler,
	exportHandler *handler.ExportHandler,
	distFS http.FileSystem) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.CORSMiddleware())

	authMw := middleware.AuthMiddleware(authService, cfg.SkipAuth)

	api := r.Group("/api")
	api.Use(authMw)
	{
		api.GET("/sites", siteHandler.List)
		api.POST("/sites", siteHandler.Create)
		api.GET("/sites/:id", siteHandler.Get)
		api.PATCH("/sites/:id", siteHandler.Update)
		api.DELETE("/sites/:id", siteHandler.Delete)
		api.POST("/sites/:id/check", siteHandler.Check)
		api.POST("/sites/reorder", siteHandler.Reorder)
		api.GET("/sites/export", exportHandler.Export)
		api.POST("/sites/import", exportHandler.Import)

		api.GET("/sites/:id/snapshots", siteHandler.GetSnapshots)
		api.GET("/sites/:id/latest-snapshot", siteHandler.GetLatestSnapshot)
		api.GET("/sites/:id/diffs", siteHandler.GetDiffs)

		api.GET("/email-config", emailHandler.Get)
		api.POST("/email-config", emailHandler.Upsert)
		api.POST("/email-config/test", emailHandler.Test)
		api.GET("/schedule-config", scheduleHandler.Get)
		api.POST("/schedule-config", scheduleHandler.Update)
		api.POST("/schedule-config/trigger", scheduleHandler.Trigger)

		api.GET("/categories", catHandler.List)
		api.POST("/categories", catHandler.Create)
		api.PATCH("/categories/:id", catHandler.Update)
		api.DELETE("/categories/:id", catHandler.Delete)
		api.POST("/categories/:id/check", catHandler.Check)

		api.GET("/exports/sites", exportHandler.Export)

		api.GET("/sites/:id/tokens", siteHandler.ListTokens)
		api.POST("/sites/:id/tokens", siteHandler.CreateToken)
		api.PUT("/sites/:id/tokens", siteHandler.UpdateToken)
		api.DELETE("/sites/:id/tokens/:tokenId", siteHandler.DeleteToken)
		api.POST("/sites/:id/tokens/:tokenId/key", siteHandler.GetTokenKey)
		api.GET("/sites/:id/groups", siteHandler.ListGroups)
		api.GET("/sites/:id/pricing", siteHandler.GetPricing)
		api.POST("/sites/:id/redeem", siteHandler.Redeem)
	}

	apiNoAuth := r.Group("/api")
	{
		apiNoAuth.POST("/auth/login", authHandler.Login)
	}

	r.NoRoute(func(c *gin.Context) {
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		path := c.Request.URL.Path
		if path[0] == '/' {
			path = path[1:]
		}
		f, err := distFS.Open(path)
		if err == nil {
			defer f.Close()
			stat, _ := f.Stat()
			if stat != nil && !stat.IsDir() {
				http.ServeContent(c.Writer, c.Request, path, stat.ModTime(), f)
				return
			}
		}
		// SPA fallback
		file, err := distFS.Open("index.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		defer file.Close()
		stat, _ := file.Stat()
		http.ServeContent(c.Writer, c.Request, "index.html", stat.ModTime(), file)
	})

	return r
}
