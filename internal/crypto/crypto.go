package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	SaltLen      = 32
	KeyLen       = 32
	ArgonTime    = 3
	ArgonMemory  = 64 * 1024
	ArgonThreads = 4
)

func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltLen)
	_, err := io.ReadFull(rand.Reader, salt)
	return salt, err
}

func DeriveKey(passphrase string, salt []byte) []byte {
	return argon2.IDKey([]byte(passphrase), salt, ArgonTime, ArgonMemory, ArgonThreads, KeyLen)
}

func Encrypt(plaintext []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

const verifierPlaintext = "maind-key-ok"

func CreateVerifier(key []byte) (string, error) {
	return Encrypt([]byte(verifierPlaintext), key)
}

func CheckVerifier(verifier string, key []byte) bool {
	plain, err := Decrypt(verifier, key)
	if err != nil {
		return false
	}
	return string(plain) == verifierPlaintext
}

func Decrypt(encoded string, key []byte) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return aesGCM.Open(nil, nonce, ciphertext, nil)
}
