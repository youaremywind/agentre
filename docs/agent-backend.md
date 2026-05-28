# 接入新的 AI Agent 后端

新增一个 Agent backend（例如 Gemini CLI / 自家 CLI / 另一个 in-process SDK）时要走的路径、改动点、约束和坑。写代码前先全部看完——`agentruntime.Runtime` 接口看着窄，但配套要补齐的东西散在 entity / repo / service / wire / daemon / 前端六层。

> 前置阅读：[architecture.md](architecture.md)（分层约定）、[development.md](development.md)（TDD/BDD + Fix Discipline）。

---

## 0. 先想清楚的问题

写代码前自问，**不清楚就停下来问用户**，不要写到一半才发现要回炉：

1. **运行模式**：是 in-process（像 builtin，直接吃 LLMProvider 走 cago app/coding），还是 wrap 一个本地 CLI 子进程（像 claudecode / codex）？两者在 ProviderType 匹配、cli_path 校验、env 透传、Prober 实现上完全不一样。
2. **是否支持远端执行**：要走 `agentred` daemon 投到 LAN 机器上跑吗？支持的话要保证 init() 注册到的 `RuntimeFor` 不依赖 desktop-only 的服务（chat_repo / GUI），且 wire 协议覆盖到所有 RPC 帧。
3. **能力矩阵**：能不能 mid-turn steer？能不能 abort？是否有 `can_use_tool` 协议？是否支持 ask_user_question？是否支持 plan / permission mode 切换？是否能 fork session？逐项列出来——这直接决定你要实现 `agentruntime` 哪几个可选子接口。
4. **协议形态**：事件流是 stdout JSONL（claudecode）？JSON-RPC over stdio（codex app-server）？还是内存 channel（builtin）？translator 是无状态纯函数 vs. 有状态聚合，决定从哪写第一条测试。
5. **session 复用 / spawn 单次**：每个 turn 起新进程（codex 当前做法），还是子进程常驻、用 LRU 缓存复用（claudecode）？复用要管理 idle evict、abort 解锁、跨 turn 状态 (permission mode / steer queue)。

---

## 1. 整体接入路径（必经的 7 层）

新 backend = 沿着以下 7 层各加一刀，缺一不可：

| 层 | 位置 | 必改？ |
| --- | --- | --- |
| 1. Entity 类型 | `internal/model/entity/agent_backend_entity/{agent_backend.go, kinds.go}` | 必须 |
| 2. 数据库迁移 | `migrations/YYYYMMDDNNNN_*.go` + `migrationList()` 末尾 append | 仅当要加新列 |
| 3. CLI/Prober/Env 装配 | `internal/pkg/agentruntime/clienv.go` + `internal/service/agent_backend_svc/{prober.go, resolve_cli.go}` | CLI 类必须；in-process 仅 Prober |
| 4. Runtime + Translator | `internal/pkg/agentruntime/runtimes/<name>/{runtime.go, translator.go}` | 必须 |
| 5. Daemon import | `internal/daemon/runtime_imports.go` blank import | 支持远端就必须 |
| 6. Wails 绑定与 svc 类型 | `internal/service/agent_backend_svc/types.go` + `internal/app/agent.go`（若引入新字段） | 仅当新字段 |
| 7. 前端绑定 + UI gating | `frontend/wailsjs/` 重生（`make generate`）+ capability pill / 选择器 | 必须 |

---

## 2. 各层必做的事

### 2.1 Entity（`agent_backend_entity`）

- 在 `agent_backend.go` 加 `BackendType` 常量：

  ```go
  TypeMyAgent BackendType = "myagent"
  ```

- 在 `kinds.go` 实现 `BackendKind` 接口并登记到 `backendKinds`：

  ```go
  type myAgentKind struct{}
  func (myAgentKind) Type() BackendType { return TypeMyAgent }
  func (myAgentKind) KnownAliases() []string { return nil }            // 没有 model_routes 就 nil
  func (myAgentKind) ProviderTypeMatch(t llm_provider_entity.ProviderType) bool {
      return t == llm_provider_entity.TypeXxx
  }
  func (myAgentKind) AllowsCLIPath() bool { return true }              // CLI 类填 true
  func (myAgentKind) ValidateExtra(ctx context.Context, b *AgentBackend) error {
      // 这里校验 sandbox / approval / default_permission_mode / env_json 独有字段；
      // 公共字段（name / type / env_json 保留键 / model_routes alias 集合 / reasoning_effort）
      // 已经在 AgentBackend.Check 里走过，**不要再重复一遍**。
      return nil
  }
  ```

