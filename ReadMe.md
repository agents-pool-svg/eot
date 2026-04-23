# EoT — Exchange-of-Thought

> A minimal, idiomatic Go implementation of the **Exchange-of-Thought** paradigm for multi-LLM collaborative reasoning, usable both as a **library** and as a **CLI**.

参考论文：*Yao et al., "Exchange-of-Thought: Enhancing Large Language Model Capabilities through Cross-Model Communication", EMNLP 2023.*

---

## 1. 框架设计思路

### 它解决什么问题

- **CoT / ToT / GoT**：单个模型内部的推理结构（链 / 树 / 图）。
- **AutoGen / LangGraph**：工程编排框架，解决"多个 Agent 怎么组织起来跑"。
- **EoT**：推理方法论，解决"多个模型之间**怎么交换中间思考**才能推理得更好"。

EoT 不和 AutoGen / LangGraph 冲突——你完全可以在它们里面把 EoT 作为"多 Agent 协作节点"使用。本库专注把 EoT 的核心抽象做得小而正交。

### 核心抽象

| 抽象 | 职责 |
|------|------|
| `Thought` | 一次推理产物（含完整 CoT 文本 + 可选的 `#### <answer>` 结尾答案） |
| `Agent` | 一个 LLM 角色（id + system prompt + model + temperature） |
| `Topology` | 通信拓扑，决定**本轮谁能看到谁的上一轮/本轮 Thought** |
| `Runner` | 多轮驱动器，负责迭代、收敛判断、产出 `Result` |

### 四种内置 Topology

| 名称 | 可见性规则 | 典型场景 |
|------|------------|----------|
| `memory` | 共享黑板，每人看所有历史 | 自由讨论 / 头脑风暴 |
| `report` | 外围 Agent 只看中心最新输出；中心看所有外围本轮输出 | 主-从 / 汇报审核 |
| `relay` | 每人只看上一位；第一位 wrap 到上一轮的末位 | 流水线 / 迭代精修 |
| `debate` | 每人看所有对手上一轮输出 | 辩论 / 交叉质询 |

扩展只需实现 `Topology` 接口的 3 个方法 (`Name`、`TurnOrder`、`VisibleThoughts`) 并在 `BuildTopology` 里注册即可。

### 收敛判断

每轮结束后，从各 Agent 本轮的 `#### <answer>` 标记中统计频次，若多数比例 ≥ `ConvergenceThreshold`（默认 `1.0` 完全一致）则提前退出，节省 token。

---

## 2. 工程结构

```
test0project/
├── go.mod                         # module: github.com/agents-pool-svg/eot
├── pkg/eot/                       # SDK，供外部 import
│   ├── doc.go
│   ├── config.go                  # 多级配置加载（显式/env/.env/ReadMe.md）
│   ├── llmclient.go               # OpenAI 兼容 HTTP 客户端
│   ├── thought.go
│   ├── agent.go
│   ├── topology.go                # 4 种拓扑 + 工厂
│   ├── runner.go
│   └── api.go                     # Run() 一站式入口
├── cmd/eot/                       # CLI 可执行
│   ├── main.go
│   └── cmd/                       # cobra 子命令
│       ├── root.go
│       ├── run.go                 # eot run
│       ├── list.go                # eot list-topologies
│       └── version.go             # eot version
├── examples/
│   ├── library_usage/main.go      # SDK 集成示例
│   └── agents.json                # CLI 用的 agents 文件样例
└── ReadMe.md
```

---

## 3. 配置来源

优先级从高到低（找到即停）：

1. **显式参数 / CLI flag**：`WithAPIBase(...)` / `--api-base` / `--api-key` / `--model`
2. **环境变量**：
   - 新名：`EOT_API_BASE` / `EOT_API_KEY` / `EOT_MODEL`
   - 兼容别名：`CODEMATRIX_LLM_API_BASE` / `CODEMATRIX_LLM_API_KEY` / `CODEMATRIX_LLM_MODEL`
   - OpenAI 兼容：`OPENAI_BASE_URL` / `OPENAI_API_KEY` / `OPENAI_MODEL`
3. **文件**（工作目录下按顺序扫描）：`.env` → `ReadMe.md` → `README.md`，逐行匹配 `KEY=VALUE`

