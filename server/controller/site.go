package controller

import (
	"errors"
	"net/http"

	"github.com/HUAHUAI23/RuiQi/server/config"
	"github.com/HUAHUAI23/RuiQi/server/dto"
	"github.com/HUAHUAI23/RuiQi/server/model"
	"github.com/HUAHUAI23/RuiQi/server/repository"
	"github.com/HUAHUAI23/RuiQi/server/service"
	"github.com/HUAHUAI23/RuiQi/server/utils/response"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// SiteController 站点控制器接口
type SiteController interface {
	CreateSite(ctx *gin.Context)
	GetSites(ctx *gin.Context)
	GetSiteByID(ctx *gin.Context)
	UpdateSite(ctx *gin.Context)
	DeleteSite(ctx *gin.Context)
}

// SiteControllerImpl 站点控制器实现
type SiteControllerImpl struct {
	siteService service.SiteService
	logger      zerolog.Logger
}

// NewSiteController 创建站点控制器
func NewSiteController(siteService service.SiteService) SiteController {
	logger := config.GetControllerLogger("site")
	return &SiteControllerImpl{
		siteService: siteService,
		logger:      logger,
	}
}

// CreateSite 创建站点
//
//	@Summary		创建新站点
//	@Description	创建一个新的站点配置
//	@Tags			站点管理
//	@Accept			json
//	@Produce		json
//	@Param			site	body	dto.CreateSiteRequest	true	"站点信息"
//	@Security		BearerAuth
//	@Success		200	{object}	model.SuccessResponse{data=dto.SiteResponse}	"站点创建成功"
//	@Failure		400	{object}	model.ErrResponse								"请求参数错误"
//	@Failure		401	{object}	model.ErrResponseDontShowError					"未授权访问"
//	@Failure		403	{object}	model.ErrResponseDontShowError					"禁止访问"
//	@Failure		409	{object}	model.ErrResponseDontShowError					"域名和端口组合已存在"
//	@Failure		500	{object}	model.ErrResponseDontShowError					"服务器内部错误"
//	@Router			/api/v1/site [post]
func (c *SiteControllerImpl) CreateSite(ctx *gin.Context) {
	var req dto.CreateSiteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warn().Err(err).Msg("请求参数绑定失败")
		response.BadRequest(ctx, err, true)
		return
	}

	c.logger.Info().Str("name", req.Name).Str("domain", req.Domain).Msg("创建站点请求")

	site, err := c.siteService.CreateSite(ctx, &req)
	if err != nil {
		if errors.Is(err, repository.ErrDomainPortExists) {
			response.Error(ctx, model.NewAPIError(http.StatusConflict, "域名和端口组合已存在", err), false)
			return
		}
		c.logger.Error().Err(err).Msg("创建站点失败")
		response.InternalServerError(ctx, err, false)
		return
	}

	c.logger.Info().Str("id", site.ID.Hex()).Str("name", site.Name).Msg("站点创建成功")
	response.Success(ctx, "站点创建成功", site)
}

// GetSites Get a list of sites
//
//	@Summary		获取站点列表
//	@Description	获取所有站点配置列表
//	@Tags			站点管理
//	@Produce		json
//	@Param			page	query	int	false	"页码"	default(1)
//	@Param			size	query	int	false	"每页数量"	default(10)
//	@Security		BearerAuth
//	@Success		200	{object}	model.SuccessResponse{data=dto.SiteListResponse}	"获取站点列表成功"
//	@Failure		401	{object}	model.ErrResponseDontShowError						"未授权访问"
//	@Failure		500	{object}	model.ErrResponseDontShowError						"服务器内部错误"
//	@Router			/api/v1/site [get]
func (c *SiteControllerImpl) GetSites(ctx *gin.Context) {
	page := ctx.DefaultQuery("page", "1")
	size := ctx.DefaultQuery("size", "10")

	c.logger.Info().Str("page", page).Str("size", size).Msg("获取站点列表请求")
	sites, total, err := c.siteService.GetSites(ctx, page, size)
	if err != nil {
		c.logger.Error().Err(err).Msg("获取站点列表失败")
		response.InternalServerError(ctx, err, false)
		return
	}

	if sites == nil {
		sites = []model.Site{}
	}

	c.logger.Info().Int64("total", total).Msg("获取站点列表成功")
	response.Success(ctx, "获取站点列表成功", gin.H{
		"total": total,
		"items": sites,
	})
}

