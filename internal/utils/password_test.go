package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashAndCheckPassword(t *testing.T) {
	pass := "secret123"
	hash, err := HashPassword(pass)
	require.NoError(t, err)
	require.NotEmpty(t, hash)
	require.True(t, CheckPassword(hash, pass))
	require.False(t, CheckPassword(hash, "wrongpass"))
}
