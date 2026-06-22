package functions

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
)

// DeriveKey derives a 32-byte AES key using machine ID and an optional rule encryption key.
func DeriveKey(ruleKey string) ([]byte, error) {
	macID, err := machineid.ID()
	if err != nil {
		macID = "lumbung-fs-fallback-machine-id"
	}
	combined := fmt.Sprintf("%s:%s", macID, ruleKey)
	u5 := uuid.NewSHA1(uuid.NameSpaceDNS, []byte(combined))
	hexStr := fmt.Sprintf("%x", u5[:])
	return []byte(hexStr), nil
}

// CompressZstd compresses input byte slice using zstd at the specified level
func CompressZstd(data []byte, level int) ([]byte, error) {
	encoderLevel := zstd.EncoderLevelFromZstd(level)
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(encoderLevel))
	if err != nil {
		return nil, err
	}
	defer encoder.Close()
	return encoder.EncodeAll(data, nil), nil
}

// DecompressZstd decompresses a zstd compressed byte slice
func DecompressZstd(data []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}
	defer decoder.Close()
	return decoder.DecodeAll(data, nil)
}

// EncryptAESGCM encrypts data using AES-256-GCM with a 32-byte key
func EncryptAESGCM(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// DecryptAESGCM decrypts AES-256-GCM encrypted data
func DecryptAESGCM(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// ProcessUploadData conditionally compresses and/or encrypts upload content
func ProcessUploadData(data []byte, compress bool, compressLevel int, encrypt bool, ruleKey string) ([]byte, error) {
	var err error
	if compress {
		data, err = CompressZstd(data, compressLevel)
		if err != nil {
			return nil, fmt.Errorf("failed to compress data: %w", err)
		}
	}
	if encrypt {
		key, err := DeriveKey(ruleKey)
		if err != nil {
			return nil, fmt.Errorf("failed to derive encryption key: %w", err)
		}
		data, err = EncryptAESGCM(data, key)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt data: %w", err)
		}
	}
	return data, nil
}

// ProcessDownloadData conditionally decrypts and/or decompresses content
func ProcessDownloadData(data []byte, compress bool, compressLevel int, encrypt bool, ruleKey string) ([]byte, error) {
	var err error
	if encrypt {
		key, err := DeriveKey(ruleKey)
		if err != nil {
			return nil, fmt.Errorf("failed to derive decryption key: %w", err)
		}
		data, err = DecryptAESGCM(data, key)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt data: %w", err)
		}
	}
	if compress {
		data, err = DecompressZstd(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress data: %w", err)
		}
	}
	return data, nil
}
