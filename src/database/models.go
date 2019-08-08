package database

import (
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
	"time"
)

type Person struct {
	gorm.Model
	UID 		string
	Name 		string
	Timestamp 	time.Time
	Traits		postgres.Jsonb
}
