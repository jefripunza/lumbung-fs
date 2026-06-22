package auth

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"lumbung-fs/core/middlewares"
	"lumbung-fs/core/variables"
)

const DefaultMD5Hash = "e10adc3949ba59abbe56e057f20f883e" // MD5 of "123456"

// InitCredentialsFile ensures password.txt exists with default hash if missing
func InitCredentialsFile() {
	passFile := variables.GetPasswordFilePath()
	if _, err := os.Stat(passFile); os.IsNotExist(err) {
		log.Println("Credentials file password.txt is missing. Creating with default credentials (admin:123456)...")
		err := os.WriteFile(passFile, []byte(DefaultMD5Hash), 0644)
		if err != nil {
			log.Printf("Warning: Failed to create default password.txt: %v", err)
		}
	}
}

// VerifyMD5 compares a plaintext string to an MD5 hash (case-insensitive)
func VerifyMD5(plaintext string, hash string) bool {
	hasher := md5.New()
	hasher.Write([]byte(plaintext))
	sum := hasher.Sum(nil)
	computed := hex.EncodeToString(sum)
	return strings.ToLower(computed) == strings.ToLower(hash)
}

// LoginHandler processes admin login credentials and returns JWT
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	passFile := variables.GetPasswordFilePath()
	hashBytes, err := os.ReadFile(passFile)
	if err != nil {
		http.Error(w, "Internal server error: unable to verify credentials", http.StatusInternalServerError)
		return
	}

	storedHash := strings.TrimSpace(string(hashBytes))

	// Allow verification of:
	// 1. Plain MD5 of password
	// 2. MD5 of username:password
	// 3. Special fallback case (admin/admin -> accepts 123456 or admin)
	valid := false
	if VerifyMD5(credentials.Password, storedHash) {
		valid = true
	} else if VerifyMD5(credentials.Username+":"+credentials.Password, storedHash) {
		valid = true
	} else if storedHash == DefaultMD5Hash {
		// Default password.txt contains MD5 of "123456"
		if credentials.Username == "admin" && (credentials.Password == "123456" || credentials.Password == "admin") {
			valid = true
		}
	}

	if !valid {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Invalid username or password"}`))
		return
	}

	token, err := middlewares.GenerateJWT(credentials.Username)
	if err != nil {
		http.Error(w, "Internal server error: token generation failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}
