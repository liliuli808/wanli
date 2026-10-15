# Kitchen Profile JSONB 500 Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `PUT /api/v1/me/kitchen` persist seasoning and cookware selections as PostgreSQL JSONB arrays without returning 500.

**Architecture:** Keep the existing Android request, API contract, service, repository interface, and database schema. Add GORM's built-in JSON serializer to the two `[]string` model fields, prove their create/update/read behavior against PostgreSQL, then build and deploy the corrected backend on port 8081.

**Tech Stack:** Go 1.22, GORM 1.25, PostgreSQL 15 JSONB, Testify, Gin, Android Kotlin/Gradle

## Global Constraints

- Do not change the REST API request or response shape.
- Do not add a database migration; the existing columns remain `JSONB NOT NULL`.
- Do not modify Android source code; the installed client already targets port `8081`.
- Preserve empty selections as JSON arrays (`[]`), not SQL `NULL`.
- Confirm the exact WanLi listener before stopping the current 8081 process.
- Do not expose JWT secrets, login tokens, or `.env` contents in command output.
- The workspace has an empty `.git` directory, so no commit step is possible.

---

### Task 1: Serialize kitchen string slices as JSONB

**Files:**
- Create: `backend-go/internal/repository/kitchen_repository_integration_test.go`
- Modify: `backend-go/internal/model/kitchen.go:18-19`

**Interfaces:**
- Consumes: `GormKitchenRepository.SaveProfile(ctx context.Context, profile *model.KitchenProfile) error`
- Produces: JSONB-safe `KitchenProfile.OwnedIngredients []string` and `KitchenProfile.OwnedTools []string`

- [ ] **Step 1: Write the failing PostgreSQL regression test**

Create `backend-go/internal/repository/kitchen_repository_integration_test.go`:

```go
package repository

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/wanli/backend-go/internal/model"
)

func TestGormKitchenRepository_SaveProfile_Integration(t *testing.T) {
	dsn := os.Getenv("WANLI_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("未设置 WANLI_TEST_DB_DSN，跳过集成测试")
	}

	db, err := gorm.Open(postgres.Open(dsn))
	require.NoError(t, err)
	repo := NewGormKitchenRepository(db)
	ctx := context.Background()

	user := model.User{
		Email:          "kitchen-" + uuid.NewString() + "@example.com",
		HashedPassword: "x",
		Nickname:       "厨房仓储测试",
	}
	require.NoError(t, db.Create(&user).Error)
	t.Cleanup(func() {
		db.Exec("DELETE FROM users WHERE id = ?", user.ID)
	})

	initial := &model.KitchenProfile{
		UserID:           user.ID,
		OwnedIngredients: []string{"盐", "生抽"},
		OwnedTools:       []string{"炒锅"},
	}
	require.NoError(t, repo.SaveProfile(ctx, initial))

	got, err := repo.GetProfile(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"盐", "生抽"}, got.OwnedIngredients)
	assert.Equal(t, []string{"炒锅"}, got.OwnedTools)

	updated := &model.KitchenProfile{
		UserID:           user.ID,
		OwnedIngredients: []string{},
		OwnedTools:       []string{"烤箱", "空气炸锅"},
	}
	require.NoError(t, repo.SaveProfile(ctx, updated))

	got, err = repo.GetProfile(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, got.OwnedIngredients)
	assert.Empty(t, got.OwnedIngredients)
	assert.Equal(t, []string{"烤箱", "空气炸锅"}, got.OwnedTools)
}
```

- [ ] **Step 2: Run the test and verify RED**

Run from `backend-go/`:

```bash
WANLI_TEST_DB_DSN='host=127.0.0.1 port=5432 user=postgres password=postgres dbname=wanli_db sslmode=disable' \
  go test ./internal/repository -run TestGormKitchenRepository_SaveProfile_Integration -count=1
```

Expected: FAIL at the first `SaveProfile` with PostgreSQL `SQLSTATE 22P02` because GORM writes `[]string` as a row expression instead of JSON.

- [ ] **Step 3: Add the minimal GORM serializer tags**

Change `KitchenProfile` in `backend-go/internal/model/kitchen.go` to:

```go
type KitchenProfile struct {
	UserID           uuid.UUID `gorm:"type:uuid;primary_key"`
	OwnedIngredients []string  `gorm:"type:jsonb;serializer:json"`
	OwnedTools       []string  `gorm:"type:jsonb;serializer:json"`
	UpdatedAt        time.Time
}
```

