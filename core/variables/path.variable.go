package variables

import (
	"os"
	"path/filepath"
	"strings"
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

// DomainToSnake converts a domain (e.g. localhost:5173 or sawang.tech) to a snake_case folder name.
// Any non-alphanumeric character is replaced by a single underscore, consolidating double underscores.
func DomainToSnake(domain string) string {
	var sb strings.Builder
	lastWasUnderscore := false
	for _, r := range domain {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
			lastWasUnderscore = false
		} else {
			if !lastWasUnderscore {
				sb.WriteRune('_')
				lastWasUnderscore = true
			}
		}
	}
	res := sb.String()
	res = strings.Trim(res, "_")
	return strings.ToLower(res)
}

