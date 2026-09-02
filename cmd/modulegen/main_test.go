package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunUsesDefaults(t *testing.T) {
	root := newCommandFixture(t)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{"-root", root}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "internal/api/admin/demo/demo.go") {
		t.Fatalf("stdout = %q, want default admin/demo output", stdout.String())
	}
}

func newCommandFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeCommandFixture(t, root, "go.mod", "module snowgo\n\ngo 1.26.0\n")
	writeCommandFixture(t, root, "internal/di/container.go", `package di

type Container struct {
	AccountContainer
	SystemContainer
}

type AccountContainer struct {
	UserService any
}

type SystemContainer struct {
	OperationLogService any
}

func NewContainer() (*Container, error) {
	repository := struct{}{}
	userDao := accountDao.NewUserDao(repository)
	operationLogDao := systemDao.NewOperationLogDao(repository)
	operationLogService := systemService.NewOperationLogService(repository, operationLogDao)
	userService := accountService.NewUserService(repository, userDao)
	container := &Container{}
	container.AccountContainer = AccountContainer{UserService: userService}
	container.SystemContainer = SystemContainer{OperationLogService: operationLogService}
	return container, nil
}

func GetContainer(c *gin.Context) *Container {
	return nil
}

func GetAccountContainer(c *gin.Context) *AccountContainer {
	container := GetContainer(c)
	return &container.AccountContainer
}

func GetSystemContainer(c *gin.Context) *SystemContainer {
	container := GetContainer(c)
	return &container.SystemContainer
}
`)
	writeCommandFixture(t, root, "internal/router/admin/router.go", `package admin

import "github.com/gin-gonic/gin"

func Register(r *gin.RouterGroup) {
	admin := r.Group("/admin")
	protected := admin.Group("", middleware.JWTAuth())
	{
		_ = protected
	}
}
`)
	writeCommandFixture(t, root, "internal/router/router.go", `package router

import (
	"github.com/gin-gonic/gin"
	"snowgo/internal/router/admin"
)

type option func(*gin.RouterGroup)

func loadRouter(apiGroup *gin.RouterGroup) {
	options := []option{admin.Register}
	for _, opt := range options {
		opt(apiGroup)
	}
}
`)
	return root
}

func writeCommandFixture(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", relative, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", relative, err)
	}
}
