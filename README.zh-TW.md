# ⌬ OpenCode

[English](README.md)

> **⚠️ 開發早期注意事項：** 本專案仍處於早期開發階段，尚未適用於正式環境。功能可能隨時變動、中斷或尚未完整。請自行承擔使用風險。

一個強大的終端機 AI 助手，為開發者提供直接在終端機中的智慧編程協助。

## 概述

OpenCode 是一個以 Go 語言開發的 CLI 應用程式，將 AI 助手帶入您的終端機。它提供 TUI（終端機使用者介面），可與多種 AI 模型互動，協助程式設計、除錯等任務。

## 功能特色

- **互動式 TUI**：採用 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 打造流暢的終端體驗
- **多家 AI 供應商支援**：支援 OpenAI、Anthropic Claude、Google Gemini、AWS Bedrock、Groq、Azure OpenAI 及 OpenRouter
- **會話管理**：可儲存並管理多個對話會話
- **工具整合**：AI 可執行指令、搜尋檔案並修改程式碼
- **類 Vim 編輯器**：內建文字輸入編輯器
- **持久化儲存**：使用 SQLite 資料庫儲存對話與會話
- **LSP 整合**：支援 Language Server Protocol 提供程式碼智慧
- **檔案變更追蹤**：於會話期間追蹤並視覺化檔案變更
- **外部編輯器支援**：可開啟您偏好的編輯器撰寫訊息

## 安裝方式

### 使用安裝腳本

```bash
# 安裝最新版
curl -fsSL https://opencode.ai/install | bash

# 安裝指定版本
curl -fsSL https://opencode.ai/install | VERSION=0.1.0 bash
```

### 使用 Homebrew（macOS 與 Linux）

```bash
brew install opencode-ai/tap/opencode
```

### 使用 AUR（Arch Linux）

```bash
# 使用 yay
yay -S opencode-bin

# 使用 paru
paru -S opencode-bin
```

### 使用 Go

```bash
go install github.com/opencode-ai/opencode@latest
```

## 設定方式

OpenCode 會在以下位置尋找設定檔：

- `$HOME/.opencode.json`
- `$XDG_CONFIG_HOME/opencode/.opencode.json`
- `./.opencode.json`（本地目錄）

### 環境變數

您可以透過環境變數設定 OpenCode：

| 環境變數名稱               | 用途                                             |
| -------------------------- | ------------------------------------------------ |
| `ANTHROPIC_API_KEY`        | 用於 Claude 模型                                 |
| `OPENAI_API_KEY`           | 用於 OpenAI 模型                                 |
| `GEMINI_API_KEY`           | 用於 Google Gemini 模型                          |
| `GROQ_API_KEY`             | 用於 Groq 模型                                   |
| `AWS_ACCESS_KEY_ID`        | 用於 AWS Bedrock (Claude)                        |
| `AWS_SECRET_ACCESS_KEY`    | 用於 AWS Bedrock (Claude)                        |
| `AWS_REGION`               | 用於 AWS Bedrock (Claude)                        |
| `AZURE_OPENAI_ENDPOINT`    | 用於 Azure OpenAI 模型                           |
| `AZURE_OPENAI_API_KEY`     | 用於 Azure OpenAI 模型（使用 Entra ID 時可選填） |
| `AZURE_OPENAI_API_VERSION` | 用於 Azure OpenAI 模型                           |

### 設定檔結構範例

```json
{
  "data": {
    "directory": ".opencode"
  },
  "providers": {
    "openai": {
      "apiKey": "your-api-key",
      "disabled": false
    },
    "anthropic": {
      "apiKey": "your-api-key",
      "disabled": false
    },
    "groq": {
      "apiKey": "your-api-key",
      "disabled": false
    },
    "openrouter": {
      "apiKey": "your-api-key",
      "disabled": false
    }
  },
  "agents": {
    "coder": {
      "model": "claude-3.7-sonnet",
      "maxTokens": 5000
    },
    "task": {
      "model": "claude-3.7-sonnet",
      "maxTokens": 5000
    },
    "title": {
      "model": "claude-3.7-sonnet",
      "maxTokens": 80
    }
  },
  "mcpServers": {
    "example": {
      "type": "stdio",
      "command": "path/to/mcp-server",
      "env": [],
      "args": []
    }
  },
  "lsp": {
    "go": {
      "disabled": false,
      "command": "gopls"
    }
  },
  "debug": false,
  "debugLSP": false
}
```

## 支援的 AI 模型

OpenCode 支援多家供應商的多種 AI 模型：

### OpenAI

- GPT-4.1 系列（gpt-4.1, gpt-4.1-mini, gpt-4.1-nano）
- GPT-4.5 Preview
- GPT-4o 系列（gpt-4o, gpt-4o-mini）
- O1 系列（o1, o1-pro, o1-mini）
- O3 系列（o3, o3-mini）
- O4 Mini

