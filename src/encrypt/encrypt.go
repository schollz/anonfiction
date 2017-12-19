package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// Encrypt returns a payload that is iv.salt.encrypteddata
func Encrypt(plaintext string, passphrase string) (encrypted string, err error) {
	key, saltBytes := deriveKey([]byte(passphrase), nil)
	ivBytes := make([]byte, 12)
	// http://nvlpubs.nist.gov/nistpubs/Legacy/SP/nistspecialpublication800-38d.pdf
	// Section 8.2
	rand.Read(ivBytes)
	b, _ := aes.NewCipher(key)
	aesgcm, _ := cipher.NewGCM(b)
	encrypted = hex.EncodeToString(ivBytes) + "." + hex.EncodeToString(saltBytes) + "." + hex.EncodeToString(aesgcm.Seal(nil, ivBytes, []byte(plaintext), nil))
	return
}

// Decrypt takes a iv.salt.encrypted paylod and returns the original data
func Decrypt(encrypted string, passphrase string) (plaintext string, err error) {
	splitData := strings.Split(encrypted, ".")
	iv, _ := hex.DecodeString(splitData[0])
	salt, _ := hex.DecodeString(splitData[1])
	data, _ := hex.DecodeString(splitData[2])
	key, _ := deriveKey([]byte(passphrase), salt)
	b, _ := aes.NewCipher(key)
	aesgcm, _ := cipher.NewGCM(b)

	plaintextB, err := aesgcm.Open(nil, iv, data, nil)
	plaintext = string(plaintextB)
	return
}

func deriveKey(passphrase []byte, salt []byte) ([]byte, []byte) {
	if salt == nil {
		salt = make([]byte, 8)
		// http://www.ietf.org/rfc/rfc2898.txt
		// Salt.
		rand.Read(salt)
	}
	return pbkdf2.Key(passphrase, salt, 1000, 32, sha256.New), salt
}