- [ ] **Step 4: Run the test and verify GREEN**

Run:

```bash
WANLI_TEST_DB_DSN='host=127.0.0.1 port=5432 user=postgres password=postgres dbname=wanli_db sslmode=disable' \
  go test ./internal/repository -run TestGormKitchenRepository_SaveProfile_Integration -count=1
```

Expected: PASS; both create and update round-trip the exact arrays, including the empty ingredient array.

- [ ] **Step 5: Run the complete repository package tests**

Run:

```bash
WANLI_TEST_DB_DSN='host=127.0.0.1 port=5432 user=postgres password=postgres dbname=wanli_db sslmode=disable' \
  go test ./internal/repository -count=1
```

Expected: PASS.

### Task 2: Verify and build the corrected source

**Files:**
- Verify: `backend-go/internal/model/kitchen.go`
- Verify: `backend-go/internal/repository/kitchen_repository_integration_test.go`
- Produce: `/tmp/wanli-api-kitchen-jsonb-20260727`

**Interfaces:**
- Consumes: corrected `KitchenProfile` GORM mapping
- Produces: verified backend binary ready to listen on port `8081`

- [ ] **Step 1: Format and statically check Go**

Run from `backend-go/`:

```bash
go fmt ./...
go vet ./...
```

Expected: both commands exit 0.

- [ ] **Step 2: Run all Go tests**

Run:

```bash
WANLI_TEST_DB_DSN='host=127.0.0.1 port=5432 user=postgres password=postgres dbname=wanli_db sslmode=disable' \
  go test ./... -count=1
```

Expected: every package passes.

- [ ] **Step 3: Build the deployment binary**

Run:

```bash
go build -buildvcs=false -o /tmp/wanli-api-kitchen-jsonb-20260727 ./cmd/api
```

Expected: exit 0 and an executable file at `/tmp/wanli-api-kitchen-jsonb-20260727`.

- [ ] **Step 4: Verify Android project quality gates**

Run from `frontend-android/`:

```bash
./gradlew testDebugUnitTest assembleRelease lintDebug
```

Expected: `BUILD SUCCESSFUL` with no compile warning or lint error.

### Task 3: Safely replace the WanLi backend on port 8081

**Files:**
- Consume: `/tmp/wanli-api-kitchen-jsonb-20260727`
- Create: `/tmp/wanli-api-before-kitchen-jsonb`
- Create: `/tmp/wanli-api-kitchen-jsonb.log`

**Interfaces:**
- Consumes: verified backend binary and existing `backend-go/.env`
- Produces: corrected WanLi API on `0.0.0.0:8081`

- [ ] **Step 1: Confirm the current service and resolve its exact PID**

Run outside the filesystem/network sandbox:

```bash
wanli_pid=$(ss -ltnp 'sport = :8081' | sed -n 's/.*pid=\([0-9]\+\).*/\1/p')
test -n "$wanli_pid"
ps -p "$wanli_pid" -o pid=,comm=,args=
readlink -f "/proc/$wanli_pid/exe"
curl -sS -i http://127.0.0.1:8081/api/v1/health
```

Expected: one `api`/WanLi listener is identified and health returns 200. Stop if the process does not belong to WanLi.

- [ ] **Step 2: Preserve the current executable for recovery**

Run:

```bash
wanli_pid=$(ss -ltnp 'sport = :8081' | sed -n 's/.*pid=\([0-9]\+\).*/\1/p')
test -n "$wanli_pid"
cp "/proc/$wanli_pid/exe" /tmp/wanli-api-before-kitchen-jsonb
chmod 0755 /tmp/wanli-api-before-kitchen-jsonb
```

Expected: `/tmp/wanli-api-before-kitchen-jsonb` is executable.

- [ ] **Step 3: Gracefully stop only the confirmed listener**

Run:

```bash
wanli_pid=$(ss -ltnp 'sport = :8081' | sed -n 's/.*pid=\([0-9]\+\).*/\1/p')
test -n "$wanli_pid"
kill -TERM "$wanli_pid"
for attempt in $(seq 1 40); do
  if ! ss -ltn 'sport = :8081' | rg -q ':8081'; then
    break
  fi
  sleep 0.25
done
! ss -ltn 'sport = :8081' | rg -q ':8081'
```

Expected: port 8081 becomes free within 10 seconds.

- [ ] **Step 4: Start the corrected backend**

Run from `backend-go/`:

```bash
PORT=8081 nohup /tmp/wanli-api-kitchen-jsonb-20260727 >/tmp/wanli-api-kitchen-jsonb.log 2>&1 &
```

Expected: the shell prints a new PID.

- [ ] **Step 5: Verify startup and listener identity**

Run:

```bash
for attempt in $(seq 1 40); do
  if curl -fsS http://127.0.0.1:8081/api/v1/health >/dev/null; then
    break
  fi
  sleep 0.25
done
curl -sS -i http://127.0.0.1:8081/api/v1/health
ss -ltnp 'sport = :8081'
```

Expected: health returns 200 and `/tmp/wanli-api-kitchen-jsonb-20260727` owns port 8081. If startup fails, inspect `/tmp/wanli-api-kitchen-jsonb.log` and restart `/tmp/wanli-api-before-kitchen-jsonb` on port 8081.

### Task 4: Verify the live kitchen API and clean up

**Files:**
- Read: `/tmp/wanli-api-kitchen-jsonb.log`
- Create temporarily: `/tmp/wanli-kitchen-put.json`
- Create temporarily: `/tmp/wanli-kitchen-get.json`

**Interfaces:**
- Consumes: live authenticated `PUT/GET /api/v1/me/kitchen`
- Produces: evidence that the Android click path now returns 200 and round-trips both arrays

- [ ] **Step 1: Request a code for the dedicated diagnostic account**

Run:

```bash
curl -fsS -X POST http://127.0.0.1:8081/api/v1/auth/email-code \
  -H 'Content-Type: application/json' \
  -d '{"email":"codex-kitchen-postfix@example.com"}'
```

Expected: a success response.

- [ ] **Step 2: Log in without printing the token**

Run:

```bash
code=$(sed -n 's/.*verification code for codex-kitchen-postfix@example.com: \([0-9]\{6\}\).*/\1/p' \
  /tmp/wanli-api-kitchen-jsonb.log | tail -n 1)
test -n "$code"
login_response=$(curl -fsS -X POST http://127.0.0.1:8081/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"codex-kitchen-postfix@example.com\",\"code\":\"$code\"}")
token=$(printf '%s' "$login_response" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
test -n "$token"
```

Expected: all checks exit 0 and neither secret nor token is printed.

- [ ] **Step 3: Save and read both kitchen lists**

Run the login parsing and requests in one shell so the token remains private:

```bash
code=$(sed -n 's/.*verification code for codex-kitchen-postfix@example.com: \([0-9]\{6\}\).*/\1/p' \
  /tmp/wanli-api-kitchen-jsonb.log | tail -n 1)
login_response=$(curl -fsS -X POST http://127.0.0.1:8081/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"codex-kitchen-postfix@example.com\",\"code\":\"$code\"}")
token=$(printf '%s' "$login_response" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
put_status=$(curl -sS -o /tmp/wanli-kitchen-put.json -w '%{http_code}' \
  -X PUT http://127.0.0.1:8081/api/v1/me/kitchen \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $token" \
  -d '{"owned_ingredients":["盐","生抽"],"owned_tools":["炒锅","烤箱"]}')
get_status=$(curl -sS -o /tmp/wanli-kitchen-get.json -w '%{http_code}' \
  http://127.0.0.1:8081/api/v1/me/kitchen \
  -H "Authorization: Bearer $token")
test "$put_status" = "200"
test "$get_status" = "200"
rg -q '"owned_ingredients":\["盐","生抽"\]' /tmp/wanli-kitchen-put.json
rg -q '"owned_tools":\["炒锅","烤箱"\]' /tmp/wanli-kitchen-put.json
rg -q '"owned_ingredients":\["盐","生抽"\]' /tmp/wanli-kitchen-get.json
rg -q '"owned_tools":\["炒锅","烤箱"\]' /tmp/wanli-kitchen-get.json
```

Expected: every assertion exits 0.

- [ ] **Step 4: Delete only the diagnostic account**

Run:

```bash
docker exec wanli-postgres psql -U postgres -d wanli_db \
  -c "DELETE FROM users WHERE email = 'codex-kitchen-postfix@example.com';"
```

Expected: `DELETE 1`; the foreign key cascade removes its kitchen profile.

- [ ] **Step 5: Perform final health verification**

Run:

```bash
curl -sS -i http://127.0.0.1:8081/api/v1/health
```

Expected: HTTP 200 after live API verification and cleanup.