### Anthropic

- Claude 3.5 Sonnet
- Claude 3.5 Haiku
- Claude 3.7 Sonnet
- Claude 3 Haiku
- Claude 3 Opus

### Google

- Gemini 2.5
- Gemini 2.5 Flash
- Gemini 2.0 Flash
- Gemini 2.0 Flash Lite

### AWS Bedrock

- Claude 3.7 Sonnet

### Groq

- Llama 4 Maverick (17b-128e-instruct)
- Llama 4 Scout (17b-16e-instruct)
- QWEN QWQ-32b
- Deepseek R1 distill Llama 70b
- Llama 3.3 70b Versatile

### Azure OpenAI

- GPT-4.1 系列（gpt-4.1, gpt-4.1-mini, gpt-4.1-nano）
- GPT-4.5 Preview
- GPT-4o 系列（gpt-4o, gpt-4o-mini）
- O1 系列（o1, o1-mini）
- O3 系列（o3, o3-mini）
- O4 Mini

## 使用方式

```bash
# 啟動 OpenCode
opencode

# 以除錯模式啟動
opencode -d

# 指定工作目錄啟動
opencode -c /path/to/project
```

## 指令列參數

| 參數      | 短參數 | 說明             |
| --------- | ------ | ---------------- |
| `--help`  | `-h`   | 顯示說明資訊     |
| `--debug` | `-d`   | 啟用除錯模式     |
| `--cwd`   | `-c`   | 設定目前工作目錄 |

## 鍵盤快捷鍵

### 全域快捷鍵

| 快捷鍵   | 動作                              |
| -------- | --------------------------------- |
| `Ctrl+C` | 離開應用程式                      |
| `Ctrl+?` | 切換說明視窗                      |
| `?`      | 切換說明視窗（非編輯模式下）      |
| `Ctrl+L` | 檢視日誌                          |
| `Ctrl+A` | 切換會話                          |
| `Ctrl+K` | 指令對話框                        |
| `Ctrl+O` | 切換模型選擇對話框                |
| `Esc`    | 關閉目前覆蓋/對話框或返回前一模式 |

### 聊天頁面快捷鍵

| 快捷鍵   | 動作                       |
| -------- | -------------------------- |
| `Ctrl+N` | 建立新會話                 |
| `Ctrl+X` | 取消目前操作/生成          |
| `i`      | 聚焦編輯器（非編寫模式下） |
| `Esc`    | 離開編寫模式並聚焦訊息     |

### 編輯器快捷鍵

| 快捷鍵              | 動作                       |
| ------------------- | -------------------------- |
| `Ctrl+S`            | 傳送訊息（編輯器聚焦時）   |
| `Enter` 或 `Ctrl+S` | 傳送訊息（編輯器未聚焦時） |
| `Ctrl+E`            | 開啟外部編輯器             |
| `Esc`               | 取消編輯器聚焦並聚焦訊息   |

### 會話對話框快捷鍵

| 快捷鍵     | 動作       |
| ---------- | ---------- |
| `↑` 或 `k` | 上一個會話 |
| `↓` 或 `j` | 下一個會話 |
| `Enter`    | 選擇會話   |
| `Esc`      | 關閉對話框 |

### 模型對話框快捷鍵

| 快捷鍵     | 動作         |
| ---------- | ------------ |
| `↑` 或 `k` | 上移         |
| `↓` 或 `j` | 下移         |
| `←` 或 `h` | 上一個供應商 |
| `→` 或 `l` | 下一個供應商 |
| `Esc`      | 關閉對話框   |

### 權限對話框快捷鍵

| 快捷鍵                  | 動作             |
| ----------------------- | ---------------- |
| `←` 或 `left`           | 左切換選項       |
| `→` 或 `right` 或 `tab` | 右切換選項       |
| `Enter` 或 `space`      | 確認選擇         |
| `a`                     | 允許權限         |
| `A`                     | 本次會話允許權限 |
| `d`                     | 拒絕權限         |

### 日誌頁面快捷鍵

| 快捷鍵             | 動作         |
| ------------------ | ------------ |
| `Backspace` 或 `q` | 返回聊天頁面 |

## AI 助手工具

OpenCode 的 AI 助手可使用多種工具協助程式開發：

### 檔案與程式碼工具

| 工具名稱      | 說明           | 參數                                                                         |
| ------------- | -------------- | ---------------------------------------------------------------------------- |
| `glob`        | 依模式尋找檔案 | `pattern`（必填），`path`（選填）                                            |
| `grep`        | 搜尋檔案內容   | `pattern`（必填），`path`（選填），`include`（選填），`literal_text`（選填） |
| `ls`          | 列出目錄內容   | `path`（選填），`ignore`（選填，模式陣列）                                   |
| `view`        | 檢視檔案內容   | `file_path`（必填），`offset`（選填），`limit`（選填）                       |
| `write`       | 寫入檔案       | `file_path`（必填），`content`（必填）                                       |
| `edit`        | 編輯檔案       | 多種檔案編輯參數                                                             |
| `patch`       | 套用檔案修補   | `file_path`（必填），`diff`（必填）                                          |
| `diagnostics` | 取得診斷資訊   | `file_path`（選填）                                                          |

