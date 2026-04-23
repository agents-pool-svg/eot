# EoT 快速上手（1 分钟版）

> 详细设计与全部配置见 [`ReadMe.md`](./ReadMe.md)。本文档只讲**怎么用起来**。

EoT 有两种使用方式，按需选一种即可：

| 使用方式 | 适合谁 |
|---------|--------|
| **CLI** `eot run ...` | 想立刻试一下、接入 shell pipeline、做对比实验 |
| **Go SDK** `eot.Run(...)` | 想把多 Agent 推理嵌入自己的 Go 程序/服务 |

---

## 0. 准备凭证（两种方式共用）

任选一种方式告诉 EoT 怎么访问 LLM：

```bash
# 方式 A：环境变量（推荐）
export EOT_API_BASE="http://ai.xxx.com"
export EOT_API_KEY="api-key-xxxxxxxx"
export EOT_MODEL="gpt-4o-mini"     # 可选
```

```bash
# 方式 B：项目根目录下的 .env，写 KEY=VALUE
cat >> .env <<'EOF'
EOT_API_BASE=http://your-llm-gateway
EOT_API_KEY=sk-xxxxxxxx
EOT_MODEL=gpt-4o-mini
EOF
# ⚠️ 记得把真实 key 填上，并把 .env 加入 .gitignore
```

> CLI 也支持用 `--api-base` / `--api-key` / `--model` 直接覆盖；SDK 用 `eot.WithAPIBase(...)` 覆盖。
> 兼容别名：`CODEMATRIX_LLM_API_*`、`OPENAI_*`。

---

## 1. CLI 模式

### 安装 / 构建

推荐按顺序由易到难尝试：

```bash
# 方式 A（最快）：直接从 GitHub 安装到 $GOBIN，一行搞定
go install github.com/agents-pool-svg/eot/cmd/eot@latest

# 确保 $GOBIN 在 PATH 里（一次性）
grep -q 'go/bin' ~/.zshrc || echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc

eot version        # 输出: eot 0.1.0
```

```bash
# 方式 B：克隆仓库后本地构建（想改代码时用）
git clone https://github.com/agents-pool-svg/eot.git
cd eot
go build -o bin/eot ./cmd/eot
./bin/eot version
```

```bash
# 方式 C：下载预编译二进制（如有 Release 时）
# 见 https://github.com/agents-pool-svg/eot/releases
```

> 之后的示例统一用 `eot` 命令；如果你用方式 B 且没安装到 PATH，把 `eot` 换成 `./bin/eot` 即可。

### 第一条命令（30 秒见效果）

```bash
eot run \
  -q "17 * 23 = ?" \
  --topo debate \
  --agent "Drafter:You compute the product step by step." \
  --agent "Checker:You verify arithmetic and correct mistakes."
```

期望输出：
```
============================================================
Topology      : debate
Rounds used   : 1
Converged     : true
Final answer  : 391
============================================================
```

### 常用用法速查

```bash
# 查看所有拓扑
eot run --help
eot list-topologies

# 从文件读问题 + JSON 输出（便于脚本处理）
cat problem.txt | eot run --question-file - --topo memory \
  --agent "Critic:..." --agent "Advocate:..." -o json

# 复杂 agents 定义写到 JSON 文件
eot run -q "..." --topo report --central Reviewer \
  --agents-file examples/agents.json --rounds 3 -v

# 放宽收敛条件（3 人 2 人同意就停，省 token）
eot run -q "..." --topo debate \
  --agent "A:..." --agent "B:..." --agent "C:..." --threshold 0.66
```

### CLI flag 速记

| Flag | 作用 |
|------|------|
| `-q, --question` / `--question-file`（`-` = stdin） | 问题来源 |
| `--topo` | `memory` / `report` / `relay` / `debate` |
| `--central` | `report` 拓扑下的中心 Agent ID |
| `--agent "ID:SYSTEM"`（可重复）/ `--agents-file` | 定义 Agent |
| `--rounds` / `--threshold` | 最大轮数 / 收敛阈值 |
| `-v, --verbose` / `-o json` | 流式打印 / JSON 输出 |
| `--api-base` / `--api-key` / `--model` | 覆盖凭证 |