- `IsXxx()` 便捷判断方法不强制加，但加了 chat_svc / 前端可读性更好（参考 `IsClaudeCode/IsCodex`）。
- 充血模型边界：**新增字段优先放在 entity，方法（校验、默认值、序列化）也写在 entity**，service 只做跨 entity 编排。`AgentBackend.Check` 已经分派到 `BackendKind.ValidateExtra`——千万别在 service 里再写 switch。

### 2.2 数据库迁移

只有新字段时才动。规则：

- 在 `migrations/` **新建** `YYYYMMDDNNNN_xxx.go`，导出 `migrationYYYYMMDDNNNN()`。
- **禁止改动既有迁移**——需要修复的也写新的补丁迁移。
- DDL 走原生 SQL (`tx.Exec(...)`)，不要依赖 `AutoMigrate` 的隐式行为。
- 在 `migrations/migrations.go::migrationList()` **末尾** append。
- 默认值必须能让既有行通过 entity.Check（通常 `'' / '{}' / 0`）。

### 2.3 Prober + CLI 探测 + Env 装配

`agent_backend_svc.Prober` 抽象「对一条 backend 跑一轮自检 → 返回 reply 或 error」，供前端「测试连通性」按钮用。

- 在 `prober.go` 注册：

  ```go
  var proberRegistry = map[agent_backend_entity.BackendType]Prober{
      agent_backend_entity.TypeBuiltin:   builtinProber{},
      agent_backend_entity.TypeMyAgent:   myAgentProber{},
  }
  ```

- CLI 类后端如果要走本地 gateway 测，把 env 装配抽到 `internal/pkg/agentruntime/<name>_env.go`，**和 chat path 的 runtime 共享同一份装配规则**（chat 实跑和 Test 不能漂移；参考 `BuildClaudeCodeEnv` / `BuildCodexEnv`）。
- 如果是 CLI 类，`resolve_cli.go` 里加 `Type` 分支让前端编辑器能 `cliprober` 探到本机 binary 绝对路径。

### 2.4 Runtime（核心）

新建包 `internal/pkg/agentruntime/runtimes/<name>/`，至少 3 个文件：

1. **`runtime.go`** — 实现 `agentruntime.Runtime` 接口：

   ```go
   var defaultRuntime = New()

   func init() {
       agentruntime.RegisterRuntime(agent_backend_entity.TypeMyAgent, defaultRuntime)
   }

   type Runtime struct { /* sessions map[int64]*active */ }

   func New() *Runtime { ... }

   func (r *Runtime) Capabilities() capability.Capabilities { ... }

   func (r *Runtime) Run(ctx context.Context, req agentruntime.RunRequest) (
       <-chan agentruntime.Event, *agentruntime.RunResult, error,
   ) { ... }
   ```

   - **`Capabilities()` 必须返回稳定结果**（同一 runtime 多次调用相同），前端 capability gating / 远端 prefetch 都依赖这个语义。
   - `Run` 返回的 channel **必须在 turn 结束时 close**（无论 Done / Error / Abort）。**channel 关闭前不允许读 `*RunResult`**——`RunResult` 是异步填充。
   - `RunRequest.Cwd` 非空时直接用作 cwd；为空时回退到 `agentruntime.AgentCwd(req.AgentID)`。**不要反向依赖 `project_svc`**。
   - `ForkAnchor` 非空时实现「重新生成」——参考 claudecode 用 `--fork-session`、codex 用 `thread/rollback`。
   - 主 goroutine 退出前 `unregister(sessionID)` 清掉 sessions map，避免 leak。

