package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Task struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

var db *pgxpool.Pool

func main() {
	// Берём строку подключения из переменной окружения.
	// Никаких зашитых паролей в коде: репозиторий публичный.
	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		log.Fatal("DATABASE_URL не задан. Пример: DATABASE_URL='postgres://user:pass@localhost:5432/todo_db' ./todo-server")
	}

	log.Printf("Connecting to database...")

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		log.Fatal("Unable to parse connection string:", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 30 * time.Second

	db, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatal("Unable to create connection pool:", err)
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		log.Fatal("Unable to ping database:", err)
	}

	log.Println("Connected to PostgreSQL (connection pool ready)")

	// Создаём таблицу, если её нет
	initDB()

	// Healthcheck эндпоинт
	http.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("GET /api/tasks", corsMiddleware(getTasks))
	http.HandleFunc("POST /api/tasks", corsMiddleware(createTask))
	http.HandleFunc("PUT /api/tasks/{id}", corsMiddleware(updateTask))
	http.HandleFunc("DELETE /api/tasks/{id}", corsMiddleware(deleteTask))
	http.HandleFunc("DELETE /api/tasks", corsMiddleware(clearTasks))

	log.Println("Server started on :8080")
	srv := &http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

// maxTitleLen — предел длины названия, чтобы TEXT не превращался в способ залить мегабайты.
const maxTitleLen = 500

// normalizeTitle обрезает пробелы и проверяет, что название не пустое и не слишком длинное.
func normalizeTitle(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" || utf8.RuneCountInString(s) > maxTitleLen {
		return "", false
	}
	return s, true
}

// initDB создаёт таблицу tasks, если она не существует
func initDB() {
	query := `
    CREATE TABLE IF NOT EXISTS tasks (
        id SERIAL PRIMARY KEY,
        title TEXT NOT NULL,
        completed BOOLEAN DEFAULT FALSE,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    `
	_, err := db.Exec(context.Background(), query)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
	log.Println("Database schema ready")
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func getTasks(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(context.Background(), "SELECT id, title, completed FROM tasks ORDER BY id")
	if err != nil {
		log.Println("Database error:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tasks := []Task{}
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Completed); err != nil {
			log.Println("Scan error:", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		log.Println("Rows error:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func createTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	title, ok := normalizeTitle(req.Title)
	if !ok {
		http.Error(w, "Invalid title", http.StatusBadRequest)
		return
	}

	var task Task
	err := db.QueryRow(context.Background(),
		"INSERT INTO tasks (title) VALUES ($1) RETURNING id, title, completed",
		title).Scan(&task.ID, &task.Title, &task.Completed)

	if err != nil {
		log.Println("Database error:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func updateTask(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	// Указатели, чтобы отличить «поле не прислали» от «прислали нулевое значение».
	// Раньше Completed был bool и PUT {title: ...} молча сбрасывал галочку в false.
	var req struct {
		Title     *string `json:"title"`
		Completed *bool   `json:"completed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	var title *string
	if req.Title != nil {
		normalized, ok := normalizeTitle(*req.Title)
		if !ok {
			http.Error(w, "Invalid title", http.StatusBadRequest)
			return
		}
		title = &normalized
	}

	// Один атомарный запрос вместо двух: COALESCE оставляет поле без изменений, если пришёл NULL.
	var task Task
	err = db.QueryRow(context.Background(),
		"UPDATE tasks SET title = COALESCE($1, title), completed = COALESCE($2, completed) WHERE id = $3 RETURNING id, title, completed",
		title, req.Completed, id).Scan(&task.ID, &task.Title, &task.Completed)

	if err == pgx.ErrNoRows {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println("Update error:", err)
		http.Error(w, "Failed to update task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(task)
}

func deleteTask(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	tag, err := db.Exec(context.Background(), "DELETE FROM tasks WHERE id=$1", id)
	if err != nil {
		log.Println("Delete error:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if tag.RowsAffected() == 0 {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func clearTasks(w http.ResponseWriter, r *http.Request) {
	_, err := db.Exec(context.Background(), "DELETE FROM tasks")
	if err != nil {
		log.Println("Clear error:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
