package utils

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateJWT(t *testing.T) {
	tests := []struct {
		name      string
		userID    int
		secret    string
		expectErr bool
	}{
		{
			name:      "valid token generation",
			userID:    1,
			secret:    "my-secret-key",
			expectErr: false,
		},
		{
			name:      "different user ID",
			userID:    12345,
			secret:    "test-secret",
			expectErr: false,
		},
		{
			name:      "long secret key",
			userID:    999,
			secret:    "very-long-secret-key-for-jwt-signing-purposes",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateJWT(tt.userID, tt.secret)

			if (err != nil) != tt.expectErr {
				t.Errorf("GenerateJWT() error = %v, wantErr %v", err, tt.expectErr)
				return
			}

			if err == nil && token == "" {
				t.Errorf("GenerateJWT() returned empty token")
				return
			}
		})
	}
}

func TestGenerateJWTTokenValidation(t *testing.T) {
	userID := 42
	secret := "test-secret"

	token, err := GenerateJWT(userID, secret)
	if err != nil {
		t.Fatalf("GenerateJWT() error = %v", err)
	}

	if token == "" {
		t.Fatal("GenerateJWT() returned empty token")
	}

	// Parse and validate the token
	parsedToken, err := jwt.ParseWithClaims(token, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if !parsedToken.Valid {
		t.Fatal("Token is not valid")
	}

	// Verify claims
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("Failed to extract claims from token")
	}

	// Check user_id claim
	extractedUserID, ok := claims["user_id"].(float64)
	if !ok {
		t.Fatal("user_id claim is missing or has wrong type")
	}

	if int(extractedUserID) != userID {
		t.Errorf("user_id = %v, want %v", int(extractedUserID), userID)
	}

	// Check that exp claim exists
	if _, ok := claims["exp"]; !ok {
		t.Fatal("exp claim is missing")
	}

	// Check that iat claim exists
	if _, ok := claims["iat"]; !ok {
		t.Fatal("iat claim is missing")
	}
}

func TestGenerateJWTWithDifferentSecrets(t *testing.T) {
	userID := 123
	secret1 := "secret-one"
	secret2 := "secret-two"

	token1, err := GenerateJWT(userID, secret1)
	if err != nil {
		t.Fatalf("GenerateJWT() error = %v", err)
	}

	token2, err := GenerateJWT(userID, secret2)
	if err != nil {
		t.Fatalf("GenerateJWT() error = %v", err)
	}

	// Tokens should be different
	if token1 == token2 {
		t.Error("Tokens generated with different secrets should not be identical")
	}

	// Token1 should not be valid with secret2
	_, err = jwt.ParseWithClaims(token1, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret2), nil
	})

	if err == nil {
		t.Error("Token signed with secret1 should not be valid with secret2")
	}
}

func TestGenerateJWTTokenStructure(t *testing.T) {
	userID := 100
	secret := "my-secret"

	token, err := GenerateJWT(userID, secret)
	if err != nil {
		t.Fatalf("GenerateJWT() error = %v", err)
	}

	// Token should be a valid JWT string (contains 3 parts separated by dots)
	parsedToken, err := jwt.ParseWithClaims(token, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	// Verify signing method is HS256
	if parsedToken.Method.Alg() != "HS256" {
		t.Errorf("Signing method = %v, want HS256", parsedToken.Method.Alg())
	}
}