// GetSiteByID 获取单个站点
//
//	@Summary		获取单个站点
//	@Description	根据ID获取站点详情
//	@Tags			站点管理
//	@Produce		json
//	@Param			id	path	string	true	"站点ID"
//	@Security		BearerAuth
//	@Success		200	{object}	model.SuccessResponse{data=dto.SiteResponse}	"获取站点详情成功"
//	@Failure		401	{object}	model.ErrResponseDontShowError					"未授权访问"
//	@Failure		404	{object}	model.ErrResponseDontShowError					"站点不存在"
//	@Failure		500	{object}	model.ErrResponseDontShowError					"服务器内部错误"
//	@Router			/api/v1/site/{id} [get]
func (c *SiteControllerImpl) GetSiteByID(ctx *gin.Context) {
	id := ctx.Param("id")

	c.logger.Info().Str("id", id).Msg("获取站点详情请求")
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		c.logger.Error().Err(err).Str("id", id).Msg("无效的ID格式")
		response.BadRequest(ctx, err, true)
		return
	}
	site, err := c.siteService.GetSiteByID(ctx, objectID)
	if err != nil {
		if errors.Is(err, repository.ErrSiteNotFound) {
			response.Error(ctx, model.NewAPIError(http.StatusNotFound, "站点不存在", err), false)
			return
		}
		c.logger.Error().Err(err).Str("id", id).Msg("获取站点详情失败")
		response.InternalServerError(ctx, err, false)
		return
	}

	c.logger.Info().Str("id", id).Str("name", site.Name).Msg("获取站点详情成功")
	response.Success(ctx, "获取站点详情成功", site)
}

// UpdateSite 更新站点
//
//	@Summary		更新站点
//	@Description	更新指定站点的配置
//	@Tags			站点管理
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string					true	"站点ID"
//	@Param			site	body	dto.UpdateSiteRequest	true	"站点更新信息"
//	@Security		BearerAuth
//	@Success		200	{object}	model.SuccessResponse{data=dto.SiteResponse}	"站点更新成功"
//	@Failure		400	{object}	model.ErrResponse								"请求参数错误"
//	@Failure		401	{object}	model.ErrResponseDontShowError					"未授权访问"
//	@Failure		403	{object}	model.ErrResponseDontShowError					"禁止访问"
//	@Failure		404	{object}	model.ErrResponseDontShowError					"站点不存在"
//	@Failure		409	{object}	model.ErrResponseDontShowError					"域名和端口组合已被其他站点使用"
//	@Failure		500	{object}	model.ErrResponseDontShowError					"服务器内部错误"
//	@Router			/api/v1/site/{id} [put]
func (c *SiteControllerImpl) UpdateSite(ctx *gin.Context) {
	id := ctx.Param("id")
	var req dto.UpdateSiteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warn().Err(err).Str("id", id).Msg("请求参数绑定失败")
		response.BadRequest(ctx, err, true)
		return
	}

	c.logger.Info().Str("id", id).Msg("更新站点请求")
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		c.logger.Error().Err(err).Str("id", id).Msg("无效的ID格式")
		response.BadRequest(ctx, err, true)
		return
	}
	site, err := c.siteService.UpdateSite(ctx, objectID, &req)
	if err != nil {
		if errors.Is(err, repository.ErrSiteNotFound) {
			response.Error(ctx, model.NewAPIError(http.StatusNotFound, "站点不存在", err), false)
			return
		} else if errors.Is(err, repository.ErrDomainPortConflict) {
			response.Error(ctx, model.NewAPIError(http.StatusConflict, "域名和端口组合已被其他站点使用", err), false)
			return
		}
		c.logger.Error().Err(err).Str("id", id).Msg("更新站点失败")
		response.InternalServerError(ctx, err, false)
		return
	}

	c.logger.Info().Str("id", id).Str("name", site.Name).Msg("站点更新成功")
	response.Success(ctx, "站点更新成功", site)
}

// DeleteSite 删除站点
//
//	@Summary		删除站点
//	@Description	删除指定的站点配置
//	@Tags			站点管理
//	@Produce		json
//	@Param			id	path	string	true	"站点ID"
//	@Security		BearerAuth
//	@Success		200	{object}	model.SuccessResponseNoData		"站点删除成功"
//	@Failure		401	{object}	model.ErrResponseDontShowError	"未授权访问"
//	@Failure		403	{object}	model.ErrResponseDontShowError	"禁止访问"
//	@Failure		404	{object}	model.ErrResponseDontShowError	"站点不存在"
//	@Failure		500	{object}	model.ErrResponseDontShowError	"服务器内部错误"
//	@Router			/api/v1/site/{id} [delete]
func (c *SiteControllerImpl) DeleteSite(ctx *gin.Context) {
	id := ctx.Param("id")

	c.logger.Info().Str("id", id).Msg("删除站点请求")
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		c.logger.Error().Err(err).Str("id", id).Msg("无效的ID格式")
		response.BadRequest(ctx, err, true)
		return
	}
	err = c.siteService.DeleteSite(ctx, objectID)
	if err != nil {
		if errors.Is(err, repository.ErrSiteNotFound) {
			response.Error(ctx, model.NewAPIError(http.StatusNotFound, "站点不存在", err), false)
			return
		}
		c.logger.Error().Err(err).Str("id", id).Msg("删除站点失败")
		response.InternalServerError(ctx, err, false)
		return
	}

	c.logger.Info().Str("id", id).Msg("站点删除成功")
	response.Success(ctx, "站点删除成功", nil)
}
