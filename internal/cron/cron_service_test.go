package cron

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/railanbaigazy/uade-api/internal/app/models"
)

type DummyPublisher struct{}

func (d *DummyPublisher) PublishGenerateContract(ctx context.Context, agreementID string) error {
	if agreementID == "error" {
		return errors.New("publish error")
	}
	return nil
}

type DummyDB struct{}

func (d *DummyDB) Select(dest interface{}, query string, args ...interface{}) error {
	agreements := []models.Agreement{
		{ID: 1, Status: "active", DueDate: time.Now().Add(48 * time.Hour)},
		{ID: 2, Status: "active", DueDate: time.Now().Add(72 * time.Hour)},
	}
	ptr := dest.(*[]models.Agreement)
	*ptr = agreements
	return nil
}

func TestCheckAgreements(t *testing.T) {
	db := &DummyDB{}
	pub := &DummyPublisher{}

	cronService := &CronService{
		DB:        db,
		Publisher: pub,
	}

	err := cronService.CheckAgreements()
	if err != nil {
		t.Errorf("CheckAgreements returned error: %v", err)
	}
}

type failingDB struct{}

var errDBFailure = errors.New("boom")

func (f *failingDB) Select(dest interface{}, query string, args ...interface{}) error {
	return errDBFailure
}

type flakyPublisher struct {
	failOn string
	calls  int
}

func (f *flakyPublisher) PublishGenerateContract(ctx context.Context, agreementID string) error {
	f.calls++
	if agreementID == f.failOn {
		return errors.New("publish error")
	}
	return nil
}

func TestCheckAgreements_Errors(t *testing.T) {
	t.Run("db failure bubbles", func(t *testing.T) {
		cronService := &CronService{
			DB:        &failingDB{},
			Publisher: &DummyPublisher{},
		}

		err := cronService.CheckAgreements()
		if !errors.Is(err, errDBFailure) {
			t.Fatalf("expected db error, got %v", err)
		}
	})

	t.Run("publish errors are skipped", func(t *testing.T) {
		db := &DummyDB{}
		pub := &flakyPublisher{failOn: "2"}

		cronService := &CronService{
			DB:        db,
			Publisher: pub,
		}

		err := cronService.CheckAgreements()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pub.calls != 2 {
			t.Fatalf("expected 2 publish attempts, got %d", pub.calls)
		}
	})
}

func TestCleanupPDFs(t *testing.T) {
	cronService := &CronService{PDFPath: "./test_tmp_pdf"}

	_ = os.MkdirAll(cronService.PDFPath, 0755)
	f, _ := os.Create(cronService.PDFPath + "/old.pdf")
	f.Close()
	_ = os.Chtimes(cronService.PDFPath+"/old.pdf", time.Now().Add(-8*24*time.Hour), time.Now().Add(-8*24*time.Hour))

	err := cronService.CleanupPDFs()
	if err != nil {
		t.Errorf("CleanupPDFs returned error: %v", err)
	}

	if _, err := os.Stat(cronService.PDFPath + "/old.pdf"); !os.IsNotExist(err) {
		t.Errorf("CleanupPDFs did not remove old file")
	}

	os.RemoveAll(cronService.PDFPath)
}
