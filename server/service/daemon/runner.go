package daemon

import (
	"context"
	"fmt"
	"sync"
	"time"

	mongodb "github.com/HUAHUAI23/RuiQi/pkg/database/mongo"
	"github.com/haproxytech/client-native/v6/models"

	"github.com/HUAHUAI23/RuiQi/server/config"
	"github.com/HUAHUAI23/RuiQi/server/model"
	"github.com/HUAHUAI23/RuiQi/server/repository"
	"github.com/HUAHUAI23/RuiQi/server/service/daemon/engine"
	"github.com/HUAHUAI23/RuiQi/server/service/daemon/haproxy"
	"github.com/rs/zerolog"
)

type ServiceState int

const (
	ServiceStopped ServiceState = iota
	ServiceRunning
	ServiceError
)

type ServiceRunner interface {
	StartServices() error
	StopServices() error
	ForceStop()
	Restart() error
	HotReload() error
	GetState() ServiceState
	GetStats() (models.NativeStats, error)
}

// ServiceRunner Отвечает за управление и координацию всех услуг бэк-офиса
type ServiceRunnerImpl struct {
	haproxyService haproxy.HAProxyService
	engineService  engine.EngineService
	ctx            context.Context
	cancel         context.CancelFunc
	logger         *zerolog.Logger
	errChan        chan error
	haproxyDone    chan struct{} // Notify that the HAProxy service has stopped
	engineDone     chan struct{} // Notify Engine that the service has stopped
	state          ServiceState
}

// 单例模式实现
var (
	instance ServiceRunner
	once     sync.Once
	initErr  error
)

// GetRunnerService 获取ServiceRunner的单例实例
func GetRunnerService() (ServiceRunner, error) {
	once.Do(func() {
		instance, initErr = newServiceRunner()
	})
	return instance, initErr
}

// NewServiceRunner 创建一个新的服务运行器
func newServiceRunner() (ServiceRunner, error) {
	logger := config.GetLogger().With().Str("component", "runner").Logger()

	haproxyService, err := haproxy.NewHAProxyService("", "", nil)
	if err != nil {
		logger.Error().Err(err).Msg("初始化 HAProxy 服务失败")
		return nil, fmt.Errorf("初始化 HAProxy 服务失败: %w", err)
	}

	engineLogger := logger.With().Str("component", "engine").Logger()
	// 创建 Engine 服务
	engineService, err := engine.NewEngineService(engineLogger, config.Global.DBConfig.URI)
	if err != nil {
		logger.Error().Err(err).Msg("初始化 Engine 服务失败")
		return nil, fmt.Errorf("初始化 Engine 服务失败: %w", err)
	}

	return &ServiceRunnerImpl{
		haproxyService: haproxyService,
		engineService:  engineService,
		logger:         &logger,
		state:          ServiceStopped,
	}, nil
}

