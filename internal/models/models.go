package models

import (
	"time"
)

// 示例model，具体参数配置可参考  https://gorm.io/zh_CN/docs/create.html
// decimal类型decimal(10,2) 表示一共10位，小数点后面保留两位12345678.12
type exampleModel struct {
	// primaryKey	指定列为主键
	// column 指定列名
	// type	列数据类型 type:varchar(30)
	// unique	指定列为唯一
	// default	指定列的默认值
	// size 指定列大小
	// not null	指定列为NOT NULL
	// index	普通索引，多个字段使用相同的名称则创建复合索引，通过设置priority表示所顺序 index:idx_xxx,priority:2
	// uniqueIndex	唯一索引
	// autoCreateTime 当为int字段，默认为s，也可以使用 nano/milli区别毫秒跟纳秒 autoCreateTime:milli
	// autoUpdateTime
	ID       uint      `gorm:"primaryKey" json:"id"`
	UserName string    `gorm:"column:username;size:20" json:"username"`
	Password string    `gorm:"column:password;type:varchar(32)" json:"password"`
	Tel      string    `gorm:"column:tel;size:20;uniqueIndex;not null;" json:"tel"`
	Sex      uint8     `gorm:"column:sex;type:tinyint;default:0;comment:'性别 -1表示未知 0表示男 1表示女'" json:"sex"`
	GroupID  int       `gorm:"column:group_id;index" json:"order_id"`
	Created  time.Time `gorm:"column:created;autoCreateTime" json:"created"`
}

func (exampleModel) TableName() string {
	return "example_table"
}

// Model 只有主键id的model
type Model struct {
	ID uint `gorm:"column:id;primaryKey" json:"id"`
}

// BaseModel 初始化id，created，updated字段的model
type BaseModel struct {
	Model
	Created time.Time `gorm:"column:created;autoCreateTime" json:"created"`
	Updated time.Time `gorm:"column:updated;autoUpdateTime" json:"updated"`
}