2. **`translator.go`** — 把 backend 自家事件翻译成 sealed `agentruntime.Event`：

   ```go
   func translate(ev myAgentEvent) (events []agentruntime.Event, usage *provider.Usage, stopErr error) { ... }
   ```

   - **保持纯函数**——一帧入、0/1/n 帧出，**不读写 runtime 状态**。状态聚合（同安全点连续到达的 SteerConsumed 合并 / pending steer FIFO 配对）在 `runtime.Run` 的 drain 循环里做。
   - 工具识别走 `internal/pkg/agentruntime/canonical`——能识别成 `FileWrite/FileEdit/PlanUpdate/AgentSpawn` 的就填 `ToolCall.Canonical`，UI 投影 / 前端卡片复用同一套；不能识别的留 raw。
   - `EventUsage`：translator 内部按 family 把 `TotalInputTokens` 算好（Anthropic = prompt+cached+cacheCreation；OpenAI = prompt），下游不再做家族判断。

3. **`runtime_test.go`** + **`translator_test.go`** — TDD 必须：
   - **Capabilities 矩阵测试**：声明的 cap=true 与实现的子接口（Steerer / Aborter / 等）必须一致。参考 `runtime_test.go::TestXxxCapabilities`。
   - **Translator pure-fn 测试**：表驱动覆盖每种事件 kind + 边界（空 tool input / 错误帧 / 部分字段缺失）。Convey 嵌套描述场景。
   - **Run 集成测试**：用 fake backend 客户端 / fake session 验证事件批次顺序、`SteerConsumed` 合并、Abort 解锁、RunResult 终态。

#### 可选控制子接口

按 Capabilities 声明实现（**声明了 cap=true 就必须实现对应接口**；matrix 测试会卡掉）：

| Cap | 接口 | 何时实现 |
| --- | --- | --- |
| `CapSteer` | `Steerer` | 支持 mid-turn 注入用户消息 |
| `CapCancelSteer` | `SteerCanceler` | 注入后还能撤回（claudecode 有；codex 无） |
| `CapDrainSteer` | `SteerDrainer` | turn 间 leftover 自动续投（仅 claudecode） |
| `CapAbort` | `Aborter` | 用户「停止」按钮——基本都得实现 |
| `CapSetPermission` | `PermissionModeSetter` | 运行时切换 permission mode |
| `CapAnswerUserAsk` | `AskAnswerSink` | 处理反向 ask_user_question |
| `CapToolPermission` | `ToolPermissionSink` | 处理 `can_use_tool` 协议 |
| `CapForkSession` | `RunRequest.ForkAnchor` 内置语义 | 「重新生成」走 fork |
| `CapReportContextWindow` | emit `ContextWindowUpdated` | runtime 能探到模型实际窗口 |
| `CapCompact` | `RunRequest.Compact=true` 内置语义 | 原生 compact turn |

不实现的 cap：**chat_svc 拿到前端请求时返回 `ErrUnsupported`**——错误码已经在 wire 层 sentinel 化跨进程透明传递，**不要私造错误**。

#### Sentinel 错误（必须用，不要造新的）

```go
agentruntime.ErrNoActiveTurn   // sessionID 没有 in-flight turn
agentruntime.ErrSteerNotFound  // CancelSteer 给的 queuedID 已被消费 / 不存在
agentruntime.ErrAborted        // 用户主动 Abort，写到 RunResult.StopErr
agentruntime.ErrUnsupported    // 当前 runtime 不支持该 cap
```

新错误必须有跨进程语义时，**先在 `errors.go` + `wire.go` 同步加 sentinel + error code**，不要在 daemon handler 里临时 stringify。

### 2.5 Daemon import（远端执行）

要让新 backend 在 `agentred` daemon 上跑，**只**改 `internal/daemon/runtime_imports.go`：

```go
import (
    _ "agentre/internal/pkg/agentruntime/runtimes/builtin"
    _ "agentre/internal/pkg/agentruntime/runtimes/claudecode"
    _ "agentre/internal/pkg/agentruntime/runtimes/codex"
    _ "agentre/internal/pkg/agentruntime/runtimes/myagent"   // ← 加这行
)
```

