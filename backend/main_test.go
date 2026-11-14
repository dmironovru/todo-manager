package main

// Регрессионные тесты для API задач.
// Запуск: DATABASE_URL='postgres://todo_user:todo_pass@localhost:5432/todo_db' go test -v ./...
// Требуется живая PostgreSQL (таблица создаётся автоматически).

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// dbSkip непустой, если PostgreSQL недоступна: тесты пропускаются, а не падают,
// чтобы `go test ./...` на машине без БД не ломал сборку.
var dbSkip string

func TestMain(m *testing.M) {
	conn := os.Getenv("DATABASE_URL")
	if conn == "" {
		conn = "postgres://todo_user:todo_pass@localhost:5432/todo_db"
	}
	pool, err := pgxpool.New(context.Background(), conn)
	if err != nil {
		dbSkip = "PostgreSQL недоступна: " + err.Error()
	} else if err := pool.Ping(context.Background()); err != nil {
		dbSkip = "PostgreSQL не отвечает: " + err.Error()
		pool.Close()
	} else {
		db = pool
		defer db.Close()
		initDB()
	}
	if dbSkip != "" {
		log.Println("SKIP:", dbSkip, "— задайте DATABASE_URL, чтобы прогнать тесты API")
	}
	os.Exit(m.Run())
}

func reset(t *testing.T) {
	t.Helper()
	if dbSkip != "" {
		t.Skip(dbSkip)
	}
	if _, err := db.Exec(context.Background(), "TRUNCATE tasks RESTART IDENTITY"); err != nil {
		t.Fatal(err)
	}
}

// call дёргает хендлер так же, как это делает роутер из main().
func call(h http.HandlerFunc, method, path, id, body string) *httptest.ResponseRecorder {
	var rdr *bytes.Reader
	if body == "" {
		rdr = bytes.NewReader(nil)
	} else {
		rdr = bytes.NewReader([]byte(body))
	}
	r := httptest.NewRequest(method, path, rdr)
	r.Header.Set("Content-Type", "application/json")
	if id != "" {
		r.SetPathValue("id", id)
	}
	w := httptest.NewRecorder()
	h(w, r)
	return w
}

func mustCreate(t *testing.T, title string) Task {
	t.Helper()
	w := call(createTask, "POST", "/api/tasks", "", `{"title":"`+title+`"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("создание задачи: код %d, тело %s", w.Code, w.Body.String())
	}
	var task Task
	if err := json.Unmarshal(w.Body.Bytes(), &task); err != nil {
		t.Fatal(err)
	}
	return task
}

func getTasksList(t *testing.T) []Task {
	t.Helper()
	w := call(getTasks, "GET", "/api/tasks", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("список задач: код %d", w.Code)
	}
	var list []Task
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	return list
}

func find(t *testing.T, id int) Task {
	t.Helper()
	for _, task := range getTasksList(t) {
		if task.ID == id {
			return task
		}
	}
	t.Fatalf("задача %d не найдена в списке", id)
	return Task{}
}

// БАГ 1: фронтенд при переименовании шлёт только {title}.
// Бэкенд обязан сохранить completed, а не сбрасывать его в false.
func TestRenameKeepsCompleted(t *testing.T) {
	reset(t)
	task := mustCreate(t, "Купить молоко")

	if w := call(updateTask, "PUT", "/api/tasks/1", itoa(task.ID), `{"completed":true}`); w.Code != http.StatusOK {
		t.Fatalf("отметка выполненной: код %d", w.Code)
	}
	if w := call(updateTask, "PUT", "/api/tasks/1", itoa(task.ID), `{"title":"Купить молоко и хлеб"}`); w.Code != http.StatusOK {
		t.Fatalf("переименование: код %d", w.Code)
	}

	got := find(t, task.ID)
	if got.Title != "Купить молоко и хлеб" {
		t.Errorf("название = %q, хочу %q", got.Title, "Купить молоко и хлеб")
	}
	if !got.Completed {
		t.Errorf("completed = false после переименования, хочу true (галочка слетела)")
	}
}

// Контроль: обратный порядок — снятие галочки не должно менять название.
func TestToggleKeepsTitle(t *testing.T) {
	reset(t)
	task := mustCreate(t, "Позвонить в банк")
	call(updateTask, "PUT", "/api/tasks/1", itoa(task.ID), `{"completed":true}`)
	call(updateTask, "PUT", "/api/tasks/1", itoa(task.ID), `{"completed":false}`)

	got := find(t, task.ID)
	if got.Title != "Позвонить в банк" || got.Completed {
		t.Errorf("получили %+v, хочу title без изменений и completed=false", got)
	}
}

// БАГ 2: DELETE несуществующей задачи должен быть 404, а не 204.
func TestDeleteMissingTask(t *testing.T) {
	reset(t)
	if w := call(deleteTask, "DELETE", "/api/tasks/999999", "999999", ""); w.Code != http.StatusNotFound {
		t.Errorf("DELETE отсутствующей задачи = %d, хочу 404", w.Code)
	}
}

// БАГ 3: PUT несуществующей задачи должен быть 404, а не 200.
func TestUpdateMissingTask(t *testing.T) {
	reset(t)
	if w := call(updateTask, "PUT", "/api/tasks/999999", "999999", `{"title":"призрак"}`); w.Code != http.StatusNotFound {
		t.Errorf("PUT отсутствующей задачи = %d, хочу 404", w.Code)
	}
}

// БАГ 4: название из одних пробелов не должно создаваться.
func TestCreateRejectsBlankTitle(t *testing.T) {
	reset(t)
	if w := call(createTask, "POST", "/api/tasks", "", `{"title":"   "}`); w.Code != http.StatusBadRequest {
		t.Errorf("POST с пробельным title = %d, хочу 400", w.Code)
	}
	if n := len(getTasksList(t)); n != 0 {
		t.Errorf("создалось %d задач, хочу 0", n)
	}
}

// БАГ 5: пустое название при редактировании — явная ошибка, а не тихий no-op.
func TestUpdateRejectsEmptyTitle(t *testing.T) {
	reset(t)
	task := mustCreate(t, "Задача")
	if w := call(updateTask, "PUT", "/api/tasks/1", itoa(task.ID), `{"title":""}`); w.Code != http.StatusBadRequest {
		t.Errorf("PUT с пустым title = %d, хочу 400", w.Code)
	}
	if got := find(t, task.ID); got.Title != "Задача" {
		t.Errorf("название изменилось на %q", got.Title)
	}
}

// Здоровье API: пробелы вокруг названия обрезаются, пустой список — [] а не null.
func TestTrimAndEmptyList(t *testing.T) {
	reset(t)
	if w := call(getTasks, "GET", "/api/tasks", "", ""); bytes.TrimSpace(w.Body.Bytes())[0] != '[' {
		t.Errorf("пустой список = %s, хочу []", w.Body.String())
	}
	task := mustCreate(t, "  с пробелами  ")
	if got := find(t, task.ID); got.Title != "с пробелами" {
		t.Errorf("title = %q, хочу обрезанный вариант", got.Title)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
