# Android Backend Port Correction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the Android client request the WanLi Go backend on its actual port, `8081`.

**Architecture:** Keep the existing asset-based Retrofit configuration and change only its port. Verify both the source asset and the value packaged into the Debug APK.

**Tech Stack:** Android, Kotlin, Gradle

## Global Constraints

- Modify only the Android API base URL port.
- Keep the host `10.0.2.2`, path `/api/v1/`, and cleartext network policy unchanged.
- Do not modify API contracts because no endpoint shape changes.

---

### Task 1: Correct the packaged backend port

**Files:**
- Modify: `frontend-android/app/src/main/assets/config.properties:1`
- Test: one-off shell assertion plus Android Debug build

**Interfaces:**
- Consumes: WanLi Go backend health endpoint on port `8081`
- Produces: Retrofit base URL `http://10.0.2.2:8081/api/v1/`

- [ ] **Step 1: Run the expected-value assertion before the change**

Run:

```bash
test "$(sed -n 's/^BASE_URL=//p' frontend-android/app/src/main/assets/config.properties)" = "http://10.0.2.2:8081/api/v1/"
```

Expected: exit code 1 because the current port is `8080`.

- [ ] **Step 2: Apply the minimal configuration change**

Set the complete file content to:

```properties
BASE_URL=http://10.0.2.2:8081/api/v1/
```

- [ ] **Step 3: Run the expected-value assertion after the change**

Run:

```bash
test "$(sed -n 's/^BASE_URL=//p' frontend-android/app/src/main/assets/config.properties)" = "http://10.0.2.2:8081/api/v1/"
```

Expected: exit code 0.

- [ ] **Step 4: Build the Debug APK**

Run:

```bash
cd frontend-android
./gradlew assembleDebug
```

Expected: exit code 0 and `BUILD SUCCESSFUL`.

- [ ] **Step 5: Inspect the value packaged in the APK**

Run:

```bash
unzip -p frontend-android/app/build/outputs/apk/debug/app-debug.apk assets/config.properties
```

Expected:

```properties
BASE_URL=http://10.0.2.2:8081/api/v1/
```

No commit step is available because the supplied workspace contains an empty
`.git` directory and is not recognized as a Git repository.
