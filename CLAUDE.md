# 项目目录结构

新增代码时优先遵循现有分层：入口组装放在 `cmd/novels_ai`，配置和基础设施初始化放在 `internal/bootstrap`，业务规则放在 `internal/biz`，数据访问放在 `internal/data`，HTTP 路由与接口适配放在 `internal/service`，跨层复用的小工具放在 `internal/pkg/common`。


# Code Style Constraints

1. Define request parameters uniformly under `internal/data/dto`. The service layer and biz layer should share the same DTOs instead of defining duplicate parameter structs.
2. The service layer is responsible for receiving requests, binding parameters, and basic parameter validation. Gin's `ShouldBindJSON` can validate based on tags from the `go-playground/validator/v10` library, so do not repeat validation in code unless the user explicitly asks for it.
3. The biz layer should only contain business logic. Do not perform input validity checks there, so the business code stays lean.
4. When updating the database, prefer passing the complete model to GORM `Save`. Do not check parameters field by field in the biz layer and manually assemble an update map.


# Constraints

1. Add detailed Chinese comments to the code you write.
2. Do not write excessive defensive code.
3. Avoid using `else` where possible; in scenarios requiring many `if-else` statements, use `switch` instead.
4. The swagger documentation is generated using the command `swag init -g .\internal\service\route.go`.


# CLAUDE.md

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

---

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.