// StartServices 启动所有服务
func (r *ServiceRunnerImpl) StartServices() error {
	// 检查服务是否已经在运行
	if r.state == ServiceRunning {
		return fmt.Errorf("The service is already running")
	}

	// 创建新的上下文和取消函数
	r.ctx, r.cancel = context.WithCancel(context.Background())

	// 创建新的通道
	r.errChan = make(chan error, 10)
	r.haproxyDone = make(chan struct{})
	r.engineDone = make(chan struct{})

	// 启动HAProxy服务
	go func() {
		defer close(r.haproxyDone) // 服务停止时关闭通道

		client, err := mongodb.Connect(config.Global.DBConfig.URI)
		if err != nil {
			r.logger.Error().Err(err).Msg("runnner start services failed to connect to database")
			r.errChan <- err
			return
		}

		// 获取数据库
		db := client.Database(config.Global.DBConfig.Database)

		var site model.Site
		siteList, err := repository.GetAllSites(r.ctx, db.Collection(site.GetCollectionName()))
		if err != nil {
			r.logger.Error().Err(err).Msg("Failed to obtain the site list")
			r.errChan <- err
			return
		}

		r.logger.Info().Msg("Start the HAProxy service...")

		if err = r.haproxyService.RemoveConfig(); err != nil {
			r.logger.Error().Err(err).Msg("Failed to delete HAProxy configuration")
			r.errChan <- err
			return
		}

		if err = r.haproxyService.InitSpoeConfig(); err != nil {
			r.logger.Error().Err(err).Msg("Initialize HAProxy SPOE Configuration failed")
			r.errChan <- err
			return
		}

		if err = r.haproxyService.InitHAProxyConfig(); err != nil {
			r.logger.Error().Err(err).Msg("初始化HAProxy配置失败")
			r.errChan <- err
			return
		}

		if err = r.haproxyService.AddCorazaBackend(); err != nil {
			r.logger.Error().Err(err).Msg("添加Coraza后端失败")
			r.errChan <- err
			return
		}

		if err = r.haproxyService.CreateHAProxyCrtStore(); err != nil {
			r.logger.Error().Err(err).Msg("创建HAProxy证书存储失败")
			r.errChan <- err
			return
		}

		for i, site := range siteList {
			if err := r.haproxyService.AddSiteConfig(site); err != nil {
				r.logger.Error().Err(err).Msgf("添加站点配置失败 %d", i)
				// 继续处理其他站点，不返回错误
			}
		}

		if err := r.haproxyService.Start(); err != nil {
			r.logger.Error().Err(err).Msg("HAProxy服务启动失败")
			r.errChan <- err
			return
		}

		// 等待停止信号
		<-r.ctx.Done()
		r.logger.Info().Msg("Received a stop signal, stop the HAProxy service")
		if err := r.haproxyService.Stop(); err != nil {
			r.logger.Error().Err(err).Msg("Failed to stop HAProxy service")
			r.errChan <- err
		}
	}()

	// Start the Engine service
	go func() {
		defer close(r.engineDone) // Close the channel when the service stops

		r.logger.Info().Msg("Start the Engine service...")
		if err := r.engineService.Start(); err != nil {
			r.logger.Error().Err(err).Msg("Engine service failed to start")
			r.errChan <- err
			return
		}

		// Waiting for the stop signal
		<-r.ctx.Done()
		r.logger.Info().Msg("Received a stop signal, stop the Engine service")
		if err := r.engineService.Stop(); err != nil {
			r.logger.Error().Err(err).Msg("Failed to stop the Engine service")
			r.errChan <- err
		}
	}()

	// Listen to the error channel, and return the first error if an error occurs
	select {
	case err := <-r.errChan:
		r.logger.Error().Err(err).Msg("An error occurred during service startup")
		r.cancel() // Cancel the context and notify all services to stop
		r.state = ServiceError
		return err
	case <-time.After(2 * time.Second): // Give the service some startup time
		r.logger.Info().Msg("All services have been started")
		r.state = ServiceRunning
		return nil
	}
}

// StopServices Stop all services
func (r *ServiceRunnerImpl) StopServices() error {
	// Check if the service is running
	if r.state != ServiceRunning {
		return fmt.Errorf("The service is not running")
	}

	r.logger.Info().Msg("Start and stop all services...")

	// 1. First cancel the context and notify all actions that use the context
	if r.cancel != nil {
		r.cancel()
	}

	// 2. Waiting for a notification signal that the service has stopped
	timeoutDuration := 15 * time.Second

	// Monitor HAProxy service stop
	haproxyOk := make(chan struct{})
	go func() {
		select {
		case <-r.haproxyDone:
			r.logger.Info().Msg("HAProxy service has stopped normally")
		case <-time.After(timeoutDuration):
			r.logger.Warn().Msg("HAProxy service stopped timeout")
			// Try to force stop
			if err := r.haproxyService.Stop(); err != nil {
				r.logger.Error().Err(err).Msg("Failed to force stop the HAProxy service")
			}
		}
		close(haproxyOk)
	}()

	// Monitor Engine service stop
	engineOk := make(chan struct{})
	go func() {
		select {
		case <-r.engineDone:
			r.logger.Info().Msg("Engine service has stopped normally")
		case <-time.After(timeoutDuration):
			r.logger.Warn().Msg("Engine service stop timeout")
			// 尝试强制停止
			if err := r.engineService.Stop(); err != nil {
				r.logger.Error().Err(err).Msg("Failed to forcibly stop the Engine service")
			}
		}
		close(engineOk)
	}()

	// 等待所有服务停止
	<-haproxyOk
	<-engineOk

	var stopErr error
	select {
	case err := <-r.errChan:
		r.logger.Error().Err(err).Msg("服务停止过程中出现错误")
		stopErr = err
	default:
		// 没有错误
	}

	// 进行HAProxy服务重置
	if err := r.haproxyService.Reset(); err != nil {
		r.logger.Error().Err(err).Msg("重置HAProxy服务失败")
		if stopErr == nil {
			stopErr = err
		}
	}

	// 清理资源
	r.ctx = nil
	r.cancel = nil
	r.errChan = nil
	r.haproxyDone = nil
	r.engineDone = nil

	// 更新状态
	r.state = ServiceStopped

	if stopErr != nil {
		return stopErr
	}

	r.logger.Info().Msg("All services have been successfully stopped")

	return nil
}

