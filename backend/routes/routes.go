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
	r.GET("/u/:appid", handlers.GetLatestVersion)
	r.GET("/u/getlink/:appid", handlers.GetLatestLink)

	api := r.Group("/api")
	{
		api.GET("/login/config", handlers.GetLoginConfig)
		api.POST("/login", handlers.Login)
		api.POST("/check-update", handlers.CheckUpdate)
		api.GET("/download/*filekey", handlers.ProxyDownload)

		admin := api.Group("/admin")
		admin.Use(middleware.AuthMiddleware())
		{
			// 存储配置档案管理（侧边栏存储管理）
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
