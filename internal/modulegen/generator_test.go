package modulegen

import (
	"errors"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOptionsValidation(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		wantErr string
	}{
		{name: "default options", opts: Options{}},
		{name: "explicit domain", opts: Options{Domain: "business", Module: "order2"}},
		{name: "uppercase module", opts: Options{Module: "Order"}, wantErr: "invalid module"},
		{name: "hyphen", opts: Options{Module: "sales-order"}, wantErr: "invalid module"},
		{name: "path separator", opts: Options{Module: "sales/order"}, wantErr: "invalid module"},
		{name: "keyword", opts: Options{Module: "type"}, wantErr: "invalid module"},
		{name: "leading digit", opts: Options{Module: "2order"}, wantErr: "invalid module"},
		{name: "main module", opts: Options{Module: "main"}, wantErr: "reserved module"},
		{name: "main domain", opts: Options{Domain: "main", Module: "order"}, wantErr: "reserved domain"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := normalizeOptions(tt.opts)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeOptions error: %v", err)
			}
			if tt.opts.Domain == "" && opts.Domain != "admin" {
				t.Fatalf("domain = %q, want admin", opts.Domain)
			}
			if tt.opts.Module == "" && opts.Module != "demo" {
				t.Fatalf("module = %q, want demo", opts.Module)
			}
		})
	}
}

func TestGenerateModuleNamedGinUsesSafeAPIAlias(t *testing.T) {
	root := newFixtureProject(t)
	if _, err := Generate(Options{Root: root, Module: "gin"}); err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	content := readFixtureFile(t, root, "internal/router/admin/gin_router.go")
	for _, want := range []string{
		`"github.com/gin-gonic/gin"`,
		`moduleAPI "snowgo/internal/api/admin/gin"`,
		`ginGroup.GET("/:id", moduleAPI.GetGinInfo)`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("gin router missing %q:\n%s", want, content)
		}
	}
}

func TestGenerateFiles(t *testing.T) {
	root := newFixtureProject(t)
	result, err := Generate(Options{Root: root, Module: "order"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	want := []string{
		"internal/dao/admin/order/order.go",
		"internal/service/admin/order/order.go",
		"internal/api/admin/order/order.go",
		"internal/router/admin/order_router.go",
	}
	for _, relative := range want {
		path := filepath.Join(root, relative)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("generated file %s: %v", relative, err)
		}
		if _, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.AllErrors); err != nil {
			t.Fatalf("parse generated file %s: %v", relative, err)
		}
	}
	if len(result.Created) != len(want) {
		t.Fatalf("created files = %v, want %v", result.Created, want)
	}

	serviceContent := readFixtureFile(t, root, "internal/service/admin/order/order.go")
	for _, fragment := range []string{
		"type OrderInfo struct",
		"func (s *OrderService) GetOrderInfo(_ context.Context, id int32) (*OrderInfo, error)",
		"return &OrderInfo{ID: id}, nil",
	} {
		if !strings.Contains(serviceContent, fragment) {
			t.Errorf("service missing %q:\n%s", fragment, serviceContent)
		}
	}

	apiContent := readFixtureFile(t, root, "internal/api/admin/order/order.go")
	for _, fragment := range []string{
		"type OrderInfo struct",
		"func GetOrderInfo(c *gin.Context)",
		"id := xgin.ParsePathID32(c)",
		"container := di.GetOrderContainer(c)",
		"container.OrderService.GetOrderInfo(ctx, id)",
		"xresponse.FailByError(c, e.HttpBadRequest)",
		"xresponse.FailByError(c, e.HttpInternalServerError)",
		"xresponse.Success(c, &OrderInfo{ID: info.ID})",
	} {
		if !strings.Contains(apiContent, fragment) {
			t.Errorf("api missing %q:\n%s", fragment, apiContent)
		}
	}

	routerContent := readFixtureFile(t, root, "internal/router/admin/order_router.go")
	for _, fragment := range []string{
		`moduleAPI "snowgo/internal/api/admin/order"`,
		`orderGroup.GET("/:id", moduleAPI.GetOrderInfo)`,
		"详情示例（仅需 JWTAuth，正式业务按需增加 PermissionAuth）",
	} {
		if !strings.Contains(routerContent, fragment) {
			t.Errorf("module router missing %q:\n%s", fragment, routerContent)
		}
	}
}

