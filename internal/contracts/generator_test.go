package contracts

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/railanbaigazy/uade-api/internal/app/models"
	"github.com/stretchr/testify/require"
)

func TestGenerator_GenerateCreatesPDFAndHash(t *testing.T) {
	tmp := t.TempDir()
	gen := NewGenerator(tmp)

	now := time.Now()
	start := now
	agreement := models.Agreement{
		ID:              1,
		LenderID:        10,
		BorrowerID:      20,
		PrincipalAmount: 1000,
		InterestRate:    0.1,
		TotalAmount:     1100,
		Currency:        "KZT",
		StartDate:       &start,
		DueDate:         now.AddDate(0, 1, 0),
		Status:          "active",
	}

	path, hash, err := gen.Generate(context.Background(), &agreement)

	// Skip test if fonts are not available (common in CI environments)
	if err != nil && strings.Contains(err.Error(), "DejaVuSans") {
		t.Skip("Fonts not available, skipping PDF generation test")
	}

	require.NoError(t, err)

	expectedPath := filepath.Join(tmp, "agreements", "1", "contract.pdf")
	require.Equal(t, expectedPath, path)
	require.Len(t, hash, 64) // sha256 hex

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.False(t, info.IsDir())
	require.Greater(t, info.Size(), int64(0))
}
