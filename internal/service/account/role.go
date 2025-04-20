package account

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"snowgo/internal/constants"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/repo"
	"snowgo/internal/dao/account"
	"snowgo/pkg/xcache"
	"snowgo/pkg/xlogger"
)

type RoleRepo interface {
	IsCodeExists(ctx context.Context, code string, roleId int32) (bool, error)
	CreateRole(ctx context.Context, role *model.Role) (*model.Role, error)
	UpdateRole(ctx context.Context, role *model.Role) (*model.Role, error)
	DeleteById(ctx context.Context, roleId int32) error
	GetRoleById(ctx context.Context, roleId int32) (*model.Role, error)
	GetRoleList(ctx context.Context, cond *account.RoleListCondition) ([]*model.Role, int64, error)
}

type RoleService struct {
	db      *repo.Repository
	roleDao RoleRepo
	cache   xcache.Cache
}

// NewRoleService 构造函数
func NewRoleService(db *repo.Repository, roleDao RoleRepo, cache xcache.Cache) *RoleService {
	return &RoleService{db: db, roleDao: roleDao, cache: cache}
}

// RoleParam 创建或更新角色输入
type RoleParam struct {
	ID          int32   `form:"id"`
	Name        string  `json:"name" binding:"required,max=128"`
	Code        string  `json:"code" binding:"required,max=64"`
	Description string  `json:"description"`
	Status      string  `json:"status" binding:"required,oneof=Active Disabled"`
	MenuIds     []int32 `json:"menu_ids" binding:"required"`
}

// RoleInfo 返回给前端的角色信息
type RoleInfo struct {
	ID          int32     `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RoleList 返回角色列表
type RoleList struct {
	List  []*RoleInfo `json:"list"`
	Total int64       `json:"total"`
}

// CreateRole 创建角色
func (s *RoleService) CreateRole(ctx context.Context, param *RoleParam) (int32, error) {
	// 校验 code 是否存在
	exists, err := s.roleDao.IsCodeExists(ctx, param.Code, 0)
	if err != nil {
		xlogger.Errorf("校验角色code异常: %v", err)
		return 0, errors.WithMessage(err, "校验角色编码失败")
	}
	if exists {
		return 0, errors.New("角色编码已存在")
	}
	// 默认启用状态
	if param.Status == "" {
		param.Status = constants.ActiveStatus
	}
	role := &model.Role{
		Name:        &param.Name,
		Code:        param.Code,
		Description: &param.Description,
		Status:      &param.Status,
	}
	rule, err := s.roleDao.CreateRole(ctx, role)
	if err != nil {
		xlogger.Errorf("角色创建失败: %v", err)
		return 0, errors.WithMessage(err, "角色创建失败")
	}
	return rule.ID, nil
}

// UpdateRole 更新角色信息
func (s *RoleService) UpdateRole(ctx context.Context, role *RoleParam) error {
	xlogger.Infof("更新角色: %+v", role)
	if role.ID <= 0 {
		return errors.New("角色ID无效")
	}
	// 校验 code 是否重复（排除当前 role.ID）
	isDuplicate, err := s.roleDao.IsCodeExists(ctx, role.Code, role.ID)
	if err != nil {
		return errors.WithMessage(err, "查询角色 code 是否存在异常")
	}
	if isDuplicate {
		return errors.New("角色编码已存在")
	}
	// 获取原始角色信息（可选）
	oldRole, err := s.roleDao.GetRoleById(ctx, role.ID)
	if err != nil {
		return errors.WithMessage(err, "角色不存在")
	}
	// 更新字段
	ruleObj, err := s.roleDao.UpdateRole(ctx, &model.Role{
		ID:          role.ID,
		Name:        &role.Name,
		Code:        role.Code,
		Description: &role.Description,
		Status:      &role.Status,
	})
	if err != nil {
		return errors.WithMessage(err, "更新角色失败")
	}
	xlogger.Infof("角色更新成功: old=%+v new=%+v", oldRole, ruleObj)
	return nil
}

// DeleteRole 删除角色
func (s *RoleService) DeleteRole(ctx context.Context, id int32) error {
	xlogger.Infof("删除角色: %d", id)
	if id <= 0 {
		return errors.New("角色ID无效")
	}
	err := s.roleDao.DeleteById(ctx, id)
	if err != nil {
		xlogger.Errorf("角色删除失败: %v", err)
		return errors.WithMessage(err, "角色删除失败")
	}
	return nil
}

// GetRoleById 获取角色详情
func (s *RoleService) GetRoleById(ctx context.Context, id int32) (*RoleInfo, error) {
	if id <= 0 {
		return nil, errors.New("角色ID无效")
	}
	r, err := s.roleDao.GetRoleById(ctx, id)
	if err != nil {
		xlogger.Errorf("获取角色失败: %v", err)
		return nil, errors.WithMessage(err, "获取角色失败")
	}
	return &RoleInfo{
		ID:          r.ID,
		Name:        *r.Name,
		Code:        r.Code,
		Description: *r.Description,
		Status:      *r.Status,
		CreatedAt:   *r.CreatedAt,
		UpdatedAt:   *r.UpdatedAt,
	}, nil
}

// ListRoles 获取角色列表
func (s *RoleService) ListRoles(ctx context.Context, cond *account.RoleListCondition) (*RoleList, error) {
	xlogger.Infof("获取角色列表: %+v", cond)
	list, total, err := s.roleDao.GetRoleList(ctx, cond)
	if err != nil {
		xlogger.Errorf("角色列表查询失败: %v", err)
		return nil, errors.WithMessage(err, "角色列表查询失败")
	}
	infos := make([]*RoleInfo, 0, len(list))
	for _, r := range list {
		infos = append(infos, &RoleInfo{
			ID:          r.ID,
			Name:        *r.Name,
			Code:        r.Code,
			Description: *r.Description,
			Status:      *r.Status,
			CreatedAt:   *r.CreatedAt,
			UpdatedAt:   *r.UpdatedAt,
		})
	}
	return &RoleList{List: infos, Total: total}, nil
}
