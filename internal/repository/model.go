package repository

import "database/sql"

type Model interface {
	TableName() string
	GetID() any
	SetID(any)
	ScanRow(scanner *sql.Row) error
	ValuesForCreate() []any
}