func TestGenerateRegistersDI(t *testing.T) {
	root := newFixtureProject(t)
	result, err := Generate(Options{Root: root, Module: "order"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	content := readFixtureFile(t, root, "internal/di/container.go")
	wants := []string{
		`adminOrderDao "snowgo/internal/dao/admin/order"`,
		`adminOrderService "snowgo/internal/service/admin/order"`,
		"SystemContainer\n\tOrderContainer",
		`type OrderContainer struct`,
		`OrderService *adminOrderService.OrderService`,
		`adminOrderDao := adminOrderDao.NewOrderDao(repository)`,
		`adminOrderService := adminOrderService.NewOrderService(repository, adminOrderDao)`,
		"// order\n\tcontainer.OrderContainer",
		`container.OrderContainer = OrderContainer{`,
		`OrderService: adminOrderService`,
		`// GetOrderContainer 获取注入的order service等`,
		`func GetOrderContainer(c *gin.Context) *OrderContainer`,
		`return &container.OrderContainer`,
	}
	for _, want := range wants {
		if !strings.Contains(content, want) {
			t.Errorf("container.go missing %q:\n%s", want, content)
		}
	}
	if !containsString(result.Updated, "internal/di/container.go") {
		t.Fatalf("updated files = %v, want internal/di/container.go", result.Updated)
	}
	assertBefore(t, content, "operationLogDao :=", "adminOrderDao :=")
	assertBefore(t, content, "userService :=", "adminOrderService :=")
	assertBefore(t, content, "container.SystemContainer =", "container.OrderContainer =")
}

func TestGenerateRegistersAdminRouter(t *testing.T) {
	root := newFixtureProject(t)
	result, err := Generate(Options{Root: root, Module: "order"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	content := readFixtureFile(t, root, "internal/router/admin/router.go")
	if !strings.Contains(content, "orderRouters(protected)") {
		t.Fatalf("admin router missing order registration:\n%s", content)
	}
	assertBefore(t, content, "accountRouters(protected)", "systemRouters(protected)")
	assertBefore(t, content, "systemRouters(protected)", "orderRouters(protected)")
	if !strings.Contains(content, "orderRouters(protected)   // order 相关") {
		t.Fatalf("admin router missing order comment:\n%s", content)
	}
	if !containsString(result.Updated, "internal/router/admin/router.go") {
		t.Fatalf("updated files = %v", result.Updated)
	}
	if containsString(result.Updated, "internal/router/router.go") {
		t.Fatalf("root router should remain unchanged when admin is already registered: %v", result.Updated)
	}
	rootRouter := readFixtureFile(t, root, "internal/router/router.go")
	if strings.Count(rootRouter, "admin.Register") != 1 {
		t.Fatalf("admin registration should appear once:\n%s", rootRouter)
	}
}

func TestGenerateRegistersCustomDomainRouter(t *testing.T) {
	root := newFixtureProject(t)
	result, err := Generate(Options{Root: root, Domain: "business", Module: "order"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	domainRouter := readFixtureFile(t, root, "internal/router/business/router.go")
	if !strings.Contains(domainRouter, "orderRouters(protected)") {
		t.Fatalf("business router missing order registration:\n%s", domainRouter)
	}
	rootRouter := readFixtureFile(t, root, "internal/router/router.go")
	if !strings.Contains(rootRouter, `businessRouter "snowgo/internal/router/business"`) {
		t.Fatalf("root router missing business import:\n%s", rootRouter)
	}
	if !strings.Contains(rootRouter, "businessRouter.Register") {
		t.Fatalf("root router missing business registration:\n%s", rootRouter)
	}
	for _, want := range []string{
		"admin.Register, // 后台管理相关路由",
		"businessRouter.Register,",
		"// 注册其他分组下的路由",
	} {
		if !strings.Contains(rootRouter, want) {
			t.Errorf("root router should preserve %q:\n%s", want, rootRouter)
		}
	}
	for _, path := range []string{"internal/router/business/router.go", "internal/router/router.go"} {
		if !containsString(result.Created, path) && !containsString(result.Updated, path) {
			t.Fatalf("result does not include %s: %+v", path, result)
		}
	}
}

func TestGenerateRegistersExistingCustomDomainInRootRouter(t *testing.T) {
	root := newFixtureProject(t)
	writeFixtureFile(t, root, "internal/router/business/router.go", `package business

import "github.com/gin-gonic/gin"

func Register(r *gin.RouterGroup) {
	business := r.Group("/business")
	protected := business.Group("", middleware.JWTAuth())
	{
		_ = protected
	}
}
`)

	if _, err := Generate(Options{Root: root, Domain: "business", Module: "order"}); err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	rootRouter := readFixtureFile(t, root, "internal/router/router.go")
	if !strings.Contains(rootRouter, "businessRouter.Register") {
		t.Fatalf("root router missing business registration:\n%s", rootRouter)
	}
}

func TestGenerateRejectsUnprotectedExistingDomainRouter(t *testing.T) {
	root := newFixtureProject(t)
	routerPath := "internal/router/business/router.go"
	writeFixtureFile(t, root, routerPath, `package business

import "github.com/gin-gonic/gin"

func Register(r *gin.RouterGroup) {
	business := r.Group("/business")
	protected := business.Group("")
	{
		_ = protected
	}
}
`)
	originalDI := readFixtureFile(t, root, "internal/di/container.go")

	_, err := Generate(Options{Root: root, Domain: "business", Module: "order"})
	if err == nil || !strings.Contains(err.Error(), "middleware.JWTAuth") {
		t.Fatalf("error = %v, want middleware.JWTAuth", err)
	}
	if got := readFixtureFile(t, root, "internal/di/container.go"); got != originalDI {
		t.Fatal("DI changed after unprotected Router rejection")
	}
	if _, err := os.Stat(filepath.Join(root, "internal/api/business/order/order.go")); !os.IsNotExist(err) {
		t.Fatalf("API should not be created, stat error = %v", err)
	}
}

func TestGenerateRefusesDuplicateRouterRegistration(t *testing.T) {
	root := newFixtureProject(t)
	routerPath := "internal/router/admin/router.go"
	writeFixtureFile(t, root, routerPath, `package admin

import "github.com/gin-gonic/gin"

func Register(r *gin.RouterGroup) {
	admin := r.Group("/admin")
	protected := admin.Group("", middleware.JWTAuth())
	{
		accountRouters(protected)
		systemRouters(protected)
		orderRouters(protected)
	}
}
`)
	originalDI := readFixtureFile(t, root, "internal/di/container.go")
	originalRouter := readFixtureFile(t, root, routerPath)

	_, err := Generate(Options{Root: root, Module: "order"})
	if err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("error = %v, want already registered", err)
	}
	if got := readFixtureFile(t, root, "internal/di/container.go"); got != originalDI {
		t.Fatal("DI changed after duplicate Router rejection")
	}
	if got := readFixtureFile(t, root, routerPath); got != originalRouter {
		t.Fatal("Router changed after duplicate Router rejection")
	}
	for _, relative := range []string{
		"internal/dao/admin/order/order.go",
		"internal/service/admin/order/order.go",
		"internal/api/admin/order/order.go",
		"internal/router/admin/order_router.go",
	} {
		if _, err := os.Stat(filepath.Join(root, relative)); !os.IsNotExist(err) {
			t.Fatalf("%s should not exist, stat error = %v", relative, err)
		}
	}
}

func TestGenerateUsesDomainQualifiedDIAliases(t *testing.T) {
	root := newFixtureProject(t)
	if _, err := Generate(Options{Root: root, Domain: "business", Module: "system"}); err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	content := readFixtureFile(t, root, "internal/di/container.go")
	for _, want := range []string{
		`businessSystemDao "snowgo/internal/dao/business/system"`,
		`businessSystemService "snowgo/internal/service/business/system"`,
		`BusinessSystemContainer`,
		`func GetBusinessSystemContainer(c *gin.Context) *BusinessSystemContainer`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("container.go missing %q:\n%s", want, content)
		}
	}
}

func TestGenerateRejectsMiddlewareDomain(t *testing.T) {
	_, err := normalizeOptions(Options{Domain: "middleware", Module: "order"})
	if err == nil || !strings.Contains(err.Error(), "reserved domain") {
		t.Fatalf("error = %v, want reserved domain", err)
	}
}

func TestGenerateRefusesOverwrite(t *testing.T) {
	root := newFixtureProject(t)
	writeFixtureFile(t, root, "internal/api/admin/order/order.go", "package order\n")
	originalDI := readFixtureFile(t, root, "internal/di/container.go")

	_, err := Generate(Options{Root: root, Module: "order"})
	if err == nil || !strings.Contains(err.Error(), "module already exists") {
		t.Fatalf("error = %v, want module already exists", err)
	}
	if got := readFixtureFile(t, root, "internal/di/container.go"); got != originalDI {
		t.Fatal("DI changed after overwrite rejection")
	}
	if _, err := os.Stat(filepath.Join(root, "internal/dao/admin/order/order.go")); !os.IsNotExist(err) {
		t.Fatalf("DAO should not be created, stat error = %v", err)
	}
}

func TestGeneratePreflightFailureWritesNothing(t *testing.T) {
	root := newFixtureProject(t)
	writeFixtureFile(t, root, "internal/di/container.go", "package di\nfunc broken(")

	_, err := Generate(Options{Root: root, Module: "order"})
	if err == nil || !strings.Contains(err.Error(), "parse container.go") {
		t.Fatalf("error = %v, want parse container.go", err)
	}
	for _, relative := range []string{
		"internal/dao/admin/order/order.go",
		"internal/service/admin/order/order.go",
		"internal/api/admin/order/order.go",
		"internal/router/admin/order_router.go",
	} {
		if _, err := os.Stat(filepath.Join(root, relative)); !os.IsNotExist(err) {
			t.Fatalf("%s should not exist, stat error = %v", relative, err)
		}
	}
}

func TestGenerateRollsBackUpdatedFilesAfterWriteFailure(t *testing.T) {
	root := newFixtureProject(t)
	diPath := "internal/di/container.go"
	routerPath := filepath.Join(root, "internal/router/admin/router.go")
	originalDI := readFixtureFile(t, root, diPath)
	originalRouter := readFixtureFile(t, root, "internal/router/admin/router.go")
	originalWriteExistingFile := writeExistingFile
	writeExistingFile = func(path string, content []byte) error {
		if path == routerPath {
			return errors.New("injected Router write failure")
		}
		return originalWriteExistingFile(path, content)
	}
	t.Cleanup(func() { writeExistingFile = originalWriteExistingFile })

	_, err := Generate(Options{Root: root, Module: "order"})
	if err == nil || !strings.Contains(err.Error(), "write internal/router/admin/router.go") {
		t.Fatalf("error = %v, want Router write failure", err)
	}
	if got := readFixtureFile(t, root, diPath); got != originalDI {
		t.Fatal("DI was not restored after Router write failure")
	}
	if got := readFixtureFile(t, root, "internal/router/admin/router.go"); got != originalRouter {
		t.Fatal("Router changed after failed write")
	}
	for _, relative := range []string{
		"internal/dao/admin/order/order.go",
		"internal/service/admin/order/order.go",
		"internal/api/admin/order/order.go",
		"internal/router/admin/order_router.go",
	} {
		if _, err := os.Stat(filepath.Join(root, relative)); !os.IsNotExist(err) {
			t.Fatalf("%s should be cleaned up, stat error = %v", relative, err)
		}
	}
}

func TestGenerateRefusesExistingRegistration(t *testing.T) {
	root := newFixtureProject(t)
	writeFixtureFile(t, root, "internal/di/container.go", `package di

type Container struct {
	OrderService int
}

func NewContainer() (*Container, error) {
	repository := struct{}{}
	_ = repository
	container := &Container{}
	return container, nil
}
`)

	_, err := Generate(Options{Root: root, Module: "order"})
	if err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("error = %v, want already registered", err)
	}
	if _, err := os.Stat(filepath.Join(root, "internal/dao/admin/order/order.go")); !os.IsNotExist(err) {
		t.Fatalf("DAO should not be created, stat error = %v", err)
	}
}

func TestGenerateRefusesExistingDIConstructorVariable(t *testing.T) {
	root := newFixtureProject(t)
	diPath := "internal/di/container.go"
	originalDI := readFixtureFile(t, root, diPath)
	updatedDI := strings.Replace(
		originalDI,
		"operationLogDao := systemDao.NewOperationLogDao(repository)",
		"operationLogDao := systemDao.NewOperationLogDao(repository)\n\tadminOrderDao := accountDao.NewUserDao(repository)\n\t_ = adminOrderDao",
		1,
	)
	writeFixtureFile(t, root, diPath, updatedDI)
	originalRouter := readFixtureFile(t, root, "internal/router/admin/router.go")

	_, err := Generate(Options{Root: root, Module: "order"})
	if err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("error = %v, want already registered", err)
	}
	if got := readFixtureFile(t, root, diPath); got != updatedDI {
		t.Fatal("DI changed after duplicate constructor variable rejection")
	}
	if got := readFixtureFile(t, root, "internal/router/admin/router.go"); got != originalRouter {
		t.Fatal("Router changed after duplicate constructor variable rejection")
	}
	if _, err := os.Stat(filepath.Join(root, "internal/api/admin/order/order.go")); !os.IsNotExist(err) {
		t.Fatalf("API should not be created, stat error = %v", err)
	}
}

func newFixtureProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFixtureFile(t, root, "go.mod", "module snowgo\n\ngo 1.26.0\n")
	writeFixtureFile(t, root, "internal/di/container.go", `package di

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

// NewContainer 构造所有依赖
func NewContainer() (*Container, error) {
	repository := struct{}{}
	// 构造Dao
	userDao := accountDao.NewUserDao(repository)
	operationLogDao := systemDao.NewOperationLogDao(repository)
	// 构造Service依赖
	operationLogService := systemService.NewOperationLogService(repository, operationLogDao)
	userService := accountService.NewUserService(repository, userDao)
	container := &Container{}
	// account
	container.AccountContainer = AccountContainer{UserService: userService}
	// system
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
	writeFixtureFile(t, root, "internal/router/admin/router.go", `package admin

import "github.com/gin-gonic/gin"

func Register(r *gin.RouterGroup) {
	admin := r.Group("/admin")
	protected := admin.Group("", middleware.JWTAuth())
	{
		accountRouters(protected) // 账户相关
		systemRouters(protected)  // 系统设置相关
	}
}
`)
	writeFixtureFile(t, root, "internal/router/router.go", `package router

import (
	"github.com/gin-gonic/gin"
	"snowgo/internal/router/admin"
)

type option func(*gin.RouterGroup)

func loadRouter(apiGroup *gin.RouterGroup) {
	options := []option{ // 服务分组路由
		admin.Register, // 后台管理相关路由
	}
	// 注册其他分组下的路由
	for _, opt := range options {
		opt(apiGroup)
	}
}
`)
	return root
}

func writeFixtureFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", relative, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", relative, err)
	}
}

func readFixtureFile(t *testing.T, root, relative string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, relative))
	if err != nil {
		t.Fatalf("read %s: %v", relative, err)
	}
	return string(content)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func assertBefore(t *testing.T, content, first, second string) {
	t.Helper()
	firstIndex := strings.Index(content, first)
	secondIndex := strings.Index(content, second)
	if firstIndex < 0 || secondIndex < 0 || firstIndex >= secondIndex {
		t.Fatalf("want %q before %q:\n%s", first, second, content)
	}
}
