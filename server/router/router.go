package router

import (
	"github.com/HUAHUAI23/RuiQi/server/controller"
	"github.com/HUAHUAI23/RuiQi/server/middleware"
	"github.com/HUAHUAI23/RuiQi/server/model"
	"github.com/HUAHUAI23/RuiQi/server/repository"
	"github.com/HUAHUAI23/RuiQi/server/service"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Setup configures all the routes for the application
func Setup(route *gin.Engine, db *mongo.Database) {
	// ===== Front-end static resource hosting=====
	SetStaticFileRouter(route)

	// Basic middleware
	route.Use(middleware.RequestID())
	route.Use(middleware.Logger())
	route.Use(middleware.Cors())
	route.Use(gin.CustomRecovery(middleware.CustomErrorHandler))

	// Create a warehouse
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	siteRepo := repository.NewSiteRepository(db)
	wafLogRepo := repository.NewWAFLogRepository(db)
	certRepo := repository.NewCertificateRepository(db)
	configRepo := repository.NewConfigRepository(db)
	ipGroupRepo := repository.NewIPGroupRepository(db)
	ruleRepo := repository.NewMicroRuleRepository(db)
	blockedIPRepo := repository.NewBlockedIPRepository(db)

	// Create service
	authService := service.NewAuthService(userRepo, roleRepo)
	siteService := service.NewSiteService(siteRepo)
	wafLogService := service.NewWAFLogService(wafLogRepo)
	certService := service.NewCertificateService(certRepo)
	runnerService, _ := service.NewRunnerService()
	configService := service.NewConfigService(configRepo)
	ipGroupService := service.NewIPGroupService(ipGroupRepo)
	ruleService := service.NewMicroRuleService(ruleRepo)
	statsService := service.NewStatsService(wafLogRepo)
	blockedIPService := service.NewBlockedIPService(blockedIPRepo)

	// Create a controller
	authController := controller.NewAuthController(authService)
	siteController := controller.NewSiteController(siteService)
	wafLogController := controller.NewWAFLogController(wafLogService)
	certController := controller.NewCertificateController(certService)
	runnerController := controller.NewRunnerController(runnerService)
	configController := controller.NewConfigController(configService)
	ipGroupController := controller.NewIPGroupController(ipGroupService)
	ruleController := controller.NewMicroRuleController(ruleService)
	statsController := controller.NewStatsController(runnerService, statsService)
	blockedIPController := controller.NewBlockedIPController(blockedIPService)

	// Add the warehouse to the context for middleware use
	route.Use(func(c *gin.Context) {
		c.Set("userRepo", userRepo)
		c.Set("roleRepo", roleRepo)
		c.Next()
	})

	// Health check endpoint
	route.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API v1 routing
	api := route.Group("/api/v1")

	// Authentication related routes-no permission check required
	auth := api.Group("/auth")
	{
		auth.POST("/login", authController.Login)

		// 需要认证的路由
		authRequired := auth.Group("")
		authRequired.Use(middleware.JWTAuth())
		{
			// 密码重置接口 - 任何已认证用户都可访问
			authRequired.POST("/reset-password", authController.ResetPassword)

			// 需要密码重置检查的路由
			passwordChecked := authRequired.Group("")
			passwordChecked.Use(middleware.PasswordResetRequired())
			{
				// 获取个人信息 - 任何已认证用户都可访问
				passwordChecked.GET("/me", authController.GetUserInfo)
			}
		}
	}

	// 需要认证和密码重置检查的API路由
	authenticated := api.Group("")
	authenticated.Use(middleware.JWTAuth())
	authenticated.Use(middleware.PasswordResetRequired())

	// 用户管理模块
	userRoutes := authenticated.Group("/users")
	{
		// 创建用户 - 需要user:create权限
		userRoutes.POST("", middleware.HasPermission(model.PermUserCreate), authController.CreateUser)
		// 获取用户列表 - 需要user:read权限
		userRoutes.GET("", middleware.HasPermission(model.PermUserRead), authController.GetUsers)
		// 更新用户 - 需要user:update权限
		userRoutes.PUT("/:id", middleware.HasPermission(model.PermUserUpdate), authController.UpdateUser)
		// 删除用户 - 需要user:delete权限
		userRoutes.DELETE("/:id", middleware.HasPermission(model.PermUserDelete), authController.DeleteUser)
	}

	// 站点管理模块
	siteRoutes := authenticated.Group("/site")
	{
		// 创建站点 - 需要site:create权限
		siteRoutes.POST("", middleware.HasPermission(model.PermSiteCreate), siteController.CreateSite)
		// 获取站点列表 - 需要site:read权限
		siteRoutes.GET("", middleware.HasPermission(model.PermSiteRead), siteController.GetSites)
		// 获取单个站点 - 需要site:read权限
		siteRoutes.GET("/:id", middleware.HasPermission(model.PermSiteRead), siteController.GetSiteByID)
		// 更新站点 - 需要site:update权限
		siteRoutes.PUT("/:id", middleware.HasPermission(model.PermSiteUpdate), siteController.UpdateSite)
		// 删除站点 - 需要site:delete权限
		siteRoutes.DELETE("/:id", middleware.HasPermission(model.PermSiteDelete), siteController.DeleteSite)
	}

	// 证书管理路由
	certRoutes := authenticated.Group("/certificate")
	{
		certRoutes.POST("", middleware.HasPermission(model.PermCertCreate), certController.CreateCertificate)
		certRoutes.GET("", middleware.HasPermission(model.PermCertRead), certController.GetCertificates)
		certRoutes.GET("/:id", middleware.HasPermission(model.PermCertRead), certController.GetCertificateByID)
		certRoutes.PUT("/:id", middleware.HasPermission(model.PermCertUpdate), certController.UpdateCertificate)
		certRoutes.DELETE("/:id", middleware.HasPermission(model.PermCertDelete), certController.DeleteCertificate)
	}

	// IP组管理路由
	ipGroupRoutes := authenticated.Group("/ip-groups")
	{
		ipGroupRoutes.POST("", middleware.HasPermission(model.PermConfigUpdate), ipGroupController.CreateIPGroup)
		ipGroupRoutes.GET("", middleware.HasPermission(model.PermConfigRead), ipGroupController.GetIPGroups)
		ipGroupRoutes.GET("/:id", middleware.HasPermission(model.PermConfigRead), ipGroupController.GetIPGroupByID)
		ipGroupRoutes.PUT("/:id", middleware.HasPermission(model.PermConfigUpdate), ipGroupController.UpdateIPGroup)
		ipGroupRoutes.DELETE("/:id", middleware.HasPermission(model.PermConfigUpdate), ipGroupController.DeleteIPGroup)
		// 添加IP到系统默认黑名单
		ipGroupRoutes.POST("/blacklist/add", middleware.HasPermission(model.PermConfigUpdate), ipGroupController.AddIPToBlacklist)
	}

	// rule 管理路由
	ruleRoutes := authenticated.Group("/micro-rules")
	{
		ruleRoutes.POST("", middleware.HasPermission(model.PermConfigUpdate), ruleController.CreateMicroRule)
		ruleRoutes.GET("", middleware.HasPermission(model.PermConfigRead), ruleController.GetMicroRules)
		ruleRoutes.GET("/:id", middleware.HasPermission(model.PermConfigRead), ruleController.GetMicroRuleByID)
		ruleRoutes.PUT("/:id", middleware.HasPermission(model.PermConfigUpdate), ruleController.UpdateMicroRule)
		ruleRoutes.DELETE("/:id", middleware.HasPermission(model.PermConfigUpdate), ruleController.DeleteMicroRule)
	}

	// 日志
	wafLogRoutes := authenticated.Group("/log")
	{
		// 获取攻击事件 - 需要logs:read权限
		wafLogRoutes.GET("/event", middleware.HasPermission(model.PermWAFLogRead), wafLogController.GetAttackEvents)
		// 获取攻击日志 - 需要logs:read权限
		wafLogRoutes.GET("", middleware.HasPermission(model.PermWAFLogRead), wafLogController.GetAttackLogs)
	}

	// 统计信息路由
	statsRoutes := authenticated.Group("/stats")
	{
		// 获取概览统计 - 需要config:read权限
		statsRoutes.GET("/overview", middleware.HasPermission(model.PermWAFLogRead), statsController.GetOverviewStats)
		// 获取实时QPS - 需要config:read权限
		statsRoutes.GET("/realtime-qps", middleware.HasPermission(model.PermWAFLogRead), statsController.GetRealtimeQPS)
		// 获取时间序列数据 - 需要config:read权限
		statsRoutes.GET("/time-series", middleware.HasPermission(model.PermWAFLogRead), statsController.GetTimeSeriesData)
		// 获取组合时间序列数据 - 需要config:read权限
		statsRoutes.GET("/combined-time-series", middleware.HasPermission(model.PermWAFLogRead), statsController.GetCombinedTimeSeriesData)
		// 获取流量时间序列数据 - 需要config:read权限
		statsRoutes.GET("/traffic-time-series", middleware.HasPermission(model.PermWAFLogRead), statsController.GetTrafficTimeSeriesData)
	}

	// 配置管理模块
	runnerRoutes := authenticated.Group("/runner")
	{
		// 获取配置 - 需要config:read权限
		runnerRoutes.GET("/status", middleware.HasPermission(model.PermConfigRead), runnerController.GetStatus)
		// 更新配置 - 需要config:update权限
		runnerRoutes.POST("/control", middleware.HasPermission(model.PermConfigUpdate), runnerController.Control)
	}
	configRoutes := authenticated.Group("/config")
	{
		// 获取配置 - 需要config:read权限
		configRoutes.GET("", middleware.HasPermission(model.PermConfigRead), configController.GetConfig)
		// 更新配置 - 需要config:update权限
		configRoutes.PATCH("", middleware.HasPermission(model.PermConfigUpdate), configController.PatchConfig)
	}

	// 封禁IP管理模块
	blockedIPRoutes := authenticated.Group("/blocked-ips")
	{
		blockedIPRoutes.GET("", middleware.HasPermission(model.PermConfigRead), blockedIPController.GetBlockedIPs)
		blockedIPRoutes.GET("/stats", middleware.HasPermission(model.PermConfigRead), blockedIPController.GetBlockedIPStats)
		blockedIPRoutes.DELETE("/cleanup", middleware.HasPermission(model.PermConfigUpdate), blockedIPController.CleanupExpiredBlockedIPs)
	}

	// 审计日志模块
	auditRoutes := authenticated.Group("/audit")
	{
		// 获取审计日志 - 需要audit:read权限
		auditRoutes.GET("", middleware.HasPermission(model.PermAuditRead), nil)
	}

	// 系统管理模块
	systemRoutes := authenticated.Group("/system")
	{
		// 获取系统状态 - 需要system:status权限
		systemRoutes.GET("/status", middleware.HasPermission(model.PermSystemStatus), nil)
		// 重启系统 - 需要system:restart权限
		systemRoutes.POST("/restart", middleware.HasPermission(model.PermSystemRestart), nil)
	}
}
