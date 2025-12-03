package router

import (
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/HUAHUAI23/RuiQi/server/config"
	"github.com/HUAHUAI23/RuiQi/server/public"
	"github.com/gin-gonic/gin"
)

// SetStaticFileRouter 设置静态文件路由
func SetStaticFileRouter(router *gin.Engine) {
	logger := config.Logger

	// Load HTML template
	// Проверяем, не установлены ли уже шаблоны
	if router.HTMLRender == nil {
		router.SetHTMLTemplate(template.Must(template.New("").ParseFS(public.Templates, "templates/*")))
	}

	// 检查是否禁用Web功能
	if config.Global.DisableWeb {
		logger.Info().Msg("Web功能已禁用")
		router.GET("/", func(ctx *gin.Context) {
			ctx.HTML(http.StatusOK, "index.tmpl", gin.H{
				"URL":               "https://github.com/HUAHUAI23/RuiQi",
				"INITIAL_COUNTDOWN": 15,
			})
		})
		return
	}

	// 根据WebPath配置选择文件系统
	if config.Global.WebPath == "" {
		// 使用嵌入的文件系统
		logger.Info().Msg("Use embedded front-end resources")
		err := initFSRouter(router, public.Public.(fs.ReadDirFS), ".")
		if err != nil {
			panic(err)
		}
		fs := http.FS(public.Public)
		router.NoRoute(newIndexNoRouteHandler(fs))
	} else {
		// 使用外部文件系统
		logger.Info().Msg("使用外部前端资源路径")
		absPath, err := filepath.Abs(config.Global.WebPath)
		if err != nil {
			panic(err)
		}
		logger.Info().Str("path", absPath).Msg("使用外部前端资源路径")
		err = initFSRouter(router, os.DirFS(absPath).(fs.ReadDirFS), ".")
		if err != nil {
			panic(err)
		}
		router.NoRoute(newDynamicNoRouteHandler(http.Dir(absPath)))
	}
}

// checkNoRouteNotFound 检查路径是否为API路由
func checkNoRouteNotFound(path string) bool {
	if strings.HasPrefix(path, "/api") ||
		strings.HasPrefix(path, "/health") {
		return true
	}
	return false
}

// newIndexNoRouteHandler 创建处理SPA路由的NoRoute处理器（嵌入文件系统）
func newIndexNoRouteHandler(fs http.FileSystem) func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		if checkNoRouteNotFound(ctx.Request.URL.Path) {
			ctx.String(http.StatusNotFound, "404 page not found")
			return
		}
		// 对于前端路由，使用目录路径来避免index.html重定向问题
		// 根据 https://github.com/gin-gonic/gin/issues/2654 的解决方案
		// 使用 "" 或 "/" 而不是 "index.html" 来避免重定向循环
		ctx.FileFromFS("", fs)
	}
}

// newDynamicNoRouteHandler 创建处理SPA路由的NoRoute处理器（外部文件系统）
func newDynamicNoRouteHandler(fs http.FileSystem) func(ctx *gin.Context) {
	fileServer := http.StripPrefix("/", http.FileServer(fs))
	return func(c *gin.Context) {
		if checkNoRouteNotFound(c.Request.URL.Path) {
			c.String(http.StatusNotFound, "404 page not found")
			return
		}

		// 尝试打开请求的文件
		f, err := fs.Open(c.Request.URL.Path)
		if err != nil {
			// 文件不存在，返回 index.html（SPA路由处理）
			c.FileFromFS("", fs)
			return
		}
		f.Close()

		// 文件存在，直接提供服务
		fileServer.ServeHTTP(c.Writer, c.Request)
	}
}

// staticFileFS 接口定义
type staticFileFS interface {
	StaticFileFS(relativePath string, filepath string, fs http.FileSystem) gin.IRoutes
}

// initFSRouter 递归遍历文件系统并注册所有静态文件
func initFSRouter(e staticFileFS, f fs.ReadDirFS, path string) error {
	dirs, err := f.ReadDir(path)
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		u, err := url.JoinPath(path, dir.Name())
		if err != nil {
			return err
		}

		if dir.IsDir() {
			// 递归处理子目录
			err = initFSRouter(e, f, u)
			if err != nil {
				return err
			}
		} else {
			// 注册单个文件，但跳过根目录的index.html以避免冲突
			routePath := "/" + u
			if path == "." {
				routePath = "/" + dir.Name()
				// 跳过根目录的index.html，由NoRoute处理器处理
				if dir.Name() == "index.html" {
					continue
				}
			}
			e.StaticFileFS(routePath, u, http.FS(f))
		}
	}
	return nil
}
