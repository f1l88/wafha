package controller

import (
	"errors"

	"github.com/HUAHUAI23/RuiQi/server/config"
	"github.com/HUAHUAI23/RuiQi/server/dto"
	"github.com/HUAHUAI23/RuiQi/server/service"
	"github.com/HUAHUAI23/RuiQi/server/service/daemon"
	"github.com/HUAHUAI23/RuiQi/server/utils/response"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// RunnerController 运行器控制器接口
type RunnerController interface {
	GetStatus(ctx *gin.Context)
	Control(ctx *gin.Context)
}

// RunnerControllerImpl 运行器控制器实现
type RunnerControllerImpl struct {
	runnerService service.RunnerService
	logger        zerolog.Logger
}

// NewRunnerController 创建运行器控制器
func NewRunnerController(runnerService service.RunnerService) RunnerController {
	logger := config.GetControllerLogger("runner")
	return &RunnerControllerImpl{
		runnerService: runnerService,
		logger:        logger,
	}
}

// getStateString 将ServiceState转换为字符串
func getStateString(state daemon.ServiceState) string {
	switch state {
	case daemon.ServiceRunning:
		return "running"
	case daemon.ServiceStopped:
		return "stopped"
	case daemon.ServiceError:
		return "error"
	default:
		return "unknown"
	}
}

// isStateRunning 检查状态是否为运行中
func isStateRunning(state daemon.ServiceState) bool {
	return state == daemon.ServiceRunning
}

// getSuccessMessage 根据操作类型返回成功消息
func getSuccessMessage(action string) string {
	switch action {
	case "start":
		return "运行器已成功启动"
	case "stop":
		return "运行器已成功停止"
	case "restart":
		return "运行器已成功重启"
	case "force_stop":
		return "运行器已强制停止"
	case "reload":
		return "运行器配置已重新加载"
	default:
		return "操作成功"
	}
}

// toRunnerStatusResponse 将状态转换为响应对象
func toRunnerStatusResponse(state daemon.ServiceState) dto.RunnerStatusResponse {
	return dto.RunnerStatusResponse{
		State:     getStateString(state),
		IsRunning: isStateRunning(state),
	}
}

// buildControlResponse 构建控制响应
func buildControlResponse(action string, state daemon.ServiceState) dto.RunnerControlResponse {
	return dto.RunnerControlResponse{
		Success: true,
		Action:  action,
		Message: getSuccessMessage(action),
		State:   getStateString(state),
	}
}

// GetStatus 获取运行器状态
//
//	@Summary		获取后台运行器状态
//	@Description	获取WAF后台运行器的运行状态
//	@Tags			运行器管理
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	model.SuccessResponse{data=dto.RunnerStatusResponse}	"获取运行器状态成功"
//	@Failure		500	{object}	model.ErrResponseDontShowError							"服务器内部错误"
//	@Router			/api/runner/status [get]
func (c *RunnerControllerImpl) GetStatus(ctx *gin.Context) {
	// 获取运行器状态
	state, err := c.runnerService.GetStatus(ctx)
	if err != nil {
		c.logger.Error().Err(err).Msg("获取运行器状态失败")
		response.InternalServerError(ctx, err, false)
		return
	}

	// 构建响应
	resp := toRunnerStatusResponse(state)

	response.Success(ctx, "获取运行器状态成功", resp)
}

// Control 控制运行器
//
//	@Summary		控制后台运行器
//	@Description	执行启动、停止、重启、强制停止或热重载操作
//	@Tags			运行器管理
//	@Accept			json
//	@Produce		json
//	@Param			request	body	dto.RunnerControlRequest	true	"运行器控制请求"
//	@Security		BearerAuth
//	@Success		200	{object}	model.SuccessResponse{data=dto.RunnerControlResponse}	"操作成功"
//	@Failure		400	{object}	model.ErrResponse										"请求参数错误"
//	@Failure		500	{object}	model.ErrResponseDontShowError							"服务器内部错误"
//	@Router			/api/runner/control [post]
func (c *RunnerControllerImpl) Control(ctx *gin.Context) {
	var req dto.RunnerControlRequest

	// 绑定请求参数
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warn().Err(err).Msg("请求参数绑定失败")
		response.BadRequest(ctx, err, true)
		return
	}

	c.logger.Info().Str("action", req.Action).Msg("控制运行器请求")

	var err error

	// 根据操作类型执行相应的操作
	switch req.Action {
	case "start":
		err = c.runnerService.Start(ctx)
	case "stop":
		err = c.runnerService.Stop(ctx)
	case "restart":
		err = c.runnerService.Restart(ctx)
	case "force_stop":
		err = c.runnerService.ForceStop(ctx)
	case "reload":
		err = c.runnerService.Reload(ctx)
	default:
		c.logger.Warn().Str("action", req.Action).Msg("Unsupported operation types")
		response.BadRequest(ctx, errors.New("Unsupported operation types"), true)
		return
	}

	// 处理操作结果
	if err != nil {
		if errors.Is(err, service.ErrRunnerNotRunning) {
			c.logger.Warn().Str("action", req.Action).Msg("The runner is not running")
			response.BadRequest(ctx, err, true)
			return
		} else if errors.Is(err, service.ErrRunnerAlreadyRunning) {
			c.logger.Warn().Str("action", req.Action).Msg("The runner is already running")
			response.BadRequest(ctx, err, true)
			return
		}

		c.logger.Error().Err(err).Str("action", req.Action).Msg("Operator operation failed")
		response.InternalServerError(ctx, err, false)
		return
	}

	// 操作成功后获取最新状态
	state, _ := c.runnerService.GetStatus(ctx)

	// 构建响应
	resp := buildControlResponse(req.Action, state)

	c.logger.Info().Str("action", req.Action).Str("state", resp.State).Msg("The operator operated successfully")
	response.Success(ctx, "Successful operation", resp)
}