- 注册依赖 init() 副作用——所以 runtime 包的 `init()` **只**做 `RegisterRuntime`，**不要起 goroutine / 读环境变量 / 打开文件**，否则 daemon 进程启动时会触发副作用。
- daemon 端不 bootstrap chat_repo——runtime 内部不能反向依赖 repository 包。需要持久化的运行时状态（比如 `LaunchPermissionMode`）走 `RunResult` 字段回吐给 chat_svc。
- `runtime_imports_test.go` 会枚举 `RegisteredRuntimes()` 跑一轮 capability 协议测试——确保新 runtime 加进去后这个测试还过。

### 2.6 Service / Wails 绑定

只有新增字段时才动：

- `internal/service/agent_backend_svc/types.go`：在 `BackendItem` / `CreateBackendRequest` / `UpdateBackendRequest` / `TestBackendRequest` 加字段。**字段名稳定、json tag 明确**——`make generate` 会把它提到前端 TS 类型。
- `internal/service/agent_backend_svc/agent_backend.go`：在 buildEntity / mapItem 里读写新字段。
- `internal/app/agent.go`：绑定层方法**只做** parse → svc → return，业务塞进 `App` 会被 `go test` 漏掉。

### 2.7 前端

- `make generate` 重生 `frontend/wailsjs/` bindings。
- 编辑器 UI（`frontend/src/pages/Settings/AgentBackends/...`）：加 type 选项、新字段表单控件——**统一用 shadcn `@/components/ui/*`**，禁止新增原生 `<select>`。
- Capability gating：前端通过 `RegisteredRuntimes()` 投影出来的 caps 控制按钮是否显示——steer chip / abort 按钮 / permission mode pill / ask_user_question 卡。新 cap 加在 [docs/frontend.md](frontend.md) 提到的 capability hook 里。

---

## 3. 远端执行（`agentred`）的额外考量

桌面端可以把单个 chat 投到 LAN 的 `agentred` 跑：

```
desktop UI → internal/app → chat_svc → remote.Runtime
           → JSON-RPC over WebSocket (wire.MethodRun)
           → daemon/handlers/RuntimeHandlers
           → agentruntime.RuntimeFor(backendType) — 跑你新写的 *Runtime
```

要走通这条路：

1. Runtime 不依赖 desktop-only 的服务（chat_repo / GUI / system tray）。需要回吐的状态走 `RunResult` 字段，不要直接调 repo。
2. 新增的 sentinel 错误同步加 `wire.ErrCode*`——不然 `errors.Is(err, agentruntime.ErrXxx)` 在客户端会失效。
3. 新增的 Event 类型同步加 `wire.Event*` 编解码——`runtime.event` notification 是按 sealed Event tag 分发的。
4. 测试覆盖远端路径：起一对 in-memory `client.Client` ↔ `daemon.handlers.RuntimeHandlers`，验证 capability 协商 / Run / Abort / Steer 跨进程语义。

---

## 4. TDD / BDD 必备测试清单

**Red 阶段先写、跑、看到失败，再实现。** 没失败测试不写实现代码——这是 [development.md](development.md) §0 的硬规则。

| 测试 | 位置 | 验什么 |
| --- | --- | --- |
| Entity Check 表驱动 | `*_test.go` 同 kinds.go | name / type / env_json 保留键 / model_routes alias / cli_path 允许性 / kind 独有字段 |
| Capabilities 矩阵 | `runtimes/<name>/runtime_test.go` | 声明的 cap=true ↔ 实现了对应接口（type assert） |
| Translator 纯函数 | `runtimes/<name>/translator_test.go` | 每种 backend 事件 kind + 边界（空 input / partial fields / error frame） |
| Run 集成 | `runtimes/<name>/runtime_test.go` | 事件批次顺序、SteerConsumed 合并、Abort 解锁、RunResult 终态、ctx cancel 行为 |
| Prober | `agent_backend_svc/prober_test.go` | provider missing / 网络错误 / 工具循环正常都翻译到合适 reply/err |
| Wire round-trip | `runtimes/remote/wire/wire_test.go` | 新 Event / 新 sentinel 编解码对称 |
| Daemon registry | `daemon/runtime_imports_test.go` | 新 backend 出现在 `RegisteredRuntimes()` 里 |
| Service create/update/delete | `agent_backend_svc/agent_backend_test.go` | mockgen mock repo，验校验 + 落库字段 |

