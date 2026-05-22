package account

import (
	"errors"
	"testing"

	"snowgo/internal/constant"
	e "snowgo/pkg/xerror"
)

func TestRoleServiceEarlyValidation(t *testing.T) {
	service := &RoleService{}

	if err := service.UpdateRole(testUserCtx(), &RoleParam{}); !errors.Is(err, ErrRoleIDInvalid) {
		t.Fatalf("UpdateRole expected ErrRoleIDInvalid, got %v", err)
	}
	if _, err := service.GetRoleById(testUserCtx(), 0); !errors.Is(err, ErrRoleIDInvalid) {
		t.Fatalf("GetRoleById expected ErrRoleIDInvalid, got %v", err)
	}

	err := service.DeleteRole(testUserCtx(), constant.SuperAdminRoleId)
	var bizErr *e.BizError
	if !errors.As(err, &bizErr) || bizErr.Code != e.SuperAdminRoleCannotDelete {
		t.Fatalf("DeleteRole super admin expected SuperAdminRoleCannotDelete, got %v", err)
	}
}
