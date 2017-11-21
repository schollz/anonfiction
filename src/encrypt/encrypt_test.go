package encrypt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncrypt(t *testing.T) {
	encrypted, err := Encrypt("hello, world", "secure passphrase")
	assert.Nil(t, err)
	decrypted, err := Decrypt(encrypted, "secure passphrase")
	assert.Nil(t, err)
	assert.Equal(t, "hello, world", decrypted)
	_, err = Decrypt(encrypted, "wrong passphrase")
	assert.NotNil(t, err)
}
