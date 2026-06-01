# MyMeetily — 本地离线 AI 会议助手

> 录音 → 实时字幕 → 语音转文字 → 自动生成会议纪要，全程离线，无需联网。

<p align="center">
  <strong>🖥️  Windows 桌面应用  · Go + 🎙️ whisper.cpp + 🦙 Ollama  ·  🔒 纯本地运行</strong>
</p>

---

## 与 Meetily 的区别

[Meetily](https://github.com/Zackriya-Solutions/meetily) 是这个领域最知名的开源项目（11k+ Stars），MyMeetily 借鉴了它的理念，但走了不同的技术路线：

| | Meetily | MyMeetily |
|---|---|---|
| 技术栈 | Tauri (Rust) + Next.js (Web 前端) | **纯 Go** + Bubble Tea (终端原生) |
| 运行环境 | 需 WebView / 浏览器 | **终端即可**，无需图形环境 |
| 安装包体积 | ~150MB+ | **~21MB** (不含模型) |
| 内存占用 | 中等 (Chromium 内核) | **极低** (纯终端) |
| 跨平台 | macOS / Windows / Linux | Windows（Go 跨平台编译可行） |
| LLM 后端 | Ollama / Claude / Groq / 多选 | **Ollama**（简单够用） |
| GPU 加速 | Metal / CUDA / Vulkan | **CUDA** |
| 中文体验 | 通用多语言 | **中文优先**（qwen 模型 + zh 调优） |
| 实时字幕 | 支持 | 支持（能量 VAD + whisper 逐句） |
| 付费模式 | 社区版免费 + Pro $10/月 | **完全免费**，无 Pro 版 |
| 设计理念 | 功能全面，企业级 | **极致轻量**，一条命令跑起来 |

**一句话：Meetily 是大而全的瑞士军刀，MyMeetily 是开箱即用的快刀。** 如果你只需要 Windows 上轻量、快速地把会议录下来转成纪要，MyMeetily 更合适。

---

## 为什么选择 MyMeetily？

|          | MyMeetily                | 在线会议工具 | 其他转写软件 |
| -------- | ------------------------ | ------------ | ------------ |
| 隐私     | ✅ 所有数据留在本机       | ❌ 音频上传云端 | ⚠️ 取决于厂商 |
| 费用     | ✅ 完全免费               | ❌ 付费订阅   | ⚠️ 按小时收费 |
| 网络     | ✅ 离线可用               | ❌ 需要联网   | ⚠️ 部分离线   |
| 中文优化 | ✅ whisper + qwen 专优    | ⚠️ 通用模型   | ⚠️ 参差不齐   |
| 可控性   | ✅ 开源可定制             | ❌ 封闭       | ❌ 封闭       |

---

## 核心功能

### 一键录音

- 自动检测麦克风和系统音频设备（WASAPI）
- 同时录制**你的声音 + 对方的声音**（系统音频回环）
- 录音文件自动存入时间戳文件夹，管理清晰

### 实时字幕

- 录音时实时显示转写字幕，延迟仅 1-3 秒
- 基于能量检测（VAD）自动切句
- 音量条随声音大小跳动，静音不动

### AI 自动整理

- 录音结束自动触发处理流水线：
  ```
  音频预处理 → whisper.cpp 语音识别 → Ollama LLM 整理 → 生成会议纪要
  ```
- 处理过程带进度条，每个阶段清晰可见

### 会议纪要输出

每次录音生成 3 个文件，放在同一个文件夹：

```
output/20260601_143052/
├── live_20260601_143052.wav      ← 原始录音
├── live_20260601_143052_转写.txt  ← 逐字转写
└── live_20260601_143052_纪要.html ← 结构化会议纪要
```

---

## 界面预览

```
┌─────────────────────────────────────────────┐
│               MyMeetily                     │
│         本地离线 AI 会议助手                   │
│                                             │
│   语音模型    Whisper Large-v3 Turbo Q5_0     │
│   引擎        whisper.cpp + Ollama           │
│   整理模型    Ollama 本机服务                  │
│   输出格式    HTML + TXT                      │
│                                              │
│           实时录音转录                         │
│                                              │
│        按任意键进入主菜单 . . .                 │
└─────────────────────────────────────────────┘

  主菜单：                         录音中：
  ▸ 实时录音                       ⏱  00:03:25
    退出                          ● 录制中
                                    ▅▆▇█▇▆▅▆▇█▅▆▇▇▆▅▄▅▆▇
  处理中：                          音量: 72%
  ◍ Eino Agent 正在执行管道...
                                    实时字幕
  [████████████░░░░░░░░░░] 2/4       大家好今天我们来讨论
    语音识别中...                     项目进度的问题
                                     我建议下周五之前完成
  ✓ 音频预处理 → ● 语音识别            …
    → ○ LLM 整理 → ○ 生成输出
```

---

## 技术架构

```
┌──────────────────────────────────────────────────┐
│                   MyMeetily                       │
├──────────┬──────────┬──────────┬────────────────┤
│  WASAPI  │ whisper  │  Ollama  │   Bubble Tea   │
│  音频采集 │  .cpp    │  LLM     │   TUI 界面     │
│  (Go)    │  语音识别 │  文本整理 │   (Go)         │
├──────────┴──────────┴──────────┴────────────────┤
│             CloudWeGo Eino Agent Graph           │
│        AudioPreprocess → ASR → Summarize →       │
│                     FormatOutput                 │
└──────────────────────────────────────────────────┘
```

| 模块     | 技术选型              | 说明                            |
| -------- | --------------------- | ------------------------------- |
| 音频采集 | WASAPI (go-wca)       | Windows 原生 API，零外部依赖    |
| 语音识别 | whisper.cpp (CUDA)    | large-v3-turbo q5_0，~548MB     |
| 文本整理 | Ollama HTTP API       | qwen2.5:1.5b-instruct（可替换） |
| 流水线   | CloudWeGo Eino        | 字节跳动 Go Agent 框架          |
| 界面     | Bubble Tea + Lipgloss | Go TUI 框架，终端原生体验       |
| 实时字幕 | 能量 VAD + whisper    | 逐句切割，异步转写              |

---

## 快速开始

### 前置条件

- Windows 10/11
- [Ollama](https://ollama.com/download/windows)（需自行安装）
- [Go](https://go.dev/dl/) 1.21+
- NVIDIA 显卡（可选，CPU 也可运行但较慢）

### 安装步骤

```bash
# 1. 克隆项目
git clone https://github.com/<your-username>/mymeetily.git
cd mymeetily

# 2. 下载依赖
go mod download

# 3. 下载 whisper.cpp 引擎
go run . init-engine

# 4. 下载模型文件（whisper + Ollama）
go run . init-model

# 5. 启动
go run .
```

### 编译打包

```bash
# 编译
go build -ldflags="-s -w" -o mymeetily.exe .

# 完整打包（含引擎 + DLL + 配置）
make package
```

---

## 适用场景

- 远程会议：同时录制你和对方的声音，自动生成纪要
- 面试记录：实时字幕辅助，事后可回溯原文
- 课堂笔记：录下讲课内容，一键转文字
- 头脑风暴：自由讨论不中断，事后 AI 提炼要点
- 个人备忘：口述想法，AI 整理成结构化笔记

---

## 配置

编辑 `configs/config.yaml`：

```yaml
asr:
  whisper_binary: "llmengine/whispercpp/Release/whisper-cli"
  model_path: "./llmmodels/whisper/ggml-large-v3-turbo-q5_0.bin"
  language: "zh"

ollama:
  endpoint: "http://127.0.0.1:11434"
  model: "qwen2.5:1.5b-instruct"       # 可换更大模型
  temperature: 0.3
  max_tokens: 4096

audio:
  backend: "wasapi"
  sample_rate: 16000
  channels: 1

output:
  output_dir: "./output"
```

---

## 开发

```bash
go run .                    # 运行
go test ./internal/.. -v    # 测试
go vet ./...                # 代码检查
```

---

## 开源协议

MIT License

---

<p align="center">
  <sub>你的会议数据，只属于你。</sub>
</p>
