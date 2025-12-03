// server/service/ip_group.go
package service

import (
	"context"
	"errors"
	"strconv"

	"github.com/HUAHUAI23/RuiQi/pkg/model"
	"github.com/HUAHUAI23/RuiQi/server/config"
	"github.com/HUAHUAI23/RuiQi/server/dto"
	"github.com/HUAHUAI23/RuiQi/server/repository"
	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	SystemDefaultBlacklistName = "system_default_blacklist" // System default blacklist group name
)

var (
	ErrIPGroupNotFound    = errors.New("IP group does not exist")
	ErrIPGroupNameExists  = errors.New("the IP group name already exists")
	ErrSystemIPGroupNoMod = errors.New("the system default IP group is not allowed to be deleted")
)

// IPGroupService IP组服务接口
type IPGroupService interface {
	CreateIPGroup(ctx context.Context, req *dto.IPGroupCreateRequest) (*model.IPGroup, error)
	GetIPGroups(ctx context.Context, pageStr, sizeStr string) ([]model.IPGroup, int64, error)
	GetIPGroupByID(ctx context.Context, id bson.ObjectID) (*model.IPGroup, error)
	UpdateIPGroup(ctx context.Context, id bson.ObjectID, req *dto.IPGroupUpdateRequest) (*model.IPGroup, error)
	DeleteIPGroup(ctx context.Context, id bson.ObjectID) error
	AddIPToBlacklist(ctx context.Context, ip string) error
}

// IPGroupServiceImpl IP组服务实现
type IPGroupServiceImpl struct {
	ipGroupRepo repository.IPGroupRepository
	logger      zerolog.Logger
}

// NewIPGroupService 创建IP组服务
func NewIPGroupService(ipGroupRepo repository.IPGroupRepository) IPGroupService {
	logger := config.GetServiceLogger("ipgroup")
	return &IPGroupServiceImpl{
		ipGroupRepo: ipGroupRepo,
		logger:      logger,
	}
}

// CreateIPGroup 创建IP组
func (s *IPGroupServiceImpl) CreateIPGroup(ctx context.Context, req *dto.IPGroupCreateRequest) (*model.IPGroup, error) {
	// 检查IP组名称是否已存在
	if req.Name != "" {
		exists, err := s.ipGroupRepo.CheckIPGroupNameExists(ctx, req.Name, bson.NilObjectID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrIPGroupNameExists
		}
	}

	// 创建新IP组
	ipGroup := &model.IPGroup{
		Name:  req.Name,
		Items: req.Items,
	}

	// 保存IP组
	err := s.ipGroupRepo.CreateIPGroup(ctx, ipGroup)
	if err != nil {
		s.logger.Error().Err(err).Msg("创建IP组失败")
		return nil, err
	}

	s.logger.Info().Str("id", ipGroup.ID.Hex()).Str("name", ipGroup.Name).Msg("IP组创建成功")
	return ipGroup, nil
}

// GetIPGroups 获取IP组列表
func (s *IPGroupServiceImpl) GetIPGroups(ctx context.Context, pageStr, sizeStr string) ([]model.IPGroup, int64, error) {
	page, err := strconv.ParseInt(pageStr, 10, 64)
	if err != nil || page < 1 {
		page = 1
	}

	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil || size < 1 {
		size = 10
	}

	ipGroups, total, err := s.ipGroupRepo.GetIPGroups(ctx, page, size)
	if err != nil {
		s.logger.Error().Err(err).Msg("获取IP组列表失败")
		return nil, 0, err
	}

	return ipGroups, total, nil
}

// GetIPGroupByID 根据ID获取IP组
func (s *IPGroupServiceImpl) GetIPGroupByID(ctx context.Context, id bson.ObjectID) (*model.IPGroup, error) {
	ipGroup, err := s.ipGroupRepo.GetIPGroupByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrIPGroupNotFound) {
			return nil, ErrIPGroupNotFound
		}
		s.logger.Error().Err(err).Str("id", id.Hex()).Msg("获取IP组失败")
		return nil, err
	}

	return ipGroup, nil
}

