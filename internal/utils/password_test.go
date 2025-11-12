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

func TestHashPassword_ErrorHandling(t *testing.T) {
	veryLongPassword := ""
	for i := 0; i < 100; i++ {
		veryLongPassword += "a"
	}

	hash, err := HashPassword(veryLongPassword)

	require.Error(t, err, "should return error for password > 72 bytes")

	require.Empty(t, hash, "hash should be empty when error occurs")

	require.Contains(t, err.Error(), "password length exceeds 72 bytes")
}

func TestCheckPassword_InvalidHash(t *testing.T) {
	// malformed hash should return false (not panic)
	result := CheckPassword("not-a-valid-bcrypt-hash", "password")
	require.False(t, result)

	// empty hash should return false
	result = CheckPassword("", "password")
	require.False(t, result)
}

func TestHashPassword_RandomSalt(t *testing.T) {
	password := "testPassword123"
	hash1, err1 := HashPassword(password)
	hash2, err2 := HashPassword(password)

	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NotEqual(t, hash1, hash2)
	require.True(t, CheckPassword(hash1, password))
	require.True(t, CheckPassword(hash2, password))
}

func TestHashPassword_SpecialCharacters(t *testing.T) {
	specialPassword := "p@$$wÖrd!#$%^&*()"
	hash, err := HashPassword(specialPassword)

	require.NoError(t, err)
	require.NotEmpty(t, hash)
	require.True(t, CheckPassword(hash, specialPassword))
	require.False(t, CheckPassword(hash, "p@$$word!#$%^&*()"))
}