### 其他工具

| 工具名稱      | 說明                   | 參數                                                                          |
| ------------- | ---------------------- | ----------------------------------------------------------------------------- |
| `bash`        | 執行 shell 指令        | `command`（必填），`timeout`（選填）                                          |
| `fetch`       | 從 URL 取得資料        | `url`（必填），`format`（必填），`timeout`（選填）                            |
| `sourcegraph` | 搜尋公開程式庫程式碼   | `query`（必填），`count`（選填），`context_window`（選填），`timeout`（選填） |
| `agent`       | 以 AI agent 執行子任務 | `prompt`（必填）                                                              |

## 架構說明

OpenCode 採用模組化架構：

- **cmd**：使用 Cobra 的命令列介面
- **internal/app**：核心應用服務
- **internal/config**：設定管理
- **internal/db**：資料庫操作與遷移
- **internal/llm**：LLM 供應商與工具整合
- **internal/tui**：終端 UI 元件與版面
- **internal/logging**：日誌基礎設施
- **internal/message**：訊息處理
- **internal/session**：會話管理
- **internal/lsp**：Language Server Protocol 整合

## MCP（Model Context Protocol）

OpenCode 實作 Model Context Protocol（MCP），可透過外部工具擴充功能。MCP 提供標準化方式讓 AI 助手與外部服務及工具互動。

### MCP 特色

- **外部工具整合**：透過標準協定連接外部工具與服務
- **工具自動發現**：自動從 MCP 伺服器發現可用工具
- **多種連線型態**：
  - **Stdio**：透過標準輸入/輸出通訊
  - **SSE**：透過 Server-Sent Events 通訊
- **安全性**：權限系統控管 MCP 工具存取

### MCP 伺服器設定

MCP 伺服器於設定檔 `mcpServers` 區段定義：

```json
{
  "mcpServers": {
    "example": {
      "type": "stdio",
      "command": "path/to/mcp-server",
      "env": [],
      "args": []
    },
    "web-example": {
      "type": "sse",
      "url": "https://example.com/mcp",
      "headers": {
        "Authorization": "Bearer token"
      }
    }
  }
}
```

### MCP 工具使用

設定完成後，MCP 工具會自動與內建工具一同提供給 AI 助手。執行時會遵循相同的權限模型，需經使用者同意。

## LSP（Language Server Protocol）

OpenCode 整合 Language Server Protocol，提供多語言程式碼智慧功能。

### LSP 特色

- **多語言支援**：可連接多種語言伺服器
- **診斷功能**：即時錯誤檢查與 lint
- **檔案監控**：自動通知語言伺服器檔案變更

### LSP 設定

語言伺服器於設定檔 `lsp` 區段設定：

```json
{
  "lsp": {
    "go": {
      "disabled": false,
      "command": "gopls"
    },
    "typescript": {
      "disabled": false,
      "command": "typescript-language-server",
      "args": ["--stdio"]
    }
  }
}
```

### LSP 與 AI 整合

AI 助手可透過 `diagnostics` 工具存取 LSP 功能，包含：

- 檢查程式碼錯誤
- 根據診斷建議修正

目前 LSP 客戶端已支援完整 LSP 協定（如補全、hover、定義等），但目前僅開放診斷功能給 AI 助手。

## 開發說明

### 先決條件

- Go 1.24.0 或以上

### 原始碼建置

```bash
# 下載原始碼
git clone https://github.com/opencode-ai/opencode.git
cd opencode

# 編譯
go build -o opencode

# 執行
./opencode
```

## 致謝

OpenCode 感謝以下關鍵人物的貢獻與支持：

- [@isaacphi](https://github.com/isaacphi) - [mcp-language-server](https://github.com/isaacphi/mcp-language-server) 專案，為 LSP 客戶端實作提供基礎
- [@adamdottv](https://github.com/adamdottv) - 設計方向與 UI/UX 架構

特別感謝廣大的開源社群，讓本專案得以實現。

## 授權

OpenCode 採用 MIT 授權。詳見 [LICENSE](LICENSE) 檔案。

## 貢獻方式

歡迎貢獻！參與方式如下：

1. Fork 本儲存庫
2. 建立功能分支（`git checkout -b feature/amazing-feature`）
3. 提交變更（`git commit -m 'Add some amazing feature'`）
4. 推送分支（`git push origin feature/amazing-feature`）
5. 建立 Pull Request

請務必適當更新測試並遵循現有程式風格。
