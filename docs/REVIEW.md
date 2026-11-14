# Разбор репозитория `dmironovru/todo-manager`

**Коммит:** `83f07a8` (main) · **Дата разбора:** 13 ноября 2026
**Стек:** React 19 (CRA 5) + Go 1.22 (net/http + pgx v5) + PostgreSQL 16 + Docker Compose

Всё ниже, кроме явно помеченных пунктов, **воспроизведено вживую**: я склонировал репозиторий, поставил Go 1.27.4 и PostgreSQL 15, собрал бэкенд, запустил его и дёргал ручки HTTP-запросами. Фронтенд собран через `npm ci` и поднят дев-сервером.

---

## Коротко

| Что | Итог |
|---|---|
| Найдено проблем | 15 (2 критичных, 8 средних, 5 мелких) |
| Воспроизведено запросами/командами | 7 |
| Исправлено в патче | 11 |
| Тестов написано | 14: 7 Go + 7 JS (на старом коде падали **6 Go + 1 JS**, теперь все зелёные) |
| Патч проверен на чистом `HEAD` | да: `git apply`, `go test`, `npm test` — всё проходит |

Главное: **галочка «выполнено» слетает, если переименовать задачу**, и **в публичном репозитории лежит пароль от БД**.

---

## Как устроено

```
браузер ──> nginx :80 (в Docker)  ──┐
        └─> CRA dev-proxy :3000  ──┴──> Go API :8080 ──> PostgreSQL
            (proxy в package.json)      5 эндпоинтов      таблица tasks
```

* Фронт ходит только по относительному `/api/tasks` — это правильно, никакого `localhost` в браузере.
* Схема создаётся на старте через `CREATE TABLE IF NOT EXISTS`, миграций нет.
* Авторизации, пагинации, лимитов нет — для демо нормально, для продакшена нет.
* `start.sh` проверяет Docker и поднимает `docker compose up -d`, либо fallback на `go run` + `npm start`.

---

## Критичное

### 1. Галочка «выполнено» слетает при переименовании — воспроизведено

**Что видит пользователь:** отметил задачу выполненной → переименовал двойным кликом → галочка пропала.

**Причина.** Фронтенд при переименовании шлёт только название (`App.js`, `saveEdit`):

```js
await axios.put(`${API_BASE}/${id}`, { title: editText });
```

А бэкенд принимает структуру с обычным `bool`, который для отсутствующего поля становится `false`, и **всегда** перезаписывает статус:

```go
var req struct {
    Title     string `json:"title"`
    Completed bool   `json:"completed"`   // не прислали → false
}
...
db.Exec(ctx, "UPDATE tasks SET completed=$1 WHERE id=$2", req.Completed, id)
```

**Живое воспроизведение** (`review/repro.py`):

```
1. Создали:            id=1  completed=False  title='Купить молоко'
2. PUT {completed:true} id=1  completed=True   title='Купить молоко'
3. PUT {title:'...'}    id=1  completed=False  title='Купить молоко и хлеб'  ← БАГ
```

**Фикс:** указатели `*string` / `*bool`, чтобы отличать «поле не прислали» от «прислали ноль», и один атомарный запрос:

```go
UPDATE tasks
   SET title = COALESCE($1, title), completed = COALESCE($2, completed)
 WHERE id = $3
RETURNING id, title, completed
```

### 2. Пароль от БД в публичном коде

В `backend/main.go` был fallback со строкой подключения и паролем `todo123321odot`. Репозиторий публичный, пароль виден всем и **остаётся в истории git** даже после удаления.

Что сделать:
1. Если этот пароль где-то реально используется — **сменить** (удаление из кода не помогает).
2. В патче fallback убран: без `DATABASE_URL` сервер падает с понятной подсказкой.
3. Пароли в `docker-compose.yml` (`todo_pass`) для локального демо допустимы, но лучше вынести в `.env` и добавить его в `.gitignore`.

---

## Среднее (всё исправлено в патче)