> 文档里用 `ReadMe.md` 做 KV 源纯粹是为了**开发期便利**，发布时请改用 env 或 `.env`。

API Base 会自动补 `/v1`（`http://host` → `http://host/v1`）。

本仓库 `ReadMe.md` 底部（见文末）就是一个示例配置。

---

## 4. 作为库使用（Go SDK）

### 4.1 安装

```bash
go get github.com/agents-pool-svg/eot@latest
```

### 4.2 最简调用：`eot.Run()` 一站式入口

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/agents-pool-svg/eot/pkg/eot"
)

func main() {
    res, err := eot.Run(context.Background(), eot.RunRequest{
        Question: "2 + 2 * 3 = ?",
        Agents: []eot.AgentSpec{
            {ID: "Planner",    System: "You are a careful math planner."},
            {ID: "Calculator", System: "You double-check arithmetic.", Temperature: 0.2},
            {ID: "Reviewer",   System: "You are a skeptical reviewer."},
        },
        Topology:  eot.TopologySpec{Name: "debate"},
        MaxRounds: 3,
        Verbose:   true,

        // 方式一：显式传凭证
        ConfigOpts: []eot.ConfigOption{
            eot.WithAPIBase("https://api.openai.com"),
            eot.WithAPIKey("sk-..."),
            eot.WithModel("gpt-4o-mini"),
        },
        // 方式二：省略 ConfigOpts，自动从 env / .env / ReadMe.md 加载
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Final answer:", res.FinalAnswer)
}
```

`RunRequest` 把"问题 / agents / topology / 凭证 / 轮数 / 阈值 / 流式回调"都打包进一次调用，非常适合被上层系统集成。

### 4.3 进阶：手动组装，便于复用与测试

```go
cfg, _ := eot.LoadConfig(eot.WithAPIBase("..."), eot.WithAPIKey("..."))
llm, _ := eot.NewLLMClient(cfg)

agents := []*eot.Agent{
    eot.NewAgent(llm, eot.AgentSpec{ID: "A", System: "role A"}),
    eot.NewAgent(llm, eot.AgentSpec{ID: "B", System: "role B"}),
}

runner := &eot.Runner{
    Agents:    agents,
    Topology:  eot.DebateTopology{},
    MaxRounds: 3,
    Verbose:   false,
    OnThought: func(t *eot.Thought) {
        // 接入日志 / WebSocket 推送 / tracing …
        fmt.Printf("[%s r%d] %d bytes\n", t.AgentID, t.Round, len(t.Content))
    },
}
res, err := runner.Run(context.Background(), "your question")
_ = res; _ = err
```

### 4.4 自定义 Topology

```go
type MyTopology struct{}

func (MyTopology) Name() string                             { return "mine" }
func (MyTopology) TurnOrder(ids []string, _ int) []string   { return ids }
func (MyTopology) VisibleThoughts(ids []string, all []*eot.Thought, cur string, r int) []*eot.Thought {
    // 自定义可见性逻辑 …
    return nil
}
```

放进 `Runner.Topology` 就能用；也可以 fork 一下在 `BuildTopology` 里注册进工厂。

### 4.5 运行示例

```bash
# 需先在 ReadMe.md / 环境变量里配好 API base/key
go run ./examples/library_usage
```

---

## 5. 作为 CLI 使用

### 5.1 构建与安装

```bash
# 本地构建
go build -o bin/eot ./cmd/eot
./bin/eot version

# 或一键安装到 $GOPATH/bin
go install github.com/agents-pool-svg/eot/cmd/eot@latest
```

### 5.2 命令总览

```text
eot                        # 根命令
├── run                    # 发起一次 EoT 会话
├── list-topologies        # 查看可用拓扑
└── version                # 打印版本
```

### 5.3 `eot run` 常见用法

```bash
# A. 最简：两个内联 agent + debate
eot run \
  --question "2 + 2 * 3 = ?" \
  --topo debate \
  --agent "Planner:You are a careful math planner." \
  --agent "Checker:You verify arithmetic rigorously."

# B. 从文件读 agents，凭证走 CLI flag（比 env 更明确）
eot run \
  --question-file ./problem.txt \
  --topo report --central Reviewer \
  --agents-file ./examples/agents.json \
  --api-base http://your-gateway --api-key sk-xxx --model gpt-4o-mini \
  --rounds 3 --verbose

# C. 从 stdin 读问题 + JSON 输出，方便下游解析
echo "rank these 3 options ..." | eot run \
  --question-file - \
  --topo memory \
  --agent "Critic:You are critical." \
  --agent "Advocate:You are optimistic." \
  --output json

# D. 放宽收敛条件：多数同意即可（3 个里 2 个一致就停）
eot run -q "..." --topo debate \
  --agent "A:role A" --agent "B:role B" --agent "C:role C" \
  --threshold 0.66
```

### 5.4 重要 flag

| Flag | 说明 |
|------|------|
| `-q, --question` / `--question-file` | 问题文本；文件支持 `-` 即 stdin |
| `--topo` | `memory` / `report` / `relay` / `debate` |
| `--central` | `report` 拓扑下的中心 Agent ID |
| `--agent "ID:SYSTEM"` | 内联 agent，可重复；简单场景首选 |
| `--agents-file` | JSON 格式 agent 列表，复杂 prompt 用它 |
| `--api-base` / `--api-key` / `--model` | 覆盖环境变量 / 文件 |
| `--rounds` | 最大轮数（默认 3） |
| `--threshold` | 收敛阈值（`1.0` 完全一致，`0.5` 多数） |
| `-v, --verbose` | 实时打印每个 Thought |
| `-o, --output` | `text`（默认汇总）/ `json`（含完整 transcript） |

### 5.5 JSON 输出结构

```json
{
  "topology": "debate",
  "rounds": 1,
  "converged": true,
  "final_answer": "577",
  "thoughts": [
    {"agent": "Planner", "round": 0, "content": "...", "answer": "577"},
    {"agent": "Calculator", "round": 0, "content": "...", "answer": "577"},
    {"agent": "Reviewer", "round": 0, "content": "...", "answer": "577"}
  ]
}
```

适合被脚本 / 其他程序消费。

---

## 6. 使用技巧

1. **Answer 约定**：Agent prompt 里已内置"最后一行用 `#### <答案>` 标记"的指令，这是收敛判断的基础。若任务没有明确最终答案（比如创意写作），把 `--threshold` 设为 `0` 相当于关闭收敛检测，只跑满 `--rounds` 轮。
2. **Topology 选型经验**
   - 发散 / 脑暴：`memory` 或 `debate`
   - 需要审核员：`report --central ReviewerID`
   - 流水线精修（草稿→润色→校对）：`relay`
3. **温度策略**：计算/校验型 Agent 用低温（0.1–0.3）保证稳定；创意/规划型用常规温度（0.7）。
4. **成本控制**
   - `--rounds 2`、`--threshold 0.5` 能显著省钱；
   - 收敛提前退出在 `debate` 下特别明显，因为多个模型往往第 1 轮就一致。
5. **调试**
   - 开发期用 `--verbose` 看每轮思维；
   - 上线用 `--output json` 做结构化落盘；
   - 库用法里传 `OnThought` 回调接日志/tracing。
6. **接入其他编排框架**：`eot.Run` 是一个纯函数，直接包一层即可嵌入 AutoGen / LangGraph / 自家工作流，作为"多 Agent 集体推理"节点使用。
7. **安全**：不要把真实 `api_key` 长期留在 `ReadMe.md` 里（当前仓库只是便于联调）；发布前改用 `.env` + `.gitignore`，或走 CI 密钥注入。

---

## 7. 开发

```bash
go build ./...       # 编译全部
go vet ./...         # 静态检查
go run ./cmd/eot --help
go run ./examples/library_usage
```

---

## 8. 本机联调配置（开发期便利）

推荐用项目根目录下的 `.env` 文件（别忘了加到 `.gitignore`）：

```dotenv
# .env
EOT_API_BASE=http://your-llm-gateway
EOT_API_KEY=sk-xxxxxxxx
EOT_MODEL=gpt-4o-mini
```

`LoadConfig` 也可以从 `ReadMe.md` / `README.md` 读 `KEY=VALUE`，但**不要把真实 key 提交到仓库**。发布前请保持本仓库的 ReadMe 不含任何真实凭证。
