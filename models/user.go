package models

import (
	"github.com/shopspring/decimal"
)

// User 用户表
type User struct {
	BaseModel
	UserName     string          `gorm:"column:username" json:"username"`                       // 用户名
	Password     string          `gorm:"column:password" json:"password"`                       // 用户密码
	Tel          string          `gorm:"column:tel" json:"tel"`                                 // 用户手机号
	Sex          uint8           `gorm:"column:sex;default:2" json:"sex"`                       // 性别 2表示未知 0表示男 1表示女
	WalletAmount decimal.Decimal `gorm:"column:wallet_amount;default:0.0" json:"wallet_amount"` // 余额
	Status       uint8           `gorm:"column:status;default:2" json:"status"`                 // 用户状态 2表示被删除 0表示不可用 1表示可用
}

func (User) TableName() string {
	return "user"
}