---

## 2. Go SDK 模式

### 安装

```bash
go get github.com/agents-pool-svg/eot@latest
```

### 最小可跑示例（30 行）

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
        Question: "17 * 23 = ?",
        Agents: []eot.AgentSpec{
            {ID: "Drafter", System: "You compute products step by step."},
            {ID: "Checker", System: "You verify arithmetic rigorously.", Temperature: 0.2},
        },
        Topology:  eot.TopologySpec{Name: "debate"},
        MaxRounds: 3,
        // 不传 ConfigOpts 时，自动从 env / .env / ReadMe.md 加载
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("converged=%v rounds=%d answer=%s\n",
        res.Converged, res.Rounds, res.FinalAnswer)
}
```

运行：
```bash
go run .
```

### 显式传凭证 / 接回调（生产场景常用）

```go
res, err := eot.Run(ctx, eot.RunRequest{
    Question: "...",
    Agents:   []eot.AgentSpec{ /* ... */ },
    Topology: eot.TopologySpec{Name: "report", Central: "Reviewer"},
    MaxRounds:            3,
    ConvergenceThreshold: 0.66,   // 多数同意即停
    ConfigOpts: []eot.ConfigOption{
        eot.WithAPIBase("https://api.openai.com"),
        eot.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        eot.WithModel("gpt-4o-mini"),
    },
    OnThought: func(t *eot.Thought) {
        // 接 WebSocket 推送 / 日志 / tracing
        log.Printf("[%s r%d] %.60s...", t.AgentID, t.Round, t.Content)
    },
})
```

### 核心 API 速记

| 符号 | 作用 |
|------|------|
| `eot.Run(ctx, RunRequest)` | **一站式入口**，90% 场景用这个 |
| `eot.LoadConfig(opts...)` + `eot.NewLLMClient(cfg)` | 手动组装，便于复用与测试 |
| `eot.AgentSpec` / `eot.NewAgent` | 定义 Agent |
| `eot.TopologySpec{Name, Central}` / `eot.BuildTopology` | 选/造拓扑 |
| `eot.Runner{Agents, Topology, MaxRounds, OnThought}` + `.Run(ctx, q)` | 底层 Runner，用于自定义 Topology |
| `eot.Result.{FinalAnswer, Rounds, Converged, Thoughts, Transcript()}` | 结果 |

---

## 3. 四种 Topology 选哪个？

| 拓扑 | 一句话 | 什么时候用 |
|------|-------|-----------|
| `memory`  | 群聊：都看所有历史 | 脑暴、开放讨论 |
| `debate`  | 辩论：看所有对手上一轮 | 多答案收敛、交叉质疑（默认首选） |
| `relay`   | 接力：只看前一位 | 流水线精修（草稿→润色→校对） |
| `report`  | 汇报：外围 → 中心 | 有明确审核者的场景 |

---

## 4. 三句话规则

1. **Agent 的最终答案必须用 `#### <答案>` 结尾**（内置 prompt 已包含），否则收敛检测失效——这时把 `--threshold 0` 关掉收敛即可。
2. **计算/校验 Agent 用低温（0.1–0.3），规划/创意用 0.7**。
3. **想省钱**：先 `--rounds 2 --threshold 0.66`，基本第 1 轮就收敛。

---

## 5. 遇到问题

| 现象 | 原因 / 解决 |
|------|------------|
| `missing LLM credentials` | 检查 env/.env/ReadMe.md，或用 `--api-base` / `--api-key` 显式传 |
| `EoT requires at least 2 agents` | 至少要 2 个 `--agent` 或 `AgentSpec` |
| `Converged: false` | Agent 答案不一致；看 `-v` 或 JSON 输出的 transcript，必要时加轮数或调低 `threshold` |
| CLI 没反应 / 超时 | 检查 `--api-base` 是否可达；HTTP 超时默认 60s，可在 SDK 里覆盖 `LLMClient.HTTP` |

更多使用技巧与设计细节见 [`ReadMe.md`](./ReadMe.md)。
