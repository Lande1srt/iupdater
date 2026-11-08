package routes

import (
	"net/http"
	"strings"

	"iupdater/handlers"
	"iupdater/middleware"
	"iupdater/web"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	r.Static("/uploads", "./data/uploads")

	// 公开版本查询（无需认证）
	r.GET("/u/getmeta/:appid", handlers.GetMeta)        // 最新版本元数据（重写原 /u/:appid）
	r.GET("/u/listmeta/:appid", handlers.ListMeta)      // 列出该 app 所有版本元数据
	r.GET("/u/getlink/:appid", handlers.GetLatestLink)  // 302 重定向到最新下载
	// 保留 /u/:appid 做兼容重定向（返回 301 → /u/getmeta/:appid）
	r.GET("/u/:appid", handlers.RedirectToGetMeta)

	api := r.Group("/api")
	{
		api.GET("/login/config", handlers.GetLoginConfig)
		api.POST("/login", handlers.Login)
		api.POST("/check-update", handlers.CheckUpdate)
		api.GET("/download/*filekey", handlers.ProxyDownload)

		// Agent 辅助更新助手 API（Bearer access_key 鉴权）
		api.POST("/agent/checkin", handlers.AgentCheckin)
		api.POST("/agent/report", handlers.AgentReport)

		admin := api.Group("/admin")
		// 统一鉴权：先试 X-API-Key / Bearer APIKey-xxx，再 fallback 旧 Bearer JWT
		admin.Use(middleware.ApiKeyMiddleware())
		{
			// —— Admin APIKey 自身管理 ——
			admin.GET("/api-keys", handlers.ListAPIKeys)
			admin.POST("/api-keys", handlers.CreateAPIKey)
			admin.PUT("/api-keys/:id", handlers.UpdateAPIKey)
			admin.DELETE("/api-keys/:id", handlers.DeleteAPIKey)
			admin.POST("/api-keys/:id/regenerate", handlers.RegenerateAPIKey) // 重新生成并返回明文（仅此一次）
			admin.POST("/api-keys/:id/toggle", handlers.ToggleAPIKey)          // enable/disable

			// —— 存储配置档案管理 ——
			admin.GET("/storage-profiles", handlers.GetStorageProfiles)
			admin.GET("/storage-profiles/:id", handlers.GetStorageProfile)
			admin.POST("/storage-profiles", handlers.CreateStorageProfile)
			admin.PUT("/storage-profiles/:id", handlers.UpdateStorageProfile)
			admin.DELETE("/storage-profiles/:id", handlers.DeleteStorageProfile)
			admin.POST("/storage-profiles/:id/check", handlers.CheckStorageProfileConnection)

			admin.GET("/apps", handlers.GetApps)
			admin.GET("/apps/:id", handlers.GetApp)
			admin.POST("/apps", handlers.CreateApp)
			admin.PUT("/apps/:id", handlers.UpdateApp)
			admin.DELETE("/apps/:id", handlers.DeleteApp)

			admin.GET("/versions", handlers.GetVersions)
			admin.GET("/versions/groups", handlers.GetVersionGroups)
			admin.GET("/versions/:id", handlers.GetVersion)
			admin.POST("/versions", handlers.CreateVersion)
			admin.POST("/versions/upload", handlers.UploadVersionFile)
			admin.POST("/versions/probe", handlers.ProbeFileInfo)
			admin.PUT("/versions/:id/status", handlers.UpdateVersionStatus)
			admin.DELETE("/versions/:id", handlers.DeleteVersion)

			admin.GET("/components", handlers.GetComponents)
			admin.GET("/components/:id", handlers.GetComponent)
			admin.POST("/components", handlers.CreateComponent)
			admin.POST("/components/upload", handlers.UploadComponentFile)
			admin.PUT("/components/:id/status", handlers.UpdateComponentStatus)
			admin.DELETE("/components/:id", handlers.DeleteComponent)

			admin.POST("/storage/check", handlers.CheckStorageConnection)

			// Agent 管理
			admin.POST("/agents", handlers.RegisterAgent)
			admin.GET("/agents", handlers.GetAgents)
			admin.GET("/agents/:id", handlers.GetAgent)
			admin.PUT("/agents/:id", handlers.UpdateAgent)
			admin.DELETE("/agents/:id", handlers.DeleteAgent)
			admin.POST("/agents/:id/control", handlers.AgentControl)
			admin.POST("/agents/:id/clear-error", handlers.ClearAgentError)
		}
	}

	setupStaticFiles(r)
}

func setupStaticFiles(r *gin.Engine) {
	distFS, err := web.GetDistFS()
	if err != nil {
		return
	}

	httpFS := http.FS(distFS)

	r.GET("/assets/*filepath", func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", "public, max-age=31536000")
		http.FileServer(httpFS).ServeHTTP(c.Writer, c.Request)
	})

	r.GET("/", func(c *gin.Context) {
		f, err := distFS.Open("index.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		defer f.Close()

		stat, _ := f.Stat()
		c.DataFromReader(http.StatusOK, stat.Size(), "text/html; charset=utf-8", f, nil)
	})

	r.GET("/favicon.ico", func(c *gin.Context) {
		f, err := distFS.Open("favicon.ico")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		defer f.Close()

		stat, _ := f.Stat()
		c.DataFromReader(http.StatusOK, stat.Size(), "image/x-icon", f, nil)
	})

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/uploads/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
			return
		}

		if strings.HasPrefix(path, "/assets/") {
			c.Status(http.StatusNotFound)
			return
		}

		f, err := distFS.Open("index.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		defer f.Close()

		stat, _ := f.Stat()
		c.DataFromReader(http.StatusOK, stat.Size(), "text/html; charset=utf-8", f, nil)
	})
}
