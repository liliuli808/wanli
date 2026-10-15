# AI Model Environment Configuration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the backend's existing `AI_MODEL` setting visible and configurable through the documented server environment variables.

**Architecture:** Keep model selection server-owned. The Go configuration and AI client already consume `AI_MODEL`, so this task only aligns the checked-in environment example and operator documentation with the implemented behavior.

**Tech Stack:** Go 1.22 configuration via Viper, dotenv-style environment files, Markdown documentation

## Global Constraints

- Do not add an Android model picker or accept a client-supplied model.
- Do not expose or commit any real API key.
- Keep `AI_MODEL=deepseek-chat` aligned with the existing Go default.
- Do not change the database, API contract, prompt, or AI input/output schema.
- A backend restart is required after changing the AI environment variables.

---

### Task 1: Document the existing AI model environment setting

**Files:**
- Modify: `backend-go/.env.example`
- Modify: `README.md`
- Modify: `docs/ai-parsing.md`

**Interfaces:**
- Consumes: `config.AIConfig{APIKey, BaseURL, Model}` loaded from `AI_API_KEY`, `AI_BASE_URL`, and `AI_MODEL`
- Produces: an operator-facing environment contract documenting the same three variable names

- [x] **Step 1: Verify the current documentation gap**

Run:

```bash
test "$(rg -l '^AI_MODEL=' backend-go/.env.example README.md docs/ai-parsing.md | wc -l)" -eq 3
```

Expected: FAIL with exit status `1`, because none of the three files currently contains an `AI_MODEL=` assignment.

- [x] **Step 2: Add the environment example**

Add this line immediately after `AI_BASE_URL` in `backend-go/.env.example`:

```env
AI_MODEL=deepseek-chat
```

- [x] **Step 3: Update the startup documentation**

Add the same line after `AI_BASE_URL` in the complete `.env` example in `README.md`. Add an explanation that:

```text
AI_MODEL is the model identifier sent to the provider configured by AI_BASE_URL.
The default is deepseek-chat, and changes take effect after restarting the backend.
```

- [x] **Step 4: Document the AI runtime variables**

Add an “AI 服务配置” subsection near the top of `docs/ai-parsing.md` that defines:

```text
AI_API_KEY: server-side provider credential; never expose it to Android or API responses.
AI_BASE_URL: base URL of an OpenAI Chat Completions-compatible service.
AI_MODEL: provider model identifier; defaults to deepseek-chat.
```

Also state that all three settings are read when the backend starts and therefore require a restart after modification.

- [x] **Step 5: Verify documentation consistency**

Run:

```bash
test "$(rg -l '^AI_MODEL=' backend-go/.env.example README.md docs/ai-parsing.md | wc -l)" -eq 3
rg -n 'AI_API_KEY|AI_BASE_URL|AI_MODEL|重启后端' README.md docs/ai-parsing.md backend-go/.env.example
```

Expected: PASS; output shows all three variable names and the restart instruction.

- [x] **Step 6: Run the focused Go configuration tests**

Run:

```bash
cd backend-go
go test ./internal/config
```

Expected: PASS.

No commit step is available because the supplied workspace is not recognized as a Git repository.
