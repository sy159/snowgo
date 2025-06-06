// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

import (
	"time"
)

const TableNameUser = "user"

// User 用户表
type User struct {
	ID        int32      `gorm:"column:id;type:int(11);primaryKey;autoIncrement:true" json:"id"`
	Username  string     `gorm:"column:username;type:varchar(64);not null;index:idx_username,priority:1;comment:登录名，业务唯一" json:"username"`                            // 登录名，业务唯一
	Tel       string     `gorm:"column:tel;type:varchar(20);not null;index:idx_tel,priority:1;comment:手机号码" json:"tel"`                                               // 手机号码
	Nickname  *string    `gorm:"column:nickname;type:varchar(60);comment:用户昵称" json:"nickname"`                                                                       // 用户昵称
	Password  string     `gorm:"column:password;type:char(64);not null;comment:pwd" json:"password"`                                                                  // pwd
	Status    *string    `gorm:"column:status;type:varchar(20);not null;index:idx_status,priority:1;default:Active;comment:状态：Active 活跃，Disabled 禁用登录" json:"status"` // 状态：Active 活跃，Disabled 禁用登录
	IsDeleted bool       `gorm:"column:is_deleted;type:tinyint(1);not null;index:idx_is_deleted,priority:1;comment:是否删除：0=未删除，1=已删除" json:"is_deleted"`               // 是否删除：0=未删除，1=已删除
	CreatedAt *time.Time `gorm:"column:created_at;type:datetime(6);not null;default:CURRENT_TIMESTAMP(6)" json:"created_at"`
	UpdatedAt *time.Time `gorm:"column:updated_at;type:datetime(6);not null;default:CURRENT_TIMESTAMP(6)" json:"updated_at"`
}

// TableName User's table name
func (*User) TableName() string {
	return TableNameUser
}
