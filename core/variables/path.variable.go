package variables

import (
	"os"
	"path/filepath"
)

const (
	BucketDir   = "bucket"
	DatabaseName = "data.db"
	PasswordFile = "password.txt"
)

// GetBucketPath returns the absolute or relative path to the bucket directory.
func GetBucketPath() string {
	return BucketDir
}

// GetDatabasePath returns the path to the SQLite3 database file.
func GetDatabasePath() string {
	return filepath.Join(BucketDir, DatabaseName)
}

// GetPasswordFilePath returns the path to the password credentials file.
func GetPasswordFilePath() string {
	return filepath.Join(BucketDir, PasswordFile)
}

// EnsureBucketDir ensures the bucket directory exists on disk.
func EnsureBucketDir() error {
	return os.MkdirAll(BucketDir, 0755)
}
