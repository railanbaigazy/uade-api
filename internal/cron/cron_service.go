package cron

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/railanbaigazy/uade-api/internal/app/models"
	"github.com/robfig/cron/v3"
)

type DBInterface interface {
	Select(dest interface{}, query string, args ...interface{}) error
}

type PublisherInterface interface {
	PublishGenerateContract(ctx context.Context, agreementID string) error
}

type CronService struct {
	DB        DBInterface
	Publisher PublisherInterface
	PDFPath   string
}

func NewCronService(db DBInterface, publisher PublisherInterface, pdfPath string) *CronService {
	return &CronService{
		DB:        db,
		Publisher: publisher,
		PDFPath:   pdfPath,
	}
}
func (c *CronService) Start() {
	cr := cron.New(cron.WithSeconds())

	_, err := cr.AddFunc("0 0 9 * * *", func() {
		if err := c.CheckAgreements(); err != nil {
			log.Printf("[CRON] CheckAgreements error: %v", err)
		}
	})

	if err != nil {
		log.Fatalf("failed to schedule CheckAgreements cron: %v", err)
	}

	_, err = cr.AddFunc("0 0 2 * * *", func() {
		if err := c.CleanupPDFs(); err != nil {
			log.Printf("[CRON] CleanupPDFs error: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("failed to schedule CleanupPDFs cron: %v", err)
	}

	cr.Start()
	log.Println("[CRON] CronService started")
}

func (c *CronService) CheckAgreements() error {
	log.Println("[CRON] Checking agreements due soon...")

	query := `
		SELECT id, lender_id, borrower_id, post_id, principal_amount, interest_rate,
		       total_amount, currency, due_date, status
		FROM agreements
		WHERE status = 'active' AND due_date <= $1
	`
	dueSoon := time.Now().Add(72 * time.Hour)
	agreements := []models.Agreement{}

	if err := c.DB.Select(&agreements, query, dueSoon); err != nil {
		return fmt.Errorf("failed to select agreements: %w", err)
	}

	for _, ag := range agreements {
		err := c.Publisher.PublishGenerateContract(context.Background(), fmt.Sprintf("%d", ag.ID))
		if err != nil {
			log.Printf("[CRON] failed to send reminder for agreement %d: %v", ag.ID, err)
			continue
		}
		log.Printf("[CRON] Reminder sent for agreement %d", ag.ID)
	}

	log.Printf("[CRON] Total agreements processed: %d", len(agreements))
	return nil
}

func (c *CronService) CleanupPDFs() error {
	log.Println("[CRON] Cleaning up old PDFs...")

	if _, err := os.Stat(c.PDFPath); os.IsNotExist(err) {
		log.Printf("[CRON] PDF path does not exist: %s", c.PDFPath)
		return nil
	}

	err := filepath.Walk(c.PDFPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("[CRON] walk error for %s: %v", path, err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if time.Since(info.ModTime()) > 7*24*time.Hour {
			if err := os.Remove(path); err != nil {
				log.Printf("[CRON] failed to remove %s: %v", path, err)
			} else {
				log.Printf("[CRON] removed old PDF: %s", path)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("walk error: %w", err)
	}

	return nil
}
