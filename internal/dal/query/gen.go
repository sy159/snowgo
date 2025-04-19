// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package query

import (
	"context"
	"database/sql"

	"gorm.io/gorm"

	"gorm.io/gen"

	"gorm.io/plugin/dbresolver"
)

func Use(db *gorm.DB, opts ...gen.DOOption) *Query {
	return &Query{
		db:       db,
		Menu:     newMenu(db, opts...),
		Role:     newRole(db, opts...),
		RoleMenu: newRoleMenu(db, opts...),
		User:     newUser(db, opts...),
		UserRole: newUserRole(db, opts...),
	}
}

type Query struct {
	db *gorm.DB

	Menu     menu
	Role     role
	RoleMenu roleMenu
	User     user
	UserRole userRole
}

func (q *Query) Available() bool { return q.db != nil }

func (q *Query) clone(db *gorm.DB) *Query {
	return &Query{
		db:       db,
		Menu:     q.Menu.clone(db),
		Role:     q.Role.clone(db),
		RoleMenu: q.RoleMenu.clone(db),
		User:     q.User.clone(db),
		UserRole: q.UserRole.clone(db),
	}
}

func (q *Query) ReadDB() *Query {
	return q.ReplaceDB(q.db.Clauses(dbresolver.Read))
}

func (q *Query) WriteDB() *Query {
	return q.ReplaceDB(q.db.Clauses(dbresolver.Write))
}

func (q *Query) ReplaceDB(db *gorm.DB) *Query {
	return &Query{
		db:       db,
		Menu:     q.Menu.replaceDB(db),
		Role:     q.Role.replaceDB(db),
		RoleMenu: q.RoleMenu.replaceDB(db),
		User:     q.User.replaceDB(db),
		UserRole: q.UserRole.replaceDB(db),
	}
}

type queryCtx struct {
	Menu     *menuDo
	Role     *roleDo
	RoleMenu *roleMenuDo
	User     *userDo
	UserRole *userRoleDo
}

func (q *Query) WithContext(ctx context.Context) *queryCtx {
	return &queryCtx{
		Menu:     q.Menu.WithContext(ctx),
		Role:     q.Role.WithContext(ctx),
		RoleMenu: q.RoleMenu.WithContext(ctx),
		User:     q.User.WithContext(ctx),
		UserRole: q.UserRole.WithContext(ctx),
	}
}

func (q *Query) Transaction(fc func(tx *Query) error, opts ...*sql.TxOptions) error {
	return q.db.Transaction(func(tx *gorm.DB) error { return fc(q.clone(tx)) }, opts...)
}

func (q *Query) Begin(opts ...*sql.TxOptions) *QueryTx {
	tx := q.db.Begin(opts...)
	return &QueryTx{Query: q.clone(tx), Error: tx.Error}
}

type QueryTx struct {
	*Query
	Error error
}

func (q *QueryTx) Commit() error {
	return q.db.Commit().Error
}

func (q *QueryTx) Rollback() error {
	return q.db.Rollback().Error
}

func (q *QueryTx) SavePoint(name string) error {
	return q.db.SavePoint(name).Error
}

func (q *QueryTx) RollbackTo(name string) error {
	return q.db.RollbackTo(name).Error
}
