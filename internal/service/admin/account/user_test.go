package account

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	"snowgo/internal/dal/repo"
	"snowgo/internal/service/admin/contract"
	"snowgo/pkg/xauth"
	"snowgo/pkg/xcryption"
	e "snowgo/pkg/xerror"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantCode e.Code
	}{
		{name: "empty", password: "", wantCode: e.PwdComplexityError},
		{name: "too short", password: "a1@", wantCode: e.PwdLengthError},
		{name: "too long", password: "a1@" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", wantCode: e.PwdLengthError},
		{name: "invalid char", password: "abc123中文", wantCode: e.PwdInvalidCharError},
		{name: "single class", password: "abcdef", wantCode: e.PwdComplexityError},
		{name: "letter and digit", password: "abc123", wantCode: nil},
		{name: "letter and symbol", password: "abcdef!", wantCode: nil},
		{name: "digit and symbol", password: "123456!", wantCode: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.password)
			if tt.wantCode == nil {
				if err != nil {
					t.Fatalf("expected password to be valid, got %v", err)
				}
				return
			}
			var bizErr *e.BizError
			if !errors.As(err, &bizErr) {
				t.Fatalf("expected BizError, got %v", err)
			}
			if bizErr.Code.GetErrCode() != tt.wantCode.GetErrCode() {
				t.Fatalf("expected code %v, got %v", tt.wantCode, bizErr.Code)
			}
		})
	}
}

func TestUserServiceEarlyValidation(t *testing.T) {
	service := &UserService{}

	if err := service.DeleteById(testUserCtx(), 0); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("DeleteById expected ErrUserNotFound, got %v", err)
	}
	if err := service.DeleteById(testUserCtx(), 1); !errors.Is(err, ErrDeleteSelf) {
		t.Fatalf("DeleteById self expected ErrDeleteSelf, got %v", err)
	}
	if err := service.ResetPwdById(testUserCtx(), 0, "abc123"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("ResetPwdById expected ErrUserNotFound, got %v", err)
	}
	err := service.ResetPwdById(testUserCtx(), 2, "abcdef")
	var bizErr *e.BizError
	if !errors.As(err, &bizErr) || bizErr.Code.GetErrCode() != e.PwdComplexityError.GetErrCode() {
		t.Fatalf("ResetPwdById weak password expected PwdComplexityError, got %v", err)
	}
}

func TestUserServiceAuthenticate(t *testing.T) {
	activeStatus := constant.UserStatusActive
	disabledStatus := constant.UserStatusDisabled
	passwordHash, err := xcryption.HashPassword("abc123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	tests := []struct {
		name       string
		repo       *fakeUserRepo
		password   string
		wantErr    error
		wantStatus int8
	}{
		{
			name:     "empty username",
			repo:     &fakeUserRepo{},
			password: "abc123",
			wantErr:  ErrUserNotFound,
		},
		{
			name:       "user not found returns auth error",
			repo:       &fakeUserRepo{userByUsernameErr: gorm.ErrRecordNotFound},
			password:   "abc123",
			wantErr:    ErrAuth,
			wantStatus: 0,
		},
		{
			name: "wrong password",
			repo: &fakeUserRepo{userByUsername: &model.SysUser{
				ID:       2,
				Username: "operator",
				Password: passwordHash,
				Status:   &activeStatus,
			}},
			password: "wrong123",
			wantErr:  ErrAuth,
		},
		{
			name: "disabled user",
			repo: &fakeUserRepo{userByUsername: &model.SysUser{
				ID:       2,
				Username: "operator",
				Password: passwordHash,
				Status:   &disabledStatus,
			}},
			password: "abc123",
			wantErr:  ErrAuth,
		},
		{
			name: "success",
			repo: &fakeUserRepo{userByUsername: &model.SysUser{
				ID:       2,
				Username: "operator",
				Password: passwordHash,
				Tel:      "18712345678",
				Status:   &activeStatus,
			}},
			password:   "abc123",
			wantStatus: activeStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username := "operator"
			if tt.wantErr == ErrUserNotFound {
				username = ""
			}
			service := &UserService{
				db:      &repo.Repository{},
				userDao: tt.repo,
			}
			got, err := service.Authenticate(testUserCtx(), username, tt.password)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected err %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected success, got %v", err)
			}
			if got.ID != 2 || got.Username != "operator" || got.Tel != "18712345678" || got.Status != tt.wantStatus {
				t.Fatalf("unexpected user info: %+v", got)
			}
		})
	}
}

