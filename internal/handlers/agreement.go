package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/railanbaigazy/uade-api/internal/app/models"
	"github.com/railanbaigazy/uade-api/internal/contracts"
	"github.com/railanbaigazy/uade-api/internal/utils"
)

type AgreementHandler struct {
	DB                *sqlx.DB
	ContractGenerator *contracts.Generator
}

func NewAgreementHandler(db *sqlx.DB) *AgreementHandler {
	return &AgreementHandler{DB: db, ContractGenerator: contracts.NewGenerator("contracts")}
}

func (h *AgreementHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input struct {
		PostID           int     `json:"post_id"`
		PrincipalAmount  float64 `json:"principal_amount"`
		InterestRate     float64 `json:"interest_rate"`
		DueDate          string  `json:"due_date"`
		PaymentFrequency string  `json:"payment_frequency"`
		NumberOfPayments int     `json:"number_of_payments"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.WriteJSONError(w, "invalid json", http.StatusBadRequest)
		return
	}

	if input.PostID <= 0 {
		utils.WriteJSONError(w, "post_id is required", http.StatusBadRequest)
		return
	}
	if input.PrincipalAmount <= 0 {
		utils.WriteJSONError(w, "principal_amount must be greater than 0", http.StatusBadRequest)
		return
	}
	if input.InterestRate < 0 {
		utils.WriteJSONError(w, "interest_rate cannot be negative", http.StatusBadRequest)
		return
	}
	if input.DueDate == "" {
		utils.WriteJSONError(w, "due_date is required", http.StatusBadRequest)
		return
	}
	if input.NumberOfPayments <= 0 {
		utils.WriteJSONError(w, "number_of_payments must be greater than 0", http.StatusBadRequest)
		return
	}

	validFrequencies := map[string]bool{
		"one_time": true,
		"weekly":   true,
		"biweekly": true,
		"monthly":  true,
	}
	if !validFrequencies[input.PaymentFrequency] {
		utils.WriteJSONError(w, "invalid payment_frequency", http.StatusBadRequest)
		return
	}

	dueDate, err := time.Parse("2006-01-02", input.DueDate)
	if err != nil {
		utils.WriteJSONError(w, "invalid due_date format, use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	if dueDate.Before(time.Now()) {
		utils.WriteJSONError(w, "due_date must be in the future", http.StatusBadRequest)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	borrowerID, _ := strconv.ParseInt(userIDStr, 10, 64)

	var post struct {
		AuthorID int    `db:"author_id"`
		Type     string `db:"type"`
	}
	err = h.DB.Get(&post, "SELECT author_id, type FROM posts WHERE id=$1", input.PostID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteJSONError(w, "post not found", http.StatusNotFound)
			return
		}

		utils.WriteJSONError(w, "failed to fetch post", http.StatusInternalServerError)
		return
	}

	if post.Type != "lend" {
		utils.WriteJSONError(w, "can only create agreements for lend posts", http.StatusBadRequest)
		return
	}

	if int64(post.AuthorID) == borrowerID {
		utils.WriteJSONError(w, "cannot create agreement with your own post", http.StatusBadRequest)
		return
	}

	lenderID := post.AuthorID
	totalAmount := input.PrincipalAmount * (1 + input.InterestRate)

	var agreement models.Agreement
	query := `
		INSERT INTO agreements (
			lender_id, borrower_id, post_id,
			principal_amount, interest_rate, total_amount, currency,
			due_date, payment_frequency, number_of_payments,
			status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		RETURNING id, created_at
	`

	err = h.DB.Get(&agreement, query,
		lenderID, borrowerID, input.PostID,
		input.PrincipalAmount, input.InterestRate, totalAmount, "KZT",
		dueDate, input.PaymentFrequency, input.NumberOfPayments,
		"pending",
	)
	if err != nil {
		utils.WriteJSONError(w, "failed to create agreement", http.StatusInternalServerError)
		return
	}

	agreement.LenderID = int64(lenderID)
	agreement.BorrowerID = borrowerID
	agreement.PostID = input.PostID
	agreement.PrincipalAmount = input.PrincipalAmount
	agreement.InterestRate = input.InterestRate
	agreement.TotalAmount = totalAmount
	agreement.Currency = "KZT"
	agreement.DueDate = dueDate
	agreement.PaymentFrequency = input.PaymentFrequency
	agreement.NumberOfPayments = input.NumberOfPayments
	agreement.Status = "pending"

	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(agreement); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func (h *AgreementHandler) GetUserAgreements(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	statusFilter := r.URL.Query().Get("status")
	roleFilter := r.URL.Query().Get("role")

	query := `
		SELECT 
			id, lender_id, borrower_id, post_id,
			principal_amount, interest_rate, total_amount, currency,
			created_at, accepted_at, disbursed_at, start_date, due_date, completed_at,
			payment_frequency, number_of_payments,
			status, contract_url, contract_hash
		FROM agreements
		WHERE (lender_id = $1 OR borrower_id = $1)
	`

	args := []any{userID}
	argCount := 1

	if statusFilter != "" {
		argCount++
		query += " AND status = $" + strconv.Itoa(argCount)
		args = append(args, statusFilter)
	}

	switch roleFilter {
	case "lender":
		query += " AND lender_id = $1"
	case "borrower":
		query += " AND borrower_id = $1"
	}

	query += " ORDER BY created_at DESC"

	agreements := make([]models.Agreement, 0)
	if err := h.DB.Select(&agreements, query, args...); err != nil {
		utils.WriteJSONError(w, "failed to fetch agreements", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(agreements); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func (h *AgreementHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userIDStr := r.Header.Get("X-User-ID")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	var agreement models.Agreement
	query := `
		SELECT 
			id, lender_id, borrower_id, post_id,
			principal_amount, interest_rate, total_amount, currency,
			created_at, accepted_at, disbursed_at, start_date, due_date, completed_at,
			payment_frequency, number_of_payments,
			status, contract_url, contract_hash
		FROM agreements
		WHERE id = $1
	`

	err := h.DB.Get(&agreement, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteJSONError(w, "agreement not found", http.StatusNotFound)
			return
		}

		utils.WriteJSONError(w, "failed to fetch agreement", http.StatusInternalServerError)
		return
	}

	if agreement.LenderID != userID && agreement.BorrowerID != userID {
		utils.WriteJSONError(w, "not authorized to view this agreement", http.StatusForbidden)
		return
	}

	if err := json.NewEncoder(w).Encode(agreement); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func (h *AgreementHandler) Accept(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userIDStr := r.Header.Get("X-User-ID")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	var agreement models.Agreement
	err := h.DB.Get(&agreement, "SELECT * FROM agreements WHERE id=$1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteJSONError(w, "agreement not found", http.StatusNotFound)
			return
		}
		utils.WriteJSONError(w, "failed to fetch agreement", http.StatusInternalServerError)
		return
	}

	if agreement.LenderID != userID {
		utils.WriteJSONError(w, "only lender can accept the agreement", http.StatusForbidden)
		return
	}

	if agreement.Status != "pending" {
		utils.WriteJSONError(w, "can only accept pending agreements", http.StatusBadRequest)
		return
	}

	now := time.Now()
	_, err = h.DB.Exec(`
		UPDATE agreements 
		SET status = 'active', accepted_at = $1, start_date = $2
		WHERE id = $3
	`, now, now, id)
	if err != nil {
		utils.WriteJSONError(w, "failed to accept agreement", http.StatusInternalServerError)
		return
	}

	agreement.Status = "active"
	agreement.AcceptedAt = &now
	startDate := now
	agreement.StartDate = &startDate

	if err := json.NewEncoder(w).Encode(agreement); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func (h *AgreementHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userIDStr := r.Header.Get("X-User-ID")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	var agreement models.Agreement
	err := h.DB.Get(&agreement, "SELECT * FROM agreements WHERE id=$1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteJSONError(w, "agreement not found", http.StatusNotFound)
			return
		}
		utils.WriteJSONError(w, "failed to fetch agreement", http.StatusInternalServerError)
		return
	}

	if agreement.LenderID != userID && agreement.BorrowerID != userID {
		utils.WriteJSONError(w, "not authorized to cancel this agreement", http.StatusForbidden)
		return
	}

	if agreement.Status != "pending" {
		utils.WriteJSONError(w, "can only cancel pending agreements", http.StatusBadRequest)
		return
	}

	_, err = h.DB.Exec("UPDATE agreements SET status = 'cancelled' WHERE id = $1", id)
	if err != nil {
		utils.WriteJSONError(w, "failed to cancel agreement", http.StatusInternalServerError)
		return
	}

	agreement.Status = "cancelled"

	if err := json.NewEncoder(w).Encode(agreement); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func (h *AgreementHandler) UpdateContract(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userIDStr := r.Header.Get("X-User-ID")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	var input struct {
		ContractURL  string `json:"contract_url"`
		ContractHash string `json:"contract_hash"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.WriteJSONError(w, "invalid json", http.StatusBadRequest)
		return
	}

	input.ContractURL = strings.TrimSpace(input.ContractURL)
	input.ContractHash = strings.TrimSpace(input.ContractHash)

	if input.ContractURL == "" {
		utils.WriteJSONError(w, "contract_url is required", http.StatusBadRequest)
		return
	}

	var agreement models.Agreement
	err := h.DB.Get(&agreement, "SELECT * FROM agreements WHERE id=$1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteJSONError(w, "agreement not found", http.StatusNotFound)
			return
		}
		utils.WriteJSONError(w, "failed to fetch agreement", http.StatusInternalServerError)
		return
	}

	if agreement.LenderID != userID {
		utils.WriteJSONError(w, "only lender can update contract", http.StatusForbidden)
		return
	}

	_, err = h.DB.Exec(`
		UPDATE agreements 
		SET contract_url = $1, contract_hash = $2
		WHERE id = $3
	`, input.ContractURL, input.ContractHash, id)
	if err != nil {
		utils.WriteJSONError(w, "failed to update contract", http.StatusInternalServerError)
		return
	}

	agreement.ContractURL = &input.ContractURL
	agreement.ContractHash = &input.ContractHash

	if err := json.NewEncoder(w).Encode(agreement); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func (h *AgreementHandler) GenerateContract(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userIDStr := r.Header.Get("X-User-ID")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	var agreement models.Agreement
	err := h.DB.Get(&agreement, "SELECT * FROM agreements WHERE id=$1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteJSONError(w, "agreement not found", http.StatusNotFound)
			return
		}
		utils.WriteJSONError(w, "failed to fetch agreement", http.StatusInternalServerError)
		return
	}

	if agreement.LenderID != userID && agreement.BorrowerID != userID {
		utils.WriteJSONError(w, "not authorized to generate contract for this agreement", http.StatusForbidden)
		return
	}

	if agreement.Status != "active" {
		utils.WriteJSONError(w, "contract generation is available only for active agreements", http.StatusBadRequest)
		return
	}

	gen := h.ContractGenerator
	if gen == nil {
		gen = contracts.NewGenerator("contracts")
	}

	contractURL, contractHash, err := gen.Generate(r.Context(), &agreement)
	if err != nil {
		utils.WriteJSONError(w, "failed to generate contract", http.StatusInternalServerError)
		return
	}

	_, err = h.DB.Exec(`
		UPDATE agreements 
		SET contract_url = $1, contract_hash = $2
		WHERE id = $3
	`, contractURL, contractHash, id)
	if err != nil {
		utils.WriteJSONError(w, "failed to persist contract metadata", http.StatusInternalServerError)
		return
	}

	agreement.ContractURL = &contractURL
	agreement.ContractHash = &contractHash

	if err := json.NewEncoder(w).Encode(agreement); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}
