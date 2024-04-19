package main

import (
	"fmt"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"os"
	"snowgo/config"
	"snowgo/internal/dal"
	"sort"
	"strings"
	"time"
)

func init() {
	// 初始化配置文件
	config.InitConf(
		config.WithMysqlConf(), // 加载mysql配置
	)
}

var modelPkg = "snowgo/internal/dal/model"

// generate code
func main() {
	dbDSN := config.MysqlConf.DSN
	//dbDSN := "root:zx.123@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=true&loc=Local"
	db, err := gorm.Open(mysql.Open(dbDSN))
	if err != nil {
		fmt.Println("db conn is error:", err)
		return
	}
	if os.Args[1] == "add" {
		genModel(db)
	} else if os.Args[1] == "update" {
		genModelByOldTables(db)
	} else if os.Args[1] == "init" {
		genAllModel(db)
	} else if os.Args[1] == "query" {
		genQuery()
	}
}

func genModel(db *gorm.DB) {
	var tablesStr string
	fmt.Print("请输入table名称(逗号隔开)：")
	_, err := fmt.Scanln(&tablesStr)
	if err != nil {
		fmt.Println("table名输入错误", err.Error())
		return
	}
	if len(tablesStr) == 0 {
		fmt.Println("未输入table名称")
		os.Exit(1)
	}
	genModelByTables(db, tablesStr)
}

func genAllModel(db *gorm.DB) {
	migrator := db.Migrator()
	// 获取所有表名
	tables, _ := migrator.GetTables()
	genModelByTables(db, strings.Join(tables, ","))
}

func genModelByTables(db *gorm.DB, tablesStr string) {
	g := gen.NewGenerator(gen.Config{
		ModelPkgPath:   "./internal/dal/model",
		FieldNullable:  true, // 数据库中的字段可为空，则生成struct字段为指针类型
		FieldCoverable: true, //如果数据库中字段有默认值，则生成指针类型的字段，以避免零值（zero-value）
		//FieldSignable: true, //	Use signable type as field’s type based on column’s data type in database
		FieldWithIndexTag: true, //	为结构体生成gorm index tag，如gorm:"index:idx_name"，默认：false
		FieldWithTypeTag:  true, //	为结构体生成gorm type tag，如：gorm:"type:varchar(12)"，默认：false
	})

	g.UseDB(db)

	// specify diy mapping relationship
	var dataMap = map[string]func(gorm.ColumnType) (dataType string){
		// int mapping
		//"int": func(columnType gorm.ColumnType) (dataType string) {
		//	if n, ok := columnType.Nullable(); ok && n {
		//		return "*int32"
		//	}
		//	return "int32"
		//},

		// bool mapping
		"tinyint": func(columnType gorm.ColumnType) (dataType string) {
			ct, _ := columnType.ColumnType()
			if strings.HasPrefix(ct, "tinyint(1)") {
				return "bool"
			}
			return "byte"
		},
	}
	g.WithDataTypeMap(dataMap)

	g.WithModelNameStrategy(func(tableName string) (targetTableName string) {
		var prefix = "t_"
		return generator.CamelCase(strings.TrimPrefix(tableName, prefix))
	})

	g.WithFileNameStrategy(func(tableName string) (targetTableName string) {
		var prefix = "t_"
		return prefix + strings.TrimPrefix(tableName, prefix)
	})

	tables := strings.Split(tablesStr, ",")
	for _, table := range tables {
		g.GenerateModel(table)
		//g.GenerateModel(table)
	}

	// execute the action of code generation
	g.Execute()
	fmt.Println("生成model完成")

	time.Sleep(1 * time.Second)
	genQuery()
}

func genModelByOldTables(db *gorm.DB) {
	xx := make([]string, 0)
	for _, i := range dal.GetQueryModels() {
		if x, ok := i.(schema.Tabler); ok {
			xx = append(xx, x.TableName())
		}
	}
	genModelByTables(db, strings.Join(xx, ","))
}

func genQuery() {
	updateModelList(modelPkg)

	g := gen.NewGenerator(gen.Config{
		OutPath:        "./internal/dal/query",
		FieldNullable:  true, // 数据库中的字段可为空，则生成struct字段为指针类型
		FieldCoverable: true, //如果数据库中字段有默认值，则生成指针类型的字段，以避免零值（zero-value）
		//FieldSignable: true, //	Use signable type as field’s type based on column’s data type in database
		FieldWithIndexTag: true, //	为结构体生成gorm index tag，如gorm:"index:idx_name"，默认：false
		FieldWithTypeTag:  true, //	为结构体生成gorm type tag，如：gorm:"type:varchar(12)"，默认：false
	})
	g.ApplyBasic(dal.GetQueryModels()...)
	// execute the action of code generation
	g.Execute()
	fmt.Println("生成query完成")
}

func updateModelList(modelPkg string) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo,
		Fset: token.NewFileSet(),
	}, modelPkg)

	if err != nil {
		fmt.Println("error:", err)
		return
	}
	modelNames := make([]string, 0)
	for _, pkg := range pkgs {
		s := pkg.Types.Scope()
		for _, n := range s.Names() {
			lookup := s.Lookup(n)
			if _, ok := lookup.Type().Underlying().(*types.Struct); ok {
				modelNames = append(modelNames, fmt.Sprintf("&model.%s{}", n))
			}
		}
	}
	sort.Strings(modelNames)
	constantContext := fmt.Sprintf(`package dal

import "%s"

func GetQueryModels() []interface{} {
	return []interface{}{
		%s,
	}
}
`, modelPkg, strings.Join(modelNames, ",\n\t\t"))
	err = os.WriteFile("./internal/dal/query_model.go", []byte(constantContext), 0644)
	if err != nil {
		panic(err)
	}
	fmt.Println("更新model列表完成")
}