// UpdateIPGroup 更新IP组
func (s *IPGroupServiceImpl) UpdateIPGroup(ctx context.Context, id bson.ObjectID, req *dto.IPGroupUpdateRequest) (*model.IPGroup, error) {
	// 获取现有IP组
	ipGroup, err := s.ipGroupRepo.GetIPGroupByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrIPGroupNotFound) {
			return nil, ErrIPGroupNotFound
		}
		return nil, err
	}

	// 检查是否是系统默认IP组
	if ipGroup.Name == SystemDefaultBlacklistName {
		s.logger.Warn().Str("id", id.Hex()).Msg("尝试修改系统默认IP组")
		// 如果是系统默认IP组，只允许更新Items，不允许更新Name
		if req.Name != "" && req.Name != SystemDefaultBlacklistName {
			return nil, ErrSystemIPGroupNoMod
		}
	}

	// 检查IP组名称是否已存在（如果要更新名称）
	if req.Name != "" && req.Name != ipGroup.Name {
		exists, err := s.ipGroupRepo.CheckIPGroupNameExists(ctx, req.Name, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrIPGroupNameExists
		}
		ipGroup.Name = req.Name
	}

	// 更新IP列表（只更新非空字段）
	if req.Items != nil {
		ipGroup.Items = req.Items
	}

	// 保存更新
	err = s.ipGroupRepo.UpdateIPGroup(ctx, ipGroup)
	if err != nil {
		s.logger.Error().Err(err).Str("id", id.Hex()).Msg("更新IP组失败")
		return nil, err
	}

	s.logger.Info().Str("id", id.Hex()).Str("name", ipGroup.Name).Msg("IP组更新成功")
	return ipGroup, nil
}

// DeleteIPGroup 删除IP组
func (s *IPGroupServiceImpl) DeleteIPGroup(ctx context.Context, id bson.ObjectID) error {
	// 检查IP组是否存在
	ipGroup, err := s.ipGroupRepo.GetIPGroupByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrIPGroupNotFound) {
			return ErrIPGroupNotFound
		}
		return err
	}

	// 检查是否是系统默认IP组
	if ipGroup.Name == SystemDefaultBlacklistName {
		s.logger.Warn().Str("id", id.Hex()).Msg("Try to delete the system default IP group")
		return ErrSystemIPGroupNoMod
	}

	// 删除IP组
	err = s.ipGroupRepo.DeleteIPGroup(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Str("id", id.Hex()).Msg("Failed to delete IP group")
		return err
	}

	s.logger.Info().Str("id", id.Hex()).Msg("The IP group was deleted successfully")
	return nil
}

// AddIPToBlacklist 添加IP到系统默认黑名单
func (s *IPGroupServiceImpl) AddIPToBlacklist(ctx context.Context, ip string) error {
	// 查找系统默认黑名单组
	ipGroup, err := s.ipGroupRepo.GetIPGroupByName(ctx, SystemDefaultBlacklistName)
	if err != nil {
		if errors.Is(err, repository.ErrIPGroupNotFound) {
			s.logger.Error().Msg("The system default blacklist group does not exist")
			return ErrIPGroupNotFound
		}
		s.logger.Error().Err(err).Msg("Failed to obtain the system default blacklist group")
		return err
	}

	// 检查IP是否已存在
	for _, existingIP := range ipGroup.Items {
		if existingIP == ip {
			s.logger.Info().Str("ip", ip).Msg("IP已存在于黑名单中")
			return nil // 如果已存在，直接返回成功
		}
	}

	// 添加IP到黑名单
	ipGroup.Items = append(ipGroup.Items, ip)

	// 更新IP组
	err = s.ipGroupRepo.UpdateIPGroup(ctx, ipGroup)
	if err != nil {
		s.logger.Error().Err(err).Str("ip", ip).Msg("添加IP到黑名单失败")
		return err
	}

	s.logger.Info().Str("ip", ip).Msg("IP成功添加到黑名单")
	return nil
}
