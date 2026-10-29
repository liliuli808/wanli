package secretbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	box, err := New("test-key-material")
	require.NoError(t, err)

	token, err := box.Encrypt("abc123%2C1793262567%2Cxyz*11")
	require.NoError(t, err)
	assert.NotContains(t, token, "abc123", "密文不应包含明文")

	got, err := box.Decrypt(token)
	require.NoError(t, err)
	assert.Equal(t, "abc123%2C1793262567%2Cxyz*11", got)
}

func TestEncrypt_ProducesDifferentCiphertext(t *testing.T) {
	box, err := New("test-key-material")
	require.NoError(t, err)

	a, err := box.Encrypt("same")
	require.NoError(t, err)
	b, err := box.Encrypt("same")
	require.NoError(t, err)
	assert.NotEqual(t, a, b, "每次加密应使用随机 nonce")
}

func TestDecrypt_WrongKey(t *testing.T) {
	box1, err := New("key-one")
	require.NoError(t, err)
	box2, err := New("key-two")
	require.NoError(t, err)

	token, err := box1.Encrypt("secret")
	require.NoError(t, err)

	_, err = box2.Decrypt(token)
	assert.Error(t, err)
}

func TestNew_EmptyKey(t *testing.T) {
	_, err := New("")
	assert.Error(t, err)
}
