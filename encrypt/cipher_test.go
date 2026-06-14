package encrypt

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAESGCM_Roundtrip(t *testing.T) {
	c, err := NewAESGCM([]byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)

	ct, err := c.Encrypt("hello, configx")
	require.NoError(t, err)
	require.NotEqual(t, "hello, configx", ct)

	pt, err := c.Decrypt(ct)
	require.NoError(t, err)
	require.Equal(t, "hello, configx", pt)
}

func TestAESGCM_KeySizeValidation(t *testing.T) {
	_, err := NewAESGCM([]byte("too-short"))
	require.Error(t, err)
}

func TestMarker(t *testing.T) {
	require.True(t, IsEncrypted("enc(abcd)"))
	require.False(t, IsEncrypted("plain"))
	v, ok := Unwrap("enc(payload)")
	require.True(t, ok)
	require.Equal(t, "payload", v)
	require.Equal(t, "enc(x)", Wrap("x"))
}
