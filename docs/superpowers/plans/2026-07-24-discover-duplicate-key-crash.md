# Discover Duplicate-Key Crash Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop the Android discover screen from crashing on duplicate recipe IDs and deploy the current backend list query on port `8081`.

**Architecture:** Use `recipe_version_id` as the Android lazy-list key while preserving the existing API DTO. Rebuild the current Go source and gracefully replace only the confirmed stale WanLi process listening on `8081`.

**Tech Stack:** Kotlin, Jetpack Compose, JUnit 4, Go 1.22+, Gin

## Global Constraints

- Do not change the REST API contract or database schema.
- Preserve the Android base URL `http://10.0.2.2:8081/api/v1/`.
- Stop only the verified stale WanLi process serving port `8081`.
- Do not expose secrets from `backend-go/.env` or process environments.

---

### Task 1: Add a stable discover-list key

**Files:**
- Create: `frontend-android/app/src/test/kotlin/com/wanli/android/presentation/discover/DiscoverScreenTest.kt`
- Modify: `frontend-android/app/src/main/kotlin/com/wanli/android/presentation/discover/DiscoverScreen.kt:119`

**Interfaces:**
- Consumes: `RecipeSummaryDto.recipe_version_id: String`
- Produces: `discoverItemKey(item: RecipeSummaryDto): String`

- [ ] **Step 1: Write the failing regression test**

```kotlin
package com.wanli.android.presentation.discover

import com.wanli.android.data.remote.RecipeSummaryDto
import org.junit.Assert.assertNotEquals
import org.junit.Test

class DiscoverScreenTest {
    @Test
    fun `list keys remain unique when recipe ids are duplicated`() {
        val first = summary("version-1")
        val second = summary("version-2")

        assertNotEquals(discoverItemKey(first), discoverItemKey(second))
    }

    private fun summary(versionId: String) = RecipeSummaryDto(
        recipe_id = "recipe-1",
        title = "红烧肉",
        recipe_version_id = versionId
    )
}
```

- [ ] **Step 2: Run the focused test and verify RED**

Run:

```bash
cd frontend-android
./gradlew testDebugUnitTest --tests '*DiscoverScreenTest'
```

Expected: compilation fails because `discoverItemKey` does not exist.

- [ ] **Step 3: Implement the minimal key selector**

Add:

```kotlin
internal fun discoverItemKey(item: RecipeSummaryDto): String = item.recipe_version_id
```

Use it in the list:

```kotlin
items(state.items, key = ::discoverItemKey) { item ->
```

- [ ] **Step 4: Run the focused test and verify GREEN**

Run:

```bash
cd frontend-android
./gradlew testDebugUnitTest --tests '*DiscoverScreenTest'
```

Expected: `BUILD SUCCESSFUL`.

### Task 2: Build and replace the stale backend process

**Files:**
- Runtime binary: `/tmp/wanli-api`
- Runtime log: `/tmp/wanli-api.log`

**Interfaces:**
- Consumes: existing `backend-go/.env`, PostgreSQL, Redis, and MinIO configuration
- Produces: current WanLi API on `0.0.0.0:8081`

- [ ] **Step 1: Verify the current failure evidence**

Run:

```bash
curl -sS 'http://127.0.0.1:8081/api/v1/recipes?sort=latest&page=1&page_size=20'
```

Expected: at least two items share the same `recipe_id`.

- [ ] **Step 2: Verify and build the current Go source**

Run:

```bash
cd backend-go
go test ./...
go vet ./...
go build -o /tmp/wanli-api.next ./cmd/api
```

Expected: all three commands exit 0.

- [ ] **Step 3: Gracefully stop only the confirmed stale process**

Run:

```bash
kill -TERM 2934506
```

Expected: PID `2934506` exits and port `8081` becomes available.

- [ ] **Step 4: Install and start the rebuilt binary**

Run from `backend-go/`:

```bash
mv /tmp/wanli-api.next /tmp/wanli-api
PORT=8081 nohup /tmp/wanli-api >/tmp/wanli-api.log 2>&1 &
```

Expected: a new `wanli-api` process listens on `8081`.

- [ ] **Step 5: Verify the new backend response**

Run:

```bash
curl -sS -i http://127.0.0.1:8081/api/v1/health
curl -sS 'http://127.0.0.1:8081/api/v1/recipes?sort=latest&page=1&page_size=20'
```

Expected: health returns 200 and the recipe list has no duplicate `recipe_id`.

### Task 3: Run project-level verification

**Files:**
- Verify: `frontend-android/app/build/outputs/apk/debug/app-debug.apk`

**Interfaces:**
- Consumes: the completed Android and backend fixes
- Produces: a tested Debug APK ready for MuMu installation

- [ ] **Step 1: Verify Android**

Run:

```bash
cd frontend-android
./gradlew testDebugUnitTest assembleDebug lintDebug
```

Expected: `BUILD SUCCESSFUL`.

- [ ] **Step 2: Verify Go**

Run:

```bash
cd backend-go
go fmt ./...
go vet ./...
go test ./...
go build ./cmd/api
```

Expected: every command exits 0.

No commit step is available because the supplied workspace is not recognized
as a Git repository.
