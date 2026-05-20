# 项目目录结构

当前仓库是 Go module `Novels_AI/backend`，核心代码按 Go 服务端常见分层组织：

```text
.
├── cmd/novels_ai/          # 服务启动入口，`main.go` 负责组装并启动应用
├── config/                 # 运行配置文件，包含默认配置和开发环境配置
├── docs/                   # Swagger/OpenAPI 文档产物
├── internal/               # 后端核心实现，仅供本模块内部引用
│   ├── biz/login/          # 登录相关业务逻辑及测试
│   ├── bootstrap/          # 配置、数据库、日志等启动初始化逻辑
│   ├── data/               # 数据访问层
│   │   └── model/          # 数据模型定义，如管理员和小说模型
│   ├── middleware/         # HTTP 中间件，如错误处理、限流、请求 ID
│   ├── pkg/common/         # 通用错误、响应和会话辅助能力
│   └── service/            # 路由注册和接口服务层
│       └── login/          # 登录接口服务实现
├── go.mod / go.sum         # Go module 依赖声明与锁定文件
├── README.md               # 项目简要说明
```

新增代码时优先遵循现有分层：入口组装放在 `cmd/novels_ai`，配置和基础设施初始化放在 `internal/bootstrap`，业务规则放在 `internal/biz`，数据访问放在 `internal/data`，HTTP 路由与接口适配放在 `internal/service`，跨层复用的小工具放在 `internal/pkg/common`。

# Constraints

1. Add detailed Chinese comments to the code you write.
2. Do not write excessive defensive code.
3. Avoid using `else` where possible; in scenarios requiring many `if-else` statements, use `switch` instead.


# AGENTS.md

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