| № | Проблема | Было | Стало |
|---|---|---|---|
| 3 | `DELETE /api/tasks/999999` — задача не существует | `204` | `404` |
| 4 | `PUT /api/tasks/999999` — задача не существует | `200` | `404` |
| 5 | `POST {"title":"   "}` — название из пробелов | `201`, задача создана | `400` |
| 6 | `PUT {"title":""}` — пустое название | `200`, тихий no-op | `400` |
| 7 | PUT делал два `UPDATE` подряд | неатомарно, состояние может разъехаться | один запрос |
| 8 | `rows.Scan()` и `rows.Err()` игнорировались | битые данные молча уходили в JSON | ошибка → 500 + лог |
| 9 | `clearTasks` отдавал клиенту `err.Error()` | утечка внутренностей БД | generic-сообщение + лог |
| 10 | `http.ListenAndServe` без таймаутов | Slowloris держит соединения | таймауты 5/15/15/60 с |

Плюс: название обрезается по пробелам и ограничено 500 символами (колонка `TEXT` без лимита — способ залить мегабайты).

---

## Мелкое (не правил, оставляю на ваше решение)

**11. Битые сабмодули — воспроизведено.** В индексе два gitlink-объекта (mode `160000`) без `.gitmodules`:

```
160000 6a3f104  backend/go-todo
160000 ab5fcd6  frontend/todo-frontend
```

Каталоги пустые, а `git submodule update --init` падает:
`fatal: No url found for submodule path 'backend/go-todo' in .gitmodules`.
Лечится: `git rm --cached backend/go-todo frontend/todo-frontend`.

**12. Скомпилированный бинарник в git.** `backend/todo-api` — 14.6 МБ ELF, весит больше, чем весь остальной репозиторий (`.git` = 7.1 МБ). Рядом `backend/go.mod.backup`. Удалить, добавить в `.gitignore`; чтобы вычистить из истории — `git filter-repo`.

**13. Единственный фронтенд-тест падал — воспроизведено и исправлено.** `App.test.js` был заглушкой из CRA и искал текст «learn react», которого в приложении нет (`Tests: 1 failed, 1 total`). Заменил на 7 осмысленных тестов с моком axios: загрузка списка, добавление, пустое название, переименование, чекбокс, удаление, ошибка сети. Отдельно зафиксирован контракт «переименование шлёт только `title`» — чтобы баг №1 не вернулся с другой стороны.

**14. `REACT_APP_API_URL` в compose — мёртвая настройка.** CRA вшивает `REACT_APP_*` **на этапе сборки**, а переменная задана в `environment` рантайма — она ни на что не влияет. Работает всё за счёт nginx-прокси, но настройка вводит в заблуждение: убрать или перенести в `build.args`.

**15. Healthcheck бэкенда в compose закомментирован.** Не блокер: `HEALTHCHECK` есть в `backend/Dockerfile`, поэтому `depends_on: service_healthy` работает. Но интервал 30 с без `start_period` — фронтенд может ждать старта до полуминуты. Стоит вернуть блок в compose с `interval: 5s`, `start_period: 5s`. Также `version: '3.8'` в compose устарел и вызывает предупреждение.

**Ещё по мелочи:** CORS `*` без ограничений; нет лимита тела запроса (`http.MaxBytesReader`); нет graceful shutdown; `created_at` в БД есть, а в API не отдаётся; нет пагинации; на фронте устаревший `onKeyPress` (лучше `onKeyDown`), нет состояний загрузки/пустого списка, а быстрые клики по чекбоксу могут дать гонку ответов (`fetchTasks` после каждой мутации). CRA официально заброшен — при случае переезд на Vite.

---

## Что в патче

`review/todo-manager-fix.patch` — три файла, +378/−25:

* `backend/main.go` — фиксы багов;
* `backend/main_test.go` — 7 тестов API (нужна живая PostgreSQL; **без БД тесты пропускаются, а не падают**, так что `go test ./...` на машине без базы остаётся зелёным);
* `frontend/src/App.test.js` — 7 тестов UI вместо заглушки CRA.

Зависимости не добавлялись, `go.mod` / `go.sum` не тронуты, `gofmt` и `go vet` чистые.