func TestUserServiceGetRoleIdsByUserId(t *testing.T) {
	cacheKey := constant.CacheUserRolePrefix + "2"

	t.Run("invalid user id", func(t *testing.T) {
		service := &UserService{}
		_, err := service.GetRoleIdsByUserId(testUserCtx(), 0)
		if !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("cache hit", func(t *testing.T) {
		cache := newFakeCache()
		cache.values[cacheKey] = "[1,2]"
		repo := &fakeUserRepo{roleIds: []int32{3}}
		service := &UserService{db: userSecurityDB(t), userDao: repo, cache: cache}

		got, err := service.GetRoleIdsByUserId(testUserCtx(), 2)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if !equalInt32s(got, []int32{1, 2}) {
			t.Fatalf("expected cached role ids [1 2], got %v", got)
		}
		if repo.getRoleIDsCalls != 0 {
			t.Fatalf("expected dao not called, got %d calls", repo.getRoleIDsCalls)
		}
	})

	t.Run("cache miss reads dao and writes cache", func(t *testing.T) {
		cache := newFakeCache()
		repo := &fakeUserRepo{roleIds: []int32{3, 4}}
		service := &UserService{db: userSecurityDB(t), userDao: repo, cache: cache}

		got, err := service.GetRoleIdsByUserId(testUserCtx(), 2)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if !equalInt32s(got, []int32{3, 4}) {
			t.Fatalf("expected dao role ids [3 4], got %v", got)
		}
		if cache.sets[cacheKey] != "[3,4]" {
			t.Fatalf("expected cache set [3,4], got %q", cache.sets[cacheKey])
		}
	})

	t.Run("broken cache falls back to dao", func(t *testing.T) {
		cache := newFakeCache()
		cache.values[cacheKey] = "not-json"
		repo := &fakeUserRepo{roleIds: []int32{5}}
		service := &UserService{db: userSecurityDB(t), userDao: repo, cache: cache}

		got, err := service.GetRoleIdsByUserId(testUserCtx(), 2)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if !equalInt32s(got, []int32{5}) {
			t.Fatalf("expected dao role ids [5], got %v", got)
		}
	})

	t.Run("dao error", func(t *testing.T) {
		service := &UserService{
			db:      &repo.Repository{},
			userDao: &fakeUserRepo{roleIdsErr: errTestDAO},
			cache:   newFakeCache(),
		}

		_, err := service.GetRoleIdsByUserId(testUserCtx(), 2)
		if !errors.Is(err, errTestDAO) {
			t.Fatalf("expected dao error, got %v", err)
		}
	})
}

func equalInt32s(a, b []int32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// 只模拟事务连接；业务读写由 fake DAO 承接，意外 SQL 会使测试失败。
type userTestDriver struct{}
type userTestConn struct{}
type userTestTx struct{}

func (userTestDriver) Open(string) (driver.Conn, error)  { return userTestConn{}, nil }
func (userTestConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unexpected SQL") }
func (userTestConn) Close() error                        { return nil }
func (userTestConn) Begin() (driver.Tx, error)           { return userTestTx{}, nil }
func (userTestTx) Commit() error                         { return nil }
func (userTestTx) Rollback() error                       { return nil }

func init() { sql.Register("user-security-test", userTestDriver{}) }

func userSecurityDB(t *testing.T) *repo.Repository {
	t.Helper()
	db, err := sql.Open("user-security-test", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	gdb, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	return repo.NewRepository(gdb, nil)
}

type assignmentRepo struct {
	UserRepo
	roles    map[int32][]int32
	menus    map[int32][]int32
	rolesErr error
	menusErr error
	writes   int
	q        *query.Query
}

func (r *assignmentRepo) IsNameTelDuplicate(context.Context, string, string, int32) (bool, error) {
	return false, nil
}
func (r *assignmentRepo) CountRoleByIds(_ context.Context, q *query.Query, ids []int32) (int64, error) {
	r.q = q
	return int64(len(ids)), nil
}
func (r *assignmentRepo) GetRoleIdsByUserId(_ context.Context, q *query.Query, id int32) ([]int32, error) {
	if q == nil || (r.q != nil && q != r.q) {
		return nil, errors.New("role lookup escaped transaction")
	}
	r.q = q
	return r.roles[id], r.rolesErr
}
func (r *assignmentRepo) GetMenuIdsByRoleIds(_ context.Context, q *query.Query, ids []int32) ([]int32, error) {
	if q != r.q && r.q != nil {
		return nil, errors.New("authorization query escaped transaction")
	}
	var menus []int32
	for _, id := range ids {
		menus = append(menus, r.menus[id]...)
	}
	return menus, r.menusErr
}
func (r *assignmentRepo) GetUserById(_ context.Context, _ *query.Query, id int32) (*model.SysUser, error) {
	return &model.SysUser{ID: id}, nil
}
func (r *assignmentRepo) CreateUser(_ context.Context, _ *query.Query, user *model.SysUser) (*model.SysUser, error) {
	r.writes++
	user.ID = 10
	return user, nil
}
func (r *assignmentRepo) UpdateUser(_ context.Context, _ *query.Query, user *model.SysUser) (*model.SysUser, error) {
	r.writes++
	return user, nil
}
func (r *assignmentRepo) CreateUserRoleInBatches(context.Context, *query.Query, []*model.SysUserRole) error {
	r.writes++
	return nil
}
func (r *assignmentRepo) DeleteUserRole(context.Context, *query.Query, int32) error {
	r.writes++
	return nil
}

type assignmentLog struct{ calls int }

func (l *assignmentLog) CreateOperationLog(context.Context, *query.Query, *contract.OperationLogInput) error {
	l.calls++
	return nil
}

func TestUserRoleAssignment(t *testing.T) {
	for _, update := range []bool{false, true} {
		name := "create"
		if update {
			name = "update"
		}
		t.Run(name, func(t *testing.T) {
			for _, tc := range []struct {
				name          string
				operatorRoles []int32
				targetRoles   []int32
				assigned      []int32
				rolesErr      error
				menusErr      error
				wantErr       error
				forbidden     bool
			}{
				{name: "cannot grant super admin", operatorRoles: []int32{2}, assigned: []int32{1}, forbidden: true},
				{name: "cannot grant greater permissions", operatorRoles: []int32{2}, assigned: []int32{3}, forbidden: true},
				{name: "can grant permission subset", operatorRoles: []int32{2}, assigned: []int32{4}},
				{name: "super admin can grant any role", operatorRoles: []int32{1}, assigned: []int32{3}},
				{name: "super admin can grant super role", operatorRoles: []int32{1}, assigned: []int32{1}},
				{name: "no assigned roles", operatorRoles: []int32{2}},
				{name: "cannot grant without permissions", assigned: []int32{4}, forbidden: true},
				{name: "role lookup failure", assigned: []int32{4}, rolesErr: errTestDAO, wantErr: errTestDAO},
				{name: "menu lookup failure", operatorRoles: []int32{2}, assigned: []int32{4}, menusErr: errTestDAO, wantErr: errTestDAO},
			} {
				t.Run(tc.name, func(t *testing.T) {
					r := &assignmentRepo{roles: map[int32][]int32{2: tc.operatorRoles, 9: tc.targetRoles}, menus: map[int32][]int32{2: {10, 20}, 3: {10, 30}, 4: {10}}, rolesErr: tc.rolesErr, menusErr: tc.menusErr}
					log := &assignmentLog{}
					s := NewUserService(userSecurityDB(t), r, newFakeCache(), nil, log)
					ctx := context.WithValue(testUserCtx(), xauth.XUserId, int32(2))
					p := &UserParam{ID: 9, Username: "target", Password: "abc123", Tel: "18100000000", RoleIds: tc.assigned}
					var err error
					if update {
						_, err = s.UpdateUser(ctx, p)
					} else {
						_, err = s.CreateUser(ctx, p)
					}
					if tc.forbidden {
						var biz *e.BizError
						if !errors.As(err, &biz) || biz.Code != e.HttpForbidden {
							t.Fatalf("expected forbidden, got %v", err)
						}
					} else if tc.wantErr != nil {
						if !errors.Is(err, tc.wantErr) {
							t.Fatalf("expected %v, got %v", tc.wantErr, err)
						}
					} else if err != nil {
						t.Fatal(err)
					}
					if tc.forbidden || tc.wantErr != nil {
						if r.writes != 0 || log.calls != 0 {
							t.Fatal("rejected assignment wrote business data or audit")
						}
					} else if r.writes == 0 || log.calls != 1 {
						t.Fatal("authorized write/audit did not execute")
					}
				})
			}
		})
	}
}

func TestUserRoleAssignmentSelfEscalation(t *testing.T) {
	r := &assignmentRepo{roles: map[int32][]int32{2: {2}}}
	log := &assignmentLog{}
	s := NewUserService(userSecurityDB(t), r, newFakeCache(), nil, log)
	ctx := context.WithValue(testUserCtx(), xauth.XUserId, int32(2))
	_, err := s.UpdateUser(ctx, &UserParam{ID: 2, Username: "operator", Tel: "18100000000", RoleIds: []int32{constant.SuperAdminRoleId}})
	if !errors.Is(err, ErrUserRoleNotAuthorized) || r.writes != 0 || log.calls != 0 {
		t.Fatalf("expected self escalation denied without writes, got %v", err)
	}
}

type refreshStateRepo struct {
	UserRepo
	q   *query.Query
	ctx context.Context
}

func (r *refreshStateRepo) GetUserById(ctx context.Context, q *query.Query, id int32) (*model.SysUser, error) {
	r.q, r.ctx = q, ctx
	status := constant.UserStatusActive
	return &model.SysUser{ID: id, Status: &status}, nil
}

func TestValidateRefreshUserUsesPrimary(t *testing.T) {
	db := userSecurityDB(t)
	r := &refreshStateRepo{}
	s := NewUserService(db, r, nil, nil, nil)
	ctx := testUserCtx()
	if err := s.ValidateRefreshUser(ctx, 5); err != nil {
		t.Fatal(err)
	}
	if r.q != db.WriteQuery() || r.ctx != ctx {
		t.Fatal("refresh validation did not use primary query and caller context")
	}
}

func TestUserRoleAssignmentProtectsSuperAdmin(t *testing.T) {
	for _, roles := range [][]int32{nil, {4}} {
		r := &assignmentRepo{roles: map[int32][]int32{2: {2}, 9: {constant.SuperAdminRoleId}}, menus: map[int32][]int32{2: {10}, 4: {10}}}
		log := &assignmentLog{}
		s := NewUserService(userSecurityDB(t), r, newFakeCache(), nil, log)
		ctx := context.WithValue(testUserCtx(), xauth.XUserId, int32(2))
		_, err := s.UpdateUser(ctx, &UserParam{ID: 9, Username: "admin", Tel: "18100000000", RoleIds: roles})
		var biz *e.BizError
		if !errors.As(err, &biz) || biz.Code != e.HttpForbidden || r.writes != 0 || log.calls != 0 {
			t.Fatalf("expected protected admin with no writes, got %v", err)
		}
	}
}
