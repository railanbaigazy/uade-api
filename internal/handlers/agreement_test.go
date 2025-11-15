package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/railanbaigazy/uade-api/internal/app/models"
	"github.com/railanbaigazy/uade-api/internal/utils"
	"github.com/stretchr/testify/require"
)

// Create
func TestAgreementHandler_Create_BadJSON(t *testing.T) {
	db, _ := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	req := httptest.NewRequest(http.MethodPost, "/api/agreements", strings.NewReader(`{`))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "invalid json")
}

func TestAgreementHandler_Create_MissingPostID(t *testing.T) {
	db, _ := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	body := `{"principal_amount": 1000, "interest_rate": 0.1}`
	req := httptest.NewRequest(http.MethodPost, "/api/agreements", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "post_id is required")
}

func TestAgreementHandler_Create_InvalidPrincipalAmount(t *testing.T) {
	db, _ := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	body := `{"post_id": 1, "principal_amount": -100, "interest_rate": 0.1, "due_date": "2026-01-01", "payment_frequency": "one_time", "number_of_payments": 1}`
	req := httptest.NewRequest(http.MethodPost, "/api/agreements", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "principal_amount must be greater than 0")
}

func TestAgreementHandler_Create_InvalidInterestRate(t *testing.T) {
	db, _ := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	body := `{"post_id": 1, "principal_amount": 1000, "interest_rate": -0.5, "due_date": "2026-01-01", "payment_frequency": "one_time", "number_of_payments": 1}`
	req := httptest.NewRequest(http.MethodPost, "/api/agreements", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "interest_rate cannot be negative")
}

func TestAgreementHandler_Create_InvalidPaymentFrequency(t *testing.T) {
	db, _ := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	body := `{"post_id": 1, "principal_amount": 1000, "interest_rate": 0.1, "due_date": "2026-01-01", "payment_frequency": "invalid", "number_of_payments": 1}`
	req := httptest.NewRequest(http.MethodPost, "/api/agreements", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "invalid payment_frequency")
}

func TestAgreementHandler_Create_InvalidDueDateFormat(t *testing.T) {
	db, _ := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	body := `{"post_id": 1, "principal_amount": 1000, "interest_rate": 0.1, "due_date": "01/01/2026", "payment_frequency": "one_time", "number_of_payments": 1}`
	req := httptest.NewRequest(http.MethodPost, "/api/agreements", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "invalid due_date format")
}

func TestAgreementHandler_Create_PostNotFound(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	body := `{"post_id": 999, "principal_amount": 1000, "interest_rate": 0.1, "due_date": "2026-12-31", "payment_frequency": "one_time", "number_of_payments": 1}`
	req := httptest.NewRequest(http.MethodPost, "/api/agreements", strings.NewReader(body))
	req.Header.Set("X-User-ID", "2")

	mock.ExpectQuery(`SELECT author_id, type FROM posts WHERE id=\$1`).
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	rec := httptest.NewRecorder()
	h.Create(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), "post not found")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAgreementHandler_Create_WrongPostType(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	body := `{"post_id": 1, "principal_amount": 1000, "interest_rate": 0.1, "due_date": "2026-12-31", "payment_frequency": "one_time", "number_of_payments": 1}`
	req := httptest.NewRequest(http.MethodPost, "/api/agreements", strings.NewReader(body))
	req.Header.Set("X-User-ID", "2")

	mock.ExpectQuery(`SELECT author_id, type FROM posts`).
		WillReturnRows(sqlmock.NewRows([]string{"author_id", "type"}).AddRow(1, "borrow"))

	rec := httptest.NewRecorder()
	h.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "can only create agreements for lend posts")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAgreementHandler_Create_OwnPost(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	body := `{"post_id": 1, "principal_amount": 1000, "interest_rate": 0.1, "due_date": "2026-12-31", "payment_frequency": "one_time", "number_of_payments": 1}`
	req := httptest.NewRequest(http.MethodPost, "/api/agreements", strings.NewReader(body))
	req.Header.Set("X-User-ID", "1")

	mock.ExpectQuery(`SELECT author_id, type FROM posts`).
		WillReturnRows(sqlmock.NewRows([]string{"author_id", "type"}).AddRow(1, "lend"))

	rec := httptest.NewRecorder()
	h.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "cannot create agreement with your own post")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAgreementHandler_Create_Success(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	body := `{"post_id": 1, "principal_amount": 1000, "interest_rate": 0.1, "due_date": "2026-12-31", "payment_frequency": "monthly", "number_of_payments": 12}`
	req := httptest.NewRequest(http.MethodPost, "/api/agreements", strings.NewReader(body))
	req.Header.Set("X-User-ID", "2")

	mock.ExpectQuery(`SELECT author_id, type FROM posts`).
		WillReturnRows(sqlmock.NewRows([]string{"author_id", "type"}).AddRow(1, "lend"))

	rows := sqlmock.NewRows([]string{"id", "created_at"}).
		AddRow(10, time.Now())

	mock.ExpectQuery(`INSERT INTO agreements`).
		WillReturnRows(rows)

	rec := httptest.NewRecorder()
	h.Create(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var agreement models.Agreement
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&agreement))
	require.Equal(t, 10, agreement.ID)
	require.Equal(t, int64(1), agreement.LenderID)
	require.Equal(t, int64(2), agreement.BorrowerID)
	require.Equal(t, 1000.0, agreement.PrincipalAmount)
	require.Equal(t, 1100.0, agreement.TotalAmount) // 1000 * 1.1

	require.NoError(t, mock.ExpectationsWereMet())
}