**Изменения контракта API (осознанные):**
* `PUT /api/tasks/{id}` теперь возвращает обновлённую задачу в JSON (раньше — пустое тело);
* `PUT` / `DELETE` по несуществующему id → `404`;
* пустое или пробельное название → `400`;
* **без `DATABASE_URL` сервер больше не стартует** (раньше молча брал зашитый пароль). `start.sh` и compose переменную задают, так что для них ничего не меняется.

Фронтенд править не нужно — баг был на стороне бэкенда.

### Как применить

```bash
git clone https://github.com/dmironovru/todo-manager.git && cd todo-manager
git apply /path/to/todo-manager-fix.patch

cd backend
DATABASE_URL='postgres://user:pass@localhost:5432/todo_db' go test -v ./...
cd ../frontend && npm ci && CI=true npm test
```

### Результат прогонов

```
БЫЛО (текущий main):            СТАЛО (с патчем):
FAIL TestRenameKeepsCompleted   PASS TestRenameKeepsCompleted
PASS TestToggleKeepsTitle       PASS TestToggleKeepsTitle
FAIL TestDeleteMissingTask      PASS TestDeleteMissingTask
FAIL TestUpdateMissingTask      PASS TestUpdateMissingTask
FAIL TestCreateRejectsBlank     PASS TestCreateRejectsBlank
FAIL TestUpdateRejectsEmpty     PASS TestUpdateRejectsEmpty
FAIL TestTrimAndEmptyList       PASS TestTrimAndEmptyList
ok  todo-api  0.027s

Фронтенд:  Tests: 1 failed  →  Tests: 7 passed
```

---

## Чек-лист перед коммитом

Патч закрывает **код**, но не всё в репозитории:

- [x] Баги API и потеря галочки — исправлены, покрыты тестами
- [x] Зашитый пароль убран из кода
- [x] Красный тест фронта заменён на 7 рабочих
- [ ] **Сменить пароль `todo123321odot`**, если он где-то используется: он остаётся в истории git, патч этого не лечит
- [ ] Удалить битые сабмодули: `git rm --cached backend/go-todo frontend/todo-frontend`
- [ ] Удалить бинарник `backend/todo-api` (14.6 МБ) и `backend/go.mod.backup`, добавить в `.gitignore`
- [ ] Убрать мёртвую `REACT_APP_API_URL` из compose и вернуть healthcheck с `interval: 5s`
- [ ] Помнить про breaking change: без `DATABASE_URL` сервер теперь не стартует

Коммит и отправка:

```bash
git apply todo-manager-fix.patch
git add backend/main.go backend/main_test.go frontend/src/App.test.js
git commit -m "fix(api): частичный PUT не сбрасывает completed, 404 на отсутствующие задачи, валидация title"
git push origin main
```

---

## Что я бы сделал дальше

1. **CI на GitHub Actions:** `go vet` + `go test` с сервисным контейнером PostgreSQL, `npm ci && npm test`, сборка Docker-образов. Сейчас красные тесты никого не останавливают.
2. Убрать бинарник и битые сабмодули, добавить `.env.example`.
3. Graceful shutdown, лимит тела запроса, пагинация, `created_at` в ответе.
4. Переезд с CRA на Vite.

---

## Файлы

| Файл | Что это |
|---|---|
| `review/REVIEW.md` | этот разбор |
| `review/todo-manager-fix.patch` | патч: фиксы + тесты, применяется к чистому `HEAD` |
| `review/main_test.go` | тесты Go отдельным файлом |
| `review/App.test.js` | тесты фронта отдельным файлом |
| `review/repro.py` | скрипт, воспроизводящий баги HTTP-запросами |
| `todo-manager/` | клон репозитория с уже применённым патчем |

**Живьём в песочнице:** PostgreSQL 15 → исправленный Go API на `:8080` → React-приложение на `:3000` (открыто в превью). Можно отметить задачу галочкой, переименовать двойным кликом и убедиться, что статус сохраняется. На оригинальном коде этот же сценарий галочку терял.
