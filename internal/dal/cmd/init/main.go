package main

import (
	"database/sql"
	"fmt"
	"os"
	"snowgo/config"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func init() {
	// 初始化配置文件
	config.Init("./config")
}

func main() {
	cfg := config.Get()
	dbDSN := cfg.Mysql.DSN
	//dbDSN := "root:zx.123@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=true&loc=Local"
	db, err := sql.Open("mysql", dbDSN)
	if err != nil {
		fmt.Println("连接数据库失败: ", err)
	}
	defer db.Close()

	// 读取 SQL 文件内容
	content, err := os.ReadFile("docs/sql/init.sql")
	if err != nil {
		fmt.Println("读取 init.sql err: ", err)
	}

	// 拆分 SQL 语句
	stmts := strings.Split(string(content), ";")

	// 使用事务执行所有语句
	tx, err := db.Begin()
	if err != nil {
		fmt.Println("开启事务失败: ", err)
	}
	for _, stmt := range stmts {
		sqlStr := strings.TrimSpace(stmt)
		if sqlStr == "" {
			continue
		}
		if _, err := tx.Exec(sqlStr); err != nil {
			fmt.Printf("执行 SQL 失败: %s; 错误: %v\n", sqlStr, err)
			_ = tx.Rollback()
			fmt.Println("初始化中断")
		}
	}
	// 提交事务
	if err := tx.Commit(); err != nil {
		fmt.Println("提交事务失败: ", err)
	}

	fmt.Println("初始化数据完成")
}