repo 单测一律 `testutils.Database(t)` + sqlmock，**禁止起真 SQLite**——参考 [development.md](development.md) §测试栈。

---

## 5. 常见坑 / 反模式（不要重蹈）

1. **不要把业务塞进 `internal/app/`**。Wails 绑定只做 `parse → svc.Xxx().Method(ctx, ...) → return`，否则 `go test` 覆盖不到。
2. **不要在 runtime `init()` 里起 goroutine / 读 env / 打开文件**。daemon 进程引导时会触发副作用，而且单测无法控制。`init()` 只做 `RegisterRuntime`。
3. **不要在 chat_svc 里 switch backendType**。新 backend 出现就抽 interface 让 runtime 自己声明能力——参考 capability matrix。`if backend.Type == "claudecode" { ... }` 是 smell。
4. **不要重复 normalize**。env / model_routes 在 entity.Check 已经 normalize 一次，service / runtime 不要再做第二次。
5. **不要让 translator 有状态**。状态聚合在 `Run` 的 drain 循环里做，translator 必须能在测试里独立跑、表驱动断言。
6. **不要把 SteerInbox / SessionCache 等运行期资源 hard-code 到 `New()`**。用包级 `SetXxx` 注入器（参考 `claudecode.Runtime.SetSteerInbox`），bootstrap 时装配，单测可换 fake。
7. **不要私造错误来表达 `ErrNoActiveTurn` / `ErrUnsupported`**。这些 sentinel 跨进程透明，私造 string 会让 chat_svc 翻译不出来。
8. **不要让 runtime 反向依赖 repository / chat_svc**。daemon 进程没 bootstrap repo——一调就 nil panic。状态走 `RunResult` 回吐。
9. **不要忽略 ctx**。`Run` 收到的 ctx 取消后必须 unblock 所有 I/O——这是 chat_svc 实现「停止」按钮的前提（claudecode 通过 control_request、codex 通过 turn/interrupt、builtin 通过 cancel turnCtx）。
10. **不要在新增 capability 的同一次提交里做 drive-by refactor**。Diff 只动 producer + 它的测试。看到无关脏数据先 flag——参考 CLAUDE.md / AGENTS.md §3 / [development.md](development.md) §Fix Discipline。

---

## 6. 提交前自检清单

- [ ] 新增 `BackendType` 常量 + `BackendKind` 实现 + 登记到 `backendKinds`
- [ ] 新字段的迁移文件 append 到 `migrationList()` 末尾，DDL 用原生 SQL，默认值能让既有行通过 Check
- [ ] 新 runtime 包 `init()` 只调 `RegisterRuntime`，无副作用
- [ ] Capabilities 声明与实际实现的子接口一致（matrix 测试过）
- [ ] Translator 是纯函数，表驱动测试覆盖 happy path + 至少一个 boundary/error
- [ ] `RunRequest.Cwd` 非空时优先用；`ForkAnchor` 非空时走 fork；`ctx` cancel 能 unblock
- [ ] `RunResult` 在 events channel close 前**不**被读；新字段（ProviderSessionID / Usage / Model / LaunchPermissionMode / ContextWindow / UserAnchor）按 backend 能力填
- [ ] 远端：`runtime_imports.go` blank import 已加；新 Event / sentinel 已加 wire 编解码 + round-trip 测试
- [ ] Wails 类型新字段稳定 json tag；`make generate` 已重生 `frontend/wailsjs/`
- [ ] Prober 已注册到 `proberRegistry`；CLI 类后端 env 装配在 `agentruntime/clienv.go` 与 chat path 共享
- [ ] 关键流程打日志：`logger.Ctx(ctx)`、message 用 `package.Method:` 前缀小写、字段用 `zap.Xxx`（参考 [development.md](development.md) §日志）
- [ ] `make check`（lint + test）全过；新包 service/repository 层覆盖率 ≥80%
