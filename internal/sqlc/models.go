// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0

package sqlc

import (
	"database/sql"
)

type UserWebsite struct {
	WebsiteUuid sql.NullString
	UserUuid    sql.NullString
	AccessTime  sql.NullTime
	GroupName   sql.NullString
}

type Website struct {
	Uuid       sql.NullString
	Url        sql.NullString
	Title      sql.NullString
	Content    sql.NullString
	UpdateTime sql.NullTime
}