// ForceStop 强制停止所有服务
func (r *ServiceRunnerImpl) ForceStop() {
	r.logger.Info().Msg("强制停止所有服务...")

	// 1. 首先取消上下文，通知所有使用该上下文的操作
	if r.cancel != nil {
		r.cancel()
	}

	// 2. 直接停止各服务
	if r.haproxyService != nil {
		r.logger.Info().Msg("正在强制停止 HAProxy 服务...")
		if err := r.haproxyService.Stop(); err != nil {
			r.logger.Error().Err(err).Msg("强制停止 HAProxy 服务出错")
		}
	}

	// 进行HAProxy服务重置
	if err := r.haproxyService.Reset(); err != nil {
		r.logger.Error().Err(err).Msg("重置HAProxy服务失败")
	}

	if r.engineService != nil {
		r.logger.Info().Msg("正在强制停止 Engine 服务...")
		if err := r.engineService.Stop(); err != nil {
			r.logger.Error().Err(err).Msg("强制停止 Engine 服务出错")
		}
	}

	// 清理资源
	r.ctx = nil
	r.cancel = nil
	r.errChan = nil
	r.haproxyDone = nil
	r.engineDone = nil

	// 更新状态
	r.state = ServiceStopped

	r.logger.Info().Msg("所有服务已强制停止")
}

// Restart 重启所有服务
func (r *ServiceRunnerImpl) Restart() error {
	// 停止服务
	if r.state == ServiceRunning {
		if err := r.StopServices(); err != nil {
			r.logger.Error().Err(err).Msg("重启时停止服务失败")
			return err
		}
	}

	// 启动服务
	return r.StartServices()
}

func (r *ServiceRunnerImpl) HotReload() error {
	// 检查服务是否正在运行
	if r.state != ServiceRunning {
		return fmt.Errorf("服务未在运行中，无法热重载")
	}
	r.logger.Info().Msg("开始热重载...")

	// // reload haproxy config

	// 进行HAProxy服务重置
	if err := r.haproxyService.Reset(); err != nil {
		r.logger.Error().Err(err).Msg("重置HAProxy服务失败")
		return err
	}

	client, err := mongodb.Connect(config.Global.DBConfig.URI)
	if err != nil {
		r.logger.Error().Err(err).Msg("hot reload failed to connect to database")
		return err
	}

	// 获取数据库
	db := client.Database(config.Global.DBConfig.Database)

	var site model.Site
	siteList, err := repository.GetAllSites(r.ctx, db.Collection(site.GetCollectionName()))
	if err != nil {
		r.logger.Error().Err(err).Msg("热加载获取站点列表失败")
		return err
	}

	r.logger.Info().Msg("开始热加载HAProxy配置...")
	if err = r.haproxyService.HotReloadRemoveConfig(); err != nil {
		r.logger.Error().Err(err).Msg("热加载删除HAProxy配置失败")
		return err
	}

	if err = r.haproxyService.InitSpoeConfig(); err != nil {
		r.logger.Error().Err(err).Msg("初始化HAProxy SPOE配置失败")
		return err
	}

	if err = r.haproxyService.InitHAProxyConfig(); err != nil {
		r.logger.Error().Err(err).Msg("初始化HAProxy配置失败")
		return err
	}

	if err = r.haproxyService.AddCorazaBackend(); err != nil {
		r.logger.Error().Err(err).Msg("添加Coraza后端失败")
		return err
	}

	if err = r.haproxyService.CreateHAProxyCrtStore(); err != nil {
		r.logger.Error().Err(err).Msg("创建HAProxy证书存储失败")
		return err
	}

	for i, site := range siteList {
		if err := r.haproxyService.AddSiteConfig(site); err != nil {
			r.logger.Error().Err(err).Msgf("添加站点配置失败 %d", i)
			// 继续处理其他站点，不返回错误
		}
	}

	if err := r.haproxyService.Reload(); err != nil {
		r.logger.Error().Err(err).Msg("热加载HAProxy配置失败")
		return err
	}

	// reload engine config

	if err := r.engineService.Reload(); err != nil {
		r.logger.Error().Err(err).Msg("热加载Engine配置失败")
		return err
	}

	r.logger.Info().Msg("热重载成功")

	return nil
}

// GetState 获取当前服务状态
func (r *ServiceRunnerImpl) GetState() ServiceState {
	return r.state
}

// GetStats 获取HAProxy的统计信息
func (r *ServiceRunnerImpl) GetStats() (models.NativeStats, error) {
	if r.haproxyService == nil {
		return models.NativeStats{}, fmt.Errorf("haproxy service not initialized")
	}
	return r.haproxyService.GetStats()
}