// GetUserAgreements
func TestAgreementHandler_GetUserAgreements_Success(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/api/agreements", nil)
	req.Header.Set("X-User-ID", "1")

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "lender_id", "borrower_id", "post_id",
		"principal_amount", "interest_rate", "total_amount", "currency",
		"created_at", "accepted_at", "disbursed_at", "start_date", "due_date", "completed_at",
		"payment_frequency", "number_of_payments",
		"status", "contract_url", "contract_hash",
	}).AddRow(
		1, 1, 2, 10,
		1000.0, 0.1, 1100.0, "KZT",
		now, nil, nil, nil, now.AddDate(0, 1, 0), nil,
		"one_time", 1,
		"active", nil, nil,
	)

	mock.ExpectQuery(`SELECT .* FROM agreements WHERE \(lender_id = \$1 OR borrower_id = \$1\)`).
		WithArgs(int64(1)).
		WillReturnRows(rows)

	rec := httptest.NewRecorder()
	h.GetUserAgreements(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"id":1`)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAgreementHandler_GetUserAgreements_WithStatusFilter(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/api/agreements?status=active", nil)
	req.Header.Set("X-User-ID", "1")

	rows := sqlmock.NewRows([]string{
		"id", "lender_id", "borrower_id", "post_id",
		"principal_amount", "interest_rate", "total_amount", "currency",
		"created_at", "accepted_at", "disbursed_at", "start_date", "due_date", "completed_at",
		"payment_frequency", "number_of_payments",
		"status", "contract_url", "contract_hash",
	})

	mock.ExpectQuery(`SELECT .* FROM agreements WHERE \(lender_id = \$1 OR borrower_id = \$1\) AND status = \$2`).
		WithArgs(int64(1), "active").
		WillReturnRows(rows)

	rec := httptest.NewRecorder()
	h.GetUserAgreements(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	require.NoError(t, mock.ExpectationsWereMet())
}

// GetByID
func TestAgreementHandler_GetByID_NotFound(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/api/agreements/999", nil)
	req.Header.Set("X-User-ID", "1")
	req.SetPathValue("id", "999")

	mock.ExpectQuery(`SELECT .* FROM agreements WHERE id = \$1`).
		WithArgs("999").
		WillReturnError(sql.ErrNoRows)

	rec := httptest.NewRecorder()
	h.GetByID(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), "agreement not found")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAgreementHandler_GetByID_Forbidden(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/api/agreements/1", nil)
	req.Header.Set("X-User-ID", "3")
	req.SetPathValue("id", "1")

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "lender_id", "borrower_id", "post_id",
		"principal_amount", "interest_rate", "total_amount", "currency",
		"created_at", "accepted_at", "disbursed_at", "start_date", "due_date", "completed_at",
		"payment_frequency", "number_of_payments",
		"status", "contract_url", "contract_hash",
	}).AddRow(
		1, 1, 2, 10,
		1000.0, 0.1, 1100.0, "KZT",
		now, nil, nil, nil, now.AddDate(0, 1, 0), nil,
		"one_time", 1,
		"pending", nil, nil,
	)

	mock.ExpectQuery(`SELECT .* FROM agreements WHERE id = \$1`).
		WithArgs("1").
		WillReturnRows(rows)

	rec := httptest.NewRecorder()
	h.GetByID(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "not authorized")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAgreementHandler_GetByID_Success(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/api/agreements/1", nil)
	req.Header.Set("X-User-ID", "1")
	req.SetPathValue("id", "1")

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "lender_id", "borrower_id", "post_id",
		"principal_amount", "interest_rate", "total_amount", "currency",
		"created_at", "accepted_at", "disbursed_at", "start_date", "due_date", "completed_at",
		"payment_frequency", "number_of_payments",
		"status", "contract_url", "contract_hash",
	}).AddRow(
		1, 1, 2, 10,
		1000.0, 0.1, 1100.0, "KZT",
		now, nil, nil, nil, now.AddDate(0, 1, 0), nil,
		"one_time", 1,
		"pending", nil, nil,
	)

	mock.ExpectQuery(`SELECT .* FROM agreements WHERE id = \$1`).
		WithArgs("1").
		WillReturnRows(rows)

	rec := httptest.NewRecorder()
	h.GetByID(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var agreement models.Agreement
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&agreement))
	require.Equal(t, 1, agreement.ID)

	require.NoError(t, mock.ExpectationsWereMet())
}

// Accept
func TestAgreementHandler_Accept_NotFound(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	req := httptest.NewRequest(http.MethodPost, "/api/agreements/999/accept", nil)
	req.Header.Set("X-User-ID", "1")
	req.SetPathValue("id", "999")

	mock.ExpectQuery(`SELECT \* FROM agreements WHERE id=\$1`).
		WithArgs("999").
		WillReturnError(sql.ErrNoRows)

	rec := httptest.NewRecorder()
	h.Accept(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), "agreement not found")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAgreementHandler_Accept_NotLender(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	req := httptest.NewRequest(http.MethodPost, "/api/agreements/1/accept", nil)
	req.Header.Set("X-User-ID", "2")
	req.SetPathValue("id", "1")

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "lender_id", "borrower_id", "post_id",
		"principal_amount", "interest_rate", "total_amount", "currency",
		"created_at", "accepted_at", "disbursed_at", "start_date", "due_date", "completed_at",
		"payment_frequency", "number_of_payments",
		"status", "contract_url", "contract_hash",
	}).AddRow(
		1, 1, 2, 10,
		1000.0, 0.1, 1100.0, "KZT",
		now, nil, nil, nil, now.AddDate(0, 1, 0), nil,
		"one_time", 1,
		"pending", nil, nil,
	)

	mock.ExpectQuery(`SELECT \* FROM agreements WHERE id=\$1`).
		WithArgs("1").
		WillReturnRows(rows)

	rec := httptest.NewRecorder()
	h.Accept(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "only lender can accept")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAgreementHandler_Accept_NotPending(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	req := httptest.NewRequest(http.MethodPost, "/api/agreements/1/accept", nil)
	req.Header.Set("X-User-ID", "1")
	req.SetPathValue("id", "1")

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "lender_id", "borrower_id", "post_id",
		"principal_amount", "interest_rate", "total_amount", "currency",
		"created_at", "accepted_at", "disbursed_at", "start_date", "due_date", "completed_at",
		"payment_frequency", "number_of_payments",
		"status", "contract_url", "contract_hash",
	}).AddRow(
		1, 1, 2, 10,
		1000.0, 0.1, 1100.0, "KZT",
		now, &now, nil, &now, now.AddDate(0, 1, 0), nil,
		"one_time", 1,
		"active", nil, nil,
	)

	mock.ExpectQuery(`SELECT \* FROM agreements WHERE id=\$1`).
		WithArgs("1").
		WillReturnRows(rows)

	rec := httptest.NewRecorder()
	h.Accept(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "can only accept pending agreements")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAgreementHandler_Accept_Success(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	req := httptest.NewRequest(http.MethodPost, "/api/agreements/1/accept", nil)
	req.Header.Set("X-User-ID", "1")
	req.SetPathValue("id", "1")

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "lender_id", "borrower_id", "post_id",
		"principal_amount", "interest_rate", "total_amount", "currency",
		"created_at", "accepted_at", "disbursed_at", "start_date", "due_date", "completed_at",
		"payment_frequency", "number_of_payments",
		"status", "contract_url", "contract_hash",
	}).AddRow(
		1, 1, 2, 10,
		1000.0, 0.1, 1100.0, "KZT",
		now, nil, nil, nil, now.AddDate(0, 1, 0), nil,
		"one_time", 1,
		"pending", nil, nil,
	)

	mock.ExpectQuery(`SELECT \* FROM agreements WHERE id=\$1`).
		WithArgs("1").
		WillReturnRows(rows)

	mock.ExpectExec(`UPDATE agreements SET status = 'active', accepted_at = \$1, start_date = \$2 WHERE id = \$3`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	rec := httptest.NewRecorder()
	h.Accept(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var agreement models.Agreement
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&agreement))
	require.Equal(t, "active", agreement.Status)
	require.NotNil(t, agreement.AcceptedAt)

	require.NoError(t, mock.ExpectationsWereMet())
}

// Cancel
func TestAgreementHandler_Cancel_NotFound(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	req := httptest.NewRequest(http.MethodPost, "/api/agreements/999/cancel", nil)
	req.Header.Set("X-User-ID", "1")
	req.SetPathValue("id", "999")

	mock.ExpectQuery(`SELECT \* FROM agreements WHERE id=\$1`).
		WithArgs("999").
		WillReturnError(sql.ErrNoRows)

	rec := httptest.NewRecorder()
	h.Cancel(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAgreementHandler_Cancel_NotAuthorized(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	req := httptest.NewRequest(http.MethodPost, "/api/agreements/1/cancel", nil)
	req.Header.Set("X-User-ID", "3")
	req.SetPathValue("id", "1")

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "lender_id", "borrower_id", "post_id",
		"principal_amount", "interest_rate", "total_amount", "currency",
		"created_at", "accepted_at", "disbursed_at", "start_date", "due_date", "completed_at",
		"payment_frequency", "number_of_payments",
		"status", "contract_url", "contract_hash",
	}).AddRow(
		1, 1, 2, 10,
		1000.0, 0.1, 1100.0, "KZT",
		now, nil, nil, nil, now.AddDate(0, 1, 0), nil,
		"one_time", 1,
		"pending", nil, nil,
	)

	mock.ExpectQuery(`SELECT \* FROM agreements WHERE id=\$1`).
		WithArgs("1").
		WillReturnRows(rows)

	rec := httptest.NewRecorder()
	h.Cancel(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "not authorized")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAgreementHandler_Cancel_Success(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	req := httptest.NewRequest(http.MethodPost, "/api/agreements/1/cancel", nil)
	req.Header.Set("X-User-ID", "1")
	req.SetPathValue("id", "1")

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "lender_id", "borrower_id", "post_id",
		"principal_amount", "interest_rate", "total_amount", "currency",
		"created_at", "accepted_at", "disbursed_at", "start_date", "due_date", "completed_at",
		"payment_frequency", "number_of_payments",
		"status", "contract_url", "contract_hash",
	}).AddRow(
		1, 1, 2, 10,
		1000.0, 0.1, 1100.0, "KZT",
		now, nil, nil, nil, now.AddDate(0, 1, 0), nil,
		"one_time", 1,
		"pending", nil, nil,
	)

	mock.ExpectQuery(`SELECT \* FROM agreements WHERE id=\$1`).
		WithArgs("1").
		WillReturnRows(rows)

	mock.ExpectExec(`UPDATE agreements SET status = 'cancelled' WHERE id = \$1`).
		WithArgs("1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	rec := httptest.NewRecorder()
	h.Cancel(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var agreement models.Agreement
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&agreement))
	require.Equal(t, "cancelled", agreement.Status)

	require.NoError(t, mock.ExpectationsWereMet())
}

// UpdateContract
func TestAgreementHandler_UpdateContract_BadJSON(t *testing.T) {
	db, _ := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	req := httptest.NewRequest(http.MethodPut, "/api/agreements/1/contract", strings.NewReader(`{`))
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	h.UpdateContract(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "invalid json")
}

func TestAgreementHandler_UpdateContract_MissingURL(t *testing.T) {
	db, _ := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	body := `{"contract_hash": "abc123"}`
	req := httptest.NewRequest(http.MethodPut, "/api/agreements/1/contract", strings.NewReader(body))
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	h.UpdateContract(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "contract_url is required")
}

func TestAgreementHandler_UpdateContract_NotLender(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	body := `{"contract_url": "https://example.com/contract.pdf", "contract_hash": "abc123"}`
	req := httptest.NewRequest(http.MethodPut, "/api/agreements/1/contract", strings.NewReader(body))
	req.Header.Set("X-User-ID", "2")
	req.SetPathValue("id", "1")

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "lender_id", "borrower_id", "post_id",
		"principal_amount", "interest_rate", "total_amount", "currency",
		"created_at", "accepted_at", "disbursed_at", "start_date", "due_date", "completed_at",
		"payment_frequency", "number_of_payments",
		"status", "contract_url", "contract_hash",
	}).AddRow(
		1, 1, 2, 10,
		1000.0, 0.1, 1100.0, "KZT",
		now, &now, nil, &now, now.AddDate(0, 1, 0), nil,
		"one_time", 1,
		"active", nil, nil,
	)

	mock.ExpectQuery(`SELECT \* FROM agreements WHERE id=\$1`).
		WithArgs("1").
		WillReturnRows(rows)

	rec := httptest.NewRecorder()
	h.UpdateContract(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "only lender can update contract")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAgreementHandler_UpdateContract_Success(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewAgreementHandler(db)

	body := `{"contract_url": "https://example.com/contract.pdf", "contract_hash": "abc123"}`
	req := httptest.NewRequest(http.MethodPut, "/api/agreements/1/contract", strings.NewReader(body))
	req.Header.Set("X-User-ID", "1")
	req.SetPathValue("id", "1")

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "lender_id", "borrower_id", "post_id",
		"principal_amount", "interest_rate", "total_amount", "currency",
		"created_at", "accepted_at", "disbursed_at", "start_date", "due_date", "completed_at",
		"payment_frequency", "number_of_payments",
		"status", "contract_url", "contract_hash",
	}).AddRow(
		1, 1, 2, 10,
		1000.0, 0.1, 1100.0, "KZT",
		now, &now, nil, &now, now.AddDate(0, 1, 0), nil,
		"one_time", 1,
		"active", nil, nil,
	)

	mock.ExpectQuery(`SELECT \* FROM agreements WHERE id=\$1`).
		WithArgs("1").
		WillReturnRows(rows)

	mock.ExpectExec(`UPDATE agreements SET contract_url = \$1, contract_hash = \$2 WHERE id = \$3`).
		WithArgs("https://example.com/contract.pdf", "abc123", "1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	rec := httptest.NewRecorder()
	h.UpdateContract(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var agreement models.Agreement
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&agreement))
	require.NotNil(t, agreement.ContractURL)
	require.Equal(t, "https://example.com/contract.pdf", *agreement.ContractURL)
	require.NotNil(t, agreement.ContractHash)
	require.Equal(t, "abc123", *agreement.ContractHash)

	require.NoError(t, mock.ExpectationsWereMet())
}
