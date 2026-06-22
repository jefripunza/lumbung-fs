package database

import (
	"lumbung-fs/core/variables"
	
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// DB is the global gorm database connection instance
var DB *gorm.DB

// Connect establishes the connection to SQLite3 using a pure-Go driver
func Connect() (*gorm.DB, error) {
	if err := variables.EnsureBucketDir(); err != nil {
		return nil, err
	}
	
	dbPath := variables.GetDatabasePath()
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	
	DB = db
	return db, nil
}
