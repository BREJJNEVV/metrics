// Package migrations содержит внедрённые SQL-файлы для миграций базы данных.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
