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

// Get All
func TestPostHandler_GetAll_Success(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewPostHandler(db)

	rows := sqlmock.NewRows([]string{"id", "title", "content", "author_id", "created_at"}).
		AddRow(1, "Hello", "World", 10, time.Now())

	mock.ExpectQuery(`SELECT id, title, content, author_id, created_at FROM posts`).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	rec := httptest.NewRecorder()

	h.GetAll(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var posts []models.Post
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&posts))
	require.Len(t, posts, 1)
	require.Equal(t, "Hello", posts[0].Title)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostHandler_GetAll_DBError(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewPostHandler(db)

	mock.ExpectQuery(`SELECT id, title, content, author_id, created_at FROM posts`).
		WillReturnError(sql.ErrConnDone)

	req := httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	rec := httptest.NewRecorder()

	h.GetAll(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Body.String(), "error")

	require.NoError(t, mock.ExpectationsWereMet())
}

// Create

func TestPostHandler_Create_BadJSON(t *testing.T) {
	db, _ := utils.NewSQLXMock(t)
	h := NewPostHandler(db)

	req := httptest.NewRequest(http.MethodPost, "/api/posts", strings.NewReader(`{`)) // invalid JSON
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "error")
}

func TestPostHandler_Create_Success(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewPostHandler(db)

	body := `{"title": "Test", "content": "Content"}`
	req := httptest.NewRequest(http.MethodPost, "/api/posts", strings.NewReader(body))
	req.Header.Set("X-User-ID", "5")

	rows := sqlmock.NewRows([]string{"id", "created_at"}).
		AddRow(10, time.Now())

	mock.ExpectQuery(`INSERT INTO posts`).
		WithArgs("Test", "Content", int64(5)).
		WillReturnRows(rows)

	rec := httptest.NewRecorder()
	h.Create(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var p models.Post
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&p))
	require.Equal(t, int64(10), p.ID)
	require.Equal(t, int64(5), p.AuthorID)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostHandler_Create_DBError(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewPostHandler(db)

	body := `{"title": "Fail", "content": "Ops"}`
	req := httptest.NewRequest(http.MethodPost, "/api/posts", strings.NewReader(body))
	req.Header.Set("X-User-ID", "1")

	mock.ExpectQuery(`INSERT INTO posts`).
		WillReturnError(sql.ErrTxDone)

	rec := httptest.NewRecorder()
	h.Create(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Body.String(), "error")

	require.NoError(t, mock.ExpectationsWereMet())
}

// Update
func TestPostHandler_Update_NotFound(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewPostHandler(db)

	req := httptest.NewRequest(http.MethodPut, "/api/posts/999", strings.NewReader(`{"title":"x"}`))
	req.SetPathValue("id", "999")

	mock.ExpectQuery(`SELECT author_id FROM posts WHERE id=\$1`).
		WithArgs("999").
		WillReturnError(sql.ErrNoRows)

	rec := httptest.NewRecorder()
	h.Update(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPostHandler_Update_Forbidden(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewPostHandler(db)

	req := httptest.NewRequest(http.MethodPut, "/api/posts/10", strings.NewReader(`{"title":"x","content":"y"}`))
	req.Header.Set("X-User-ID", "2") // acting user
	req.SetPathValue("id", "10")

	// author is user 1, acting user is 2 → forbidden
	mock.ExpectQuery(`SELECT author_id FROM posts`).
		WillReturnRows(sqlmock.NewRows([]string{"author_id"}).AddRow(1))

	rec := httptest.NewRecorder()
	h.Update(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "not allowed")
}

func TestPostHandler_Update_Success(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewPostHandler(db)

	req := httptest.NewRequest(http.MethodPut, "/api/posts/10", strings.NewReader(`{"title":"new","content":"updated"}`))
	req.Header.Set("X-User-ID", "3")
	req.SetPathValue("id", "10")

	mock.ExpectQuery(`SELECT author_id FROM posts WHERE id=`).
		WillReturnRows(sqlmock.NewRows([]string{"author_id"}).AddRow(3))

	mock.ExpectExec(`UPDATE posts`).
		WithArgs("new", "updated", "10").
		WillReturnResult(sqlmock.NewResult(0, 1))

	rec := httptest.NewRecorder()
	h.Update(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var p models.Post
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&p))
	require.Equal(t, "new", p.Title)

	require.NoError(t, mock.ExpectationsWereMet())
}

// Delete
func TestPostHandler_Delete_NotFound(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewPostHandler(db)

	req := httptest.NewRequest(http.MethodDelete, "/api/posts/20", nil)
	req.SetPathValue("id", "20")

	mock.ExpectQuery(`SELECT author_id FROM posts`).
		WillReturnError(sql.ErrNoRows)

	rec := httptest.NewRecorder()
	h.Delete(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPostHandler_Delete_Forbidden(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewPostHandler(db)

	req := httptest.NewRequest(http.MethodDelete, "/api/posts/12", nil)
	req.Header.Set("X-User-ID", "9")
	req.SetPathValue("id", "12")

	mock.ExpectQuery(`SELECT author_id FROM posts`).
		WillReturnRows(sqlmock.NewRows([]string{"author_id"}).AddRow(5))

	rec := httptest.NewRecorder()
	h.Delete(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestPostHandler_Delete_Success(t *testing.T) {
	db, mock := utils.NewSQLXMock(t)
	h := NewPostHandler(db)

	req := httptest.NewRequest(http.MethodDelete, "/api/posts/7", nil)
	req.Header.Set("X-User-ID", "3")
	req.SetPathValue("id", "7")

	mock.ExpectQuery(`SELECT author_id FROM posts`).
		WillReturnRows(sqlmock.NewRows([]string{"author_id"}).AddRow(3))

	mock.ExpectExec(`DELETE FROM posts`).
		WithArgs("7").
		WillReturnResult(sqlmock.NewResult(0, 1))

	rec := httptest.NewRecorder()
	h.Delete(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)

	require.NoError(t, mock.ExpectationsWereMet())
}
