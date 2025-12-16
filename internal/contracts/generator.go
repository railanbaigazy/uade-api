package contracts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/railanbaigazy/uade-api/internal/app/models"
)

type Generator struct {
	basePath string
}

func NewGenerator(basePath string) *Generator {
	return &Generator{basePath: basePath}
}

func (g *Generator) Generate(_ context.Context, agreement *models.Agreement) (string, string, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetCreationDate(time.Unix(0, 0))
	pdf.SetTitle(fmt.Sprintf("Agreement #%d", agreement.ID), false)

	pdf.AddPage()
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 10, "Қарыз шарты / Договор займа")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 11)
	pdf.MultiCell(0, 8, g.buildKazakhSection(agreement), "", "L", false)
	pdf.Ln(4)
	pdf.MultiCell(0, 8, g.buildRussianSection(agreement), "", "L", false)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return "", "", fmt.Errorf("pdf output: %w", err)
	}

	hashBytes := sha256.Sum256(buf.Bytes())
	hash := hex.EncodeToString(hashBytes[:])

	dir := filepath.Join(g.basePath, "agreements", fmt.Sprintf("%d", agreement.ID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", fmt.Errorf("create dir: %w", err)
	}

	path := filepath.Join(dir, "contract.pdf")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return "", "", fmt.Errorf("write pdf: %w", err)
	}

	return path, hash, nil
}

func (g *Generator) buildKazakhSection(a *models.Agreement) string {
	return fmt.Sprintf(
		"Қарыз шарты #%d\nБеруші (ID: %d)\nАлатын (ID: %d)\nНегізгі сома: %.2f %s\nСыйақы мөлшері: %.2f\nҚайтару сомасы: %.2f %s\nӨтеу күні: %s\nКелісім мерзімі: %s",
		a.ID, a.LenderID, a.BorrowerID, a.PrincipalAmount, a.Currency, a.InterestRate, a.TotalAmount, a.Currency,
		a.DueDate.Format("2006-01-02"), g.formatStartDate(a.StartDate),
	)
}

func (g *Generator) buildRussianSection(a *models.Agreement) string {
	return fmt.Sprintf(
		"Договор займа #%d\nЗаймодавец (ID: %d)\nЗаемщик (ID: %d)\nОсновная сумма: %.2f %s\nСтавка: %.2f\nИтого к возврату: %.2f %s\nСрок возврата: %s\nДата начала: %s",
		a.ID, a.LenderID, a.BorrowerID, a.PrincipalAmount, a.Currency, a.InterestRate, a.TotalAmount, a.Currency,
		a.DueDate.Format("2006-01-02"), g.formatStartDate(a.StartDate),
	)
}

func (g *Generator) formatStartDate(start *time.Time) string {
	if start == nil {
		return "—"
	}
	return start.Format("2006-01-02")
}
