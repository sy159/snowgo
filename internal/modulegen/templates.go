package modulegen

import (
	"bytes"
	"go/format"
	"text/template"
)

type templateData struct {
	ModulePath    string
	Domain        string
	Module        string
	TypeName      string
	ContainerName string
	DaoAlias      string
	ServiceAlias  string
	RouterAlias   string
}

var moduleTemplates = map[string]string{
	"dao": `package {{.Module}}

import "{{.ModulePath}}/internal/dal/repo"

type {{.TypeName}}Dao struct {
	repo *repo.Repository
}

func New{{.TypeName}}Dao(repository *repo.Repository) *{{.TypeName}}Dao {
	return &{{.TypeName}}Dao{repo: repository}
}
`,
	"service": `package {{.Module}}

import (
	"context"

	"{{.ModulePath}}/internal/dal/repo"
	{{.Module}}Dao "{{.ModulePath}}/internal/dao/{{.Domain}}/{{.Module}}"
)

type {{.TypeName}}Service struct {
	db         *repo.Repository
	{{.Module}}Dao *{{.Module}}Dao.{{.TypeName}}Dao
}

type {{.TypeName}}Info struct {
	ID int32
}

func New{{.TypeName}}Service(repository *repo.Repository, dao *{{.Module}}Dao.{{.TypeName}}Dao) *{{.TypeName}}Service {
	return &{{.TypeName}}Service{db: repository, {{.Module}}Dao: dao}
}

// Get{{.TypeName}}Info 获取{{.Module}}详情示例
func (s *{{.TypeName}}Service) Get{{.TypeName}}Info(_ context.Context, id int32) (*{{.TypeName}}Info, error) {
	return &{{.TypeName}}Info{ID: id}, nil
}
`,
	"api": `package {{.Module}}

import (
	"errors"

	"github.com/gin-gonic/gin"
	"{{.ModulePath}}/internal/di"
	e "{{.ModulePath}}/pkg/xerror"
	"{{.ModulePath}}/pkg/xgin"
	"{{.ModulePath}}/pkg/xlogger"
	"{{.ModulePath}}/pkg/xresponse"
)

type {{.TypeName}}Info struct {
	ID int32 ` + "`json:\"id\"`" + `
}

// Get{{.TypeName}}Info 获取{{.Module}}详情示例
func Get{{.TypeName}}Info(c *gin.Context) {
	id := xgin.ParsePathID32(c)
	if id < 1 {
		xresponse.FailByError(c, e.HttpBadRequest)
		return
	}
	ctx := c.Request.Context()

	container := di.Get{{.ContainerName}}(c)
	info, err := container.{{.TypeName}}Service.Get{{.TypeName}}Info(ctx, id)
	if err != nil {
		var bizErr *e.BizError
		if errors.As(err, &bizErr) {
			xresponse.FailByError(c, bizErr.Code)
			return
		}
		xlogger.ErrorfCtx(ctx, "get {{.Module}} info is err: %v", err)
		xresponse.FailByError(c, e.HttpInternalServerError)
		return
	}
	xresponse.Success(c, &{{.TypeName}}Info{ID: info.ID})
}
`,
	"router": `package {{.Domain}}

import (
	"github.com/gin-gonic/gin"
	moduleAPI "{{.ModulePath}}/internal/api/{{.Domain}}/{{.Module}}"
)

func {{.Module}}Routers(r *gin.RouterGroup) {
	{{.Module}}Group := r.Group("/{{.Module}}")
	// 详情示例（仅需 JWTAuth，正式业务按需增加 PermissionAuth）
	{{.Module}}Group.GET("/:id", moduleAPI.Get{{.TypeName}}Info)
}
	`,
}

const domainRouterTemplate = `package {{.Domain}}

import (
	"github.com/gin-gonic/gin"
	"{{.ModulePath}}/internal/router/middleware"
)

func Register(r *gin.RouterGroup) {
	{{.Domain}} := r.Group("/{{.Domain}}")
	protected := {{.Domain}}.Group("", middleware.JWTAuth())
	{
		{{.Module}}Routers(protected) // {{.Module}} 相关
	}
}
`

func renderModuleTemplate(name string, data templateData) ([]byte, error) {
	tmpl, err := template.New(name).Parse(moduleTemplates[name])
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		return nil, err
	}
	return format.Source(output.Bytes())
}

func renderDomainRouter(data templateData) ([]byte, error) {
	tmpl, err := template.New("domain-router").Parse(domainRouterTemplate)
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		return nil, err
	}
	return format.Source(output.Bytes())
}
