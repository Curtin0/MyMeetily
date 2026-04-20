# 音视频转录工作指南（AI执行版）

> 使用 OpenAI Whisper 进行本地音频转录的完整流程
> 
> 本文档面向 AI 助手，用于指导自动化音频转录任务

---

## 执行原则

1. **优先检查现有工具**：执行前先检查环境中是否已有 Python、ffmpeg、Whisper，有则直接使用，无需重复安装
2. **首次部署先用 CPU 版本验证流程**：PyTorch CPU 版本约 115MB，下载快、可快速验证转录流程是否走通；GPU 版本约 1.9GB，下载耗时长，首次部署避免在此卡住
3. **确认流程通后再升级 GPU**：CPU 版本验证无误后，有 NVIDIA 显卡时再安装 PyTorch CUDA 版本加速转录
4. **禁止使用官方 pip 源**：国内网络环境下载 PyTorch 等大包必失败，必须使用国内镜像源
5. **优先用 curl/aria2c 下载大文件**：pip 下载 PyTorch CUDA whl（约1.9GB）速度慢且容易超时中断，应先 curl/aria2c 下载到本地再 pip 本地安装
6. **后台执行长任务**：转录过程在后台运行，终端提供简单进度提示
7. **最小化用户干扰**：减少向用户询问，自动完成环境准备和转录

---

## 第一步：环境检查与准备

### 1.1 检查 Python

```bash
# 检查 Python 是否已安装
python --version 2>/dev/null || python3 --version 2>/dev/null

# 如果未安装，使用 winget 安装
winget install Python.Python.3.11 --silent --accept-package-agreements --accept-source-agreements

# 注意：其他 AI 工具可能自带 Python 311 或 312，无需重复安装
# 如果 AI 检测到的 Python 版本不同，在下方替换 Python311 为实际版本
PYTHON_VER="311"  # 根据实际版本修改此处
PYTHON_PATH="/c/Users/$USERNAME/AppData/Local/Programs/Python/Python${PYTHON_VER}"
PYTHON_SCRIPTS="$PYTHON_PATH/Scripts"
```

### 1.2 检查并配置 pip 国内镜像源

```bash
# 检查当前 pip 源
pip config get global.index-url 2>/dev/null

# 如果返回官方源(pypi.org)或为空，则切换为阿里云镜像
# 官方源在国内 100% 会导致下载超时失败，必须配置国内镜像
CURRENT_MIRROR=$(pip config get global.index-url 2>/dev/null || echo "")
if [[ "$CURRENT_MIRROR" != *"aliyun"* ]] && [[ "$CURRENT_MIRROR" != *"tsinghua"* ]] && [[ "$CURRENT_MIRROR" != *"tuna"* ]] && [[ "$CURRENT_MIRROR" != *"ustc"* ]]; then
    # 未配置国内镜像，设置为阿里云
    pip config set global.index-url https://mirrors.aliyun.com/pypi/simple/
    pip config set global.trusted-host mirrors.aliyun.com
    echo "已切换 pip 源为阿里云镜像"
else
    echo "pip 已使用国内镜像: $CURRENT_MIRROR"
fi
```

> **可用国内镜像源**：
> - 阿里云：`https://mirrors.aliyun.com/pypi/simple/`
> - 清华：`https://pypi.tuna.tsinghua.edu.cn/simple/`
> - 中科大：`https://pypi.mirrors.ustc.edu.cn/simple/`

### 1.3 检查 ffmpeg

```bash
# 检查 ffmpeg 是否已安装
which ffmpeg 2>/dev/null || where ffmpeg 2>/dev/null || find /c -name "ffmpeg.exe" 2>/dev/null | head -1

# 如果未安装，下载解压到 /tmp
if [ ! -f /tmp/ffmpeg-*/bin/ffmpeg.exe ]; then
    curl -L "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip" -o /tmp/ffmpeg.zip
    unzip -q /tmp/ffmpeg.zip -d /tmp/
fi

# 设置 ffmpeg 路径变量
FFMPEG_PATH=$(find /tmp -name "ffmpeg.exe" -path "*essentials*" 2>/dev/null | head -1 | xargs dirname)
```

> **下载卡住的备选方案**：如果 gyan.dev 下载困难，可手动从以下任一地址下载后放到 `/tmp/ffmpeg/` 目录：
> - 阿里云盘分享（推荐国内用户）
> - SourceForge：`https://sourceforge.net/projects/ffmpeg-windows-builds/files/`
> - 官网：`https://ffmpeg.org/download.html`

### 1.4 检查 Whisper

```bash
# 检查 Whisper 是否已安装（使用实际 Python 版本）
export PATH="/c/Users/\$USERNAME/AppData/Local/Programs/Python/Python${PYTHON_VER}:/c/Users/\$USERNAME/AppData/Local/Programs/Python/Python${PYTHON_VER}/Scripts:\$PATH"
whisper --version 2>/dev/null || python -m pip install -U openai-whisper
```

---

## 第二步：GPU 加速配置（优先执行）

### 2.1 检查 NVIDIA GPU

```bash
# 检查是否有 NVIDIA 显卡
nvidia-smi 2>/dev/null && echo "GPU detected" || echo "No GPU"

# 获取 GPU 型号和 CUDA 版本
nvidia-smi --query-gpu=name,driver_version --format=csv,noheader
```

### 2.2 安装 PyTorch（两步走策略）

> **推荐策略：先 CPU 后 GPU**
> - PyTorch CPU 版本约 115MB，下载快（1-2分钟），适合首次快速验证转录流程
> - PyTorch CUDA 版本约 1.9GB，下载耗时长（15-30分钟），首次部署不建议在此卡住
> - CPU 版本验证流程走通后，有 NVIDIA 显卡再升级 GPU 版本加速

#### 阶段一：安装 PyTorch CPU 版本（首次部署推荐）

```bash
# 如果之前装了任何版本的 PyTorch，先卸载
python -m pip uninstall torch torchvision torchaudio -y

# 安装 CPU 版本（从阿里云 PyPI 镜像，115MB，约1-2分钟）
python -m pip install torch torchvision torchaudio

# 验证安装成功
python -c "import torch; print(f'PyTorch: {torch.__version__}')"
```

#### 阶段二：升级 GPU 加速（可选，CPU 验证通过后执行）

如果检测到 NVIDIA GPU，且 CPU 版本验证转录流程正常，可升级为 GPU 版本。

> **重要经验**：PyTorch CUDA whl 文件约 1.9GB，通过 pip 直接下载速度很慢（约1.5MB/s），且容易超时中断。推荐做法是先用 curl 或 aria2c 下载 whl 文件到本地，再 pip 本地安装。

##### 步骤一：确定 CUDA 版本

```bash
# 查看 nvidia-smi 右上角的 CUDA Version，例如 CUDA 13.1
# 选择 <= 该版本的 PyTorch CUDA wheel，如 cu130 / cu126 / cu124
nvidia-smi
```

##### 步骤二：下载 PyTorch CUDA whl 文件

```bash
# 优先使用 aria2c（多线程，速度快），没有则用 curl
# 根据上一步的 CUDA 版本替换 cu130 为对应版本
# Python 版本查看：python --version 输出如 Python 3.11.4 -> cp311, Python 3.14 -> cp314
PYTHON_TAG="cp314"  # 根据实际 Python 版本修改（cp311/cp312/cp314）

# 方式1：aria2c（推荐，支持多线程断点续传）
aria2c -x 16 -s 16 -c -d /tmp -o torch_cu130.whl \
  "https://mirrors.aliyun.com/pytorch-wheels/cu130/torch-2.11.0%2Bcu130-${PYTHON_TAG}-${PYTHON_TAG}-win_amd64.whl"

# 方式2：curl（备选）
curl -L -C - -o /tmp/torch_cu130.whl \
  "https://mirrors.aliyun.com/pytorch-wheels/cu130/torch-2.11.0%2Bcu130-${PYTHON_TAG}-${PYTHON_TAG}-win_amd64.whl"

# 备用镜像：南京大学
# 将上方 URL 中的 mirrors.aliyun.com/pytorch-wheels 替换为 mirrors.nju.edu.cn/pytorch/whl
```

> **国内 PyTorch CUDA 镜像源**：
> - 阿里云：`https://mirrors.aliyun.com/pytorch-wheels/cu130/`
> - 南京大学：`https://mirrors.nju.edu.cn/pytorch/whl/cu130/`
> 
> 注意：`-f` 参数仅指定查找链接页面，不能用 `--index-url` 替代（阿里云 PyTorch 镜像不是标准 PyPI 索引）
> 
> 可用 CUDA 版本：cu124 / cu126 / cu130，选择 <= nvidia-smi 显示的 CUDA 版本

##### 步骤三：本地安装

```bash
# 如果之前装了 CPU 版本，先卸载
python -m pip uninstall torch torchvision torchaudio -y

# 本地安装下载好的 whl 文件
python -m pip install /tmp/torch_cu130.whl

# 安装配套的 torchvision 和 torchaudio（从阿里云 PyPI 镜像）
python -m pip install torchvision torchaudio -f https://mirrors.aliyun.com/pytorch-wheels/cu130

# 验证 CUDA 是否可用
python -c "import torch; print(f'CUDA可用: {torch.cuda.is_available()}'); print(f'GPU: {torch.cuda.get_device_name(0)}'); print(f'PyTorch: {torch.__version__}')"
```

> **如果 CUDA 不可用**：
> 1. 检查 PyTorch 版本中的 CUDA 版本是否 <= nvidia-smi 显示的 CUDA 版本
> 2. 检查是否误装了 CPU 版本（版本号含 `+cpu`）
> 3. 更新 NVIDIA 驱动
> 4. 降级使用更低版本 CUDA wheel（如 cu126）

### 2.3 设置设备参数

```bash
# 如果有 GPU，使用 --device cuda
# 如果只有 CPU，使用 --device cpu（或不指定，默认CPU）

if python -c "import torch; exit(0 if torch.cuda.is_available() else 1)" 2>/dev/null; then
    DEVICE="--device cuda"
    echo "使用 GPU 加速"
else
    DEVICE="--device cpu"
    echo "使用 CPU 转录"
fi
```

### 2.4 预下载 Whisper 模型（可选，避免首次转录时下载卡住）

Whisper 首次执行时会自动下载模型文件（约 244MB for small），如果网络慢可以在正式转录前提前下载：

```bash
# 下载 small 模型到本地缓存目录
python -c "import whisper; model = whisper.load_model('small')"

# 验证模型文件
ls -lh ~/.cache/whisper/ 2>/dev/null
```

---

## 第三步：执行转录（后台运行）

### 3.1 准备环境变量

```bash
# 设置完整 PATH（根据实际 Python 版本和 ffmpeg 路径调整）
PYTHON_VER="311"  # 根据实际版本修改
PYTHON_PATH="/c/Users/$USERNAME/AppData/Local/Programs/Python/Python${PYTHON_VER}"
export PATH="$FFMPEG_PATH:$PYTHON_PATH:$PYTHON_PATH/Scripts:$PATH"

# 音频文件路径
AUDIO_FILE="序列 01.mp3"
```

### 3.2 后台执行转录

```bash
# 在后台执行转录，同时记录日志
whisper "$AUDIO_FILE" --model small --language Chinese --output_format txt --output_dir . $DEVICE > /tmp/whisper_progress.log 2>&1 &

WHISPER_PID=$!
echo "转录进程已启动，PID: $WHISPER_PID"
```

### 3.3 显示进度提示

```bash
# 简单进度提示循环
while kill -0 $WHISPER_PID 2>/dev/null; do
    if ls *.txt 1>/dev/null 2>&1; then
        echo "转录完成！"
        break
    fi
    echo "转录中..."
    sleep 10
done

# 检查输出文件
ls -lh *.txt
```

---

## 第四步：快速执行模板

### 完整一键执行脚本

```bash
#!/bin/bash

# ===== 配置区 =====
AUDIO_FILE="序列 01.mp3"
MODEL="small"
LANGUAGE="Chinese"
# ==================

echo "=== 音频转录任务开始 ==="

# 1. 检查/配置 pip 镜像源
CURRENT_MIRROR=$(pip config get global.index-url 2>/dev/null || echo "")
if [[ "$CURRENT_MIRROR" != *"aliyun"* ]] && [[ "$CURRENT_MIRROR" != *"tsinghua"* ]] && [[ "$CURRENT_MIRROR" != *"tuna"* ]] && [[ "$CURRENT_MIRROR" != *"ustc"* ]]; then
    pip config set global.index-url https://mirrors.aliyun.com/pypi/simple/
    pip config set global.trusted-host mirrors.aliyun.com
    echo "已切换 pip 源为阿里云镜像"
fi

# 2. 检查/设置 ffmpeg
if [ -z "$FFMPEG_PATH" ]; then
    FFMPEG_BIN=$(find /tmp -name "ffmpeg.exe" -path "*essentials*" 2>/dev/null | head -1)
    if [ -z "$FFMPEG_BIN" ]; then
        echo "下载 ffmpeg..."
        curl -L "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip" -o /tmp/ffmpeg.zip
        unzip -q /tmp/ffmpeg.zip -d /tmp/
        FFMPEG_BIN=$(find /tmp -name "ffmpeg.exe" -path "*essentials*" 2>/dev/null | head -1)
    fi
    FFMPEG_PATH=$(dirname "$FFMPEG_BIN")
fi

# 3. 设置 Python 路径（根据实际 Python 版本调整）
PYTHON_VER="311"  # 如果是 Python 3.12 改为 312，Python 3.14 改为 314
PYTHON_PATH="/c/Users/$USERNAME/AppData/Local/Programs/Python/Python${PYTHON_VER}"
export PATH="$FFMPEG_PATH:$PYTHON_PATH:$PYTHON_PATH/Scripts:$PATH"

# 4. 检查/安装 PyTorch（先 CPU，验证后有需要再 GPU）
# 默认安装 CPU 版本（快速，115MB）
if ! python -c "import torch" 2>/dev/null; then
    echo "安装 PyTorch CPU 版本..."
    python -m pip install torch torchvision torchaudio
fi

# 可选：如果有 NVIDIA GPU 且需要加速，下载安装 CUDA 版本
# if nvidia-smi &>/dev/null; then
#     echo "检测到 NVIDIA GPU，可升级 CUDA 版本加速..."
#     # 获取 CUDA 版本并选择对应 wheel
#     CUDA_VER=$(nvidia-smi | grep "CUDA Version" | grep -oP '\d+\.\d+')
#     echo "检测到 CUDA $CUDA_VER"
#     # Python 版本标签：cp311/cp312/cp314
#     PYTHON_TAG="cp314"  # 根据实际版本修改
#     # 下载并本地安装（以 cu130 为例，根据实际版本调整）
#     curl -L -C - -o /tmp/torch_cu130.whl \
#       "https://mirrors.aliyun.com/pytorch-wheels/cu130/torch-2.11.0%2Bcu130-${PYTHON_TAG}-${PYTHON_TAG}-win_amd64.whl"
#     python -m pip uninstall torch torchvision torchaudio -y
#     python -m pip install /tmp/torch_cu130.whl
#     python -m pip install torchvision torchaudio -f https://mirrors.aliyun.com/pytorch-wheels/cu130
# fi

# 5. 检查 GPU 并设置设备
if python -c "import torch; exit(0 if torch.cuda.is_available() else 1)" 2>/dev/null; then
    DEVICE="--device cuda"
    echo "使用 GPU 加速"
else
    DEVICE="--device cpu"
    echo "使用 CPU 转录"
fi

# 6. 检查/安装 Whisper
if ! whisper --version &>/dev/null; then
    echo "安装 Whisper..."
    python -m pip install -U openai-whisper
fi

# 7. 执行转录
echo "开始转录: $AUDIO_FILE"
whisper "$AUDIO_FILE" --model $MODEL --language $LANGUAGE --output_format txt $DEVICE &
WHISPER_PID=$!

# 8. 等待完成
while kill -0 $WHISPER_PID 2>/dev/null; do
    sleep 5
    if ls *.txt 1>/dev/null 2>&1; then
        echo "转录完成！"
        break
    fi
    echo "转录中..."
done

# 9. 显示结果
ls -lh *.txt
echo "=== 任务完成 ==="
```

---

## 常用参数速查

| 参数 | 说明 | 推荐值 |
|------|------|--------|
| `--model` | 模型大小 | small（平衡速度质量）|
| `--language` | 语言 | Chinese / English |
| `--output_format` | 输出格式 | txt / srt / vtt / json |
| `--device` | 计算设备 | cuda（优先）/ cpu |
| `--output_dir` | 输出目录 | .（当前目录）|

---

## 模型选择参考

| 模型 | 大小 | GPU速度 | CPU速度 | 准确度 | 适用场景 |
|------|------|---------|---------|--------|----------|
| tiny | 39 MB | 极快 | 快 | 一般 | 快速测试 |
| base | 74 MB | 很快 | 中等 | 较好 | 日常使用 |
| small | 244 MB | 快 | 慢 | 好 | **推荐** |
| medium | 769 MB | 中等 | 很慢 | 很好 | 高质量 |
| large | 1550 MB | 慢 | 极慢 | 最好 | 专业用途 |

> **注意**：只有 GPU 版本才推荐 medium/large，CPU 转录强烈建议用 small 及以下模型，否则 60 分钟音频 medium 可能需要 2-4 小时

---

## 故障排查

### 问题：pip 下载超时/失败
**原因**：使用的是 PyPI 官方源，国内网络无法稳定访问
**解决**：
1. 执行 `pip config set global.index-url https://mirrors.aliyun.com/pypi/simple/`
2. 执行 `pip config set global.trusted-host mirrors.aliyun.com`
3. 确认：`pip config get global.index-url` 应返回阿里云地址

### 问题：PyTorch CUDA whl 下载太慢或中断
**原因**：pip 从阿里云 PyTorch 镜像下载大文件速度慢，且容易超时
**解决**：
1. 使用 aria2c 多线程下载：`aria2c -x 16 -s 16 -c -d /tmp -o torch_cu130.whl "https://mirrors.aliyun.com/pytorch-wheels/cu130/torch-2.11.0%2Bcu130-cp314-cp314-win_amd64.whl"`
2. 或使用 curl 断点续传：`curl -L -C - -o /tmp/torch_cu130.whl "https://..."`
3. 下载完成后本地安装：`pip install /tmp/torch_cu130.whl`
4. 可尝试南京大学镜像：`mirrors.nju.edu.cn/pytorch/whl`

### 问题：ffmpeg 找不到
**解决**：检查 FFMPEG_PATH 是否正确设置，ffmpeg.exe 是否在 PATH 中

### 问题：CUDA 不可用
**这是正常的**——如果使用的是 PyTorch CPU 版本，`torch.cuda.is_available()` 返回 False 是预期行为，无需解决。

**如需 CUDA 加速**（仅当确认 CPU 版本转录流程正常后执行）：
1. 检查 NVIDIA 驱动是否安装
2. 确认安装的是 CUDA 版本而非 CPU 版本（`python -c "import torch; print(torch.__version__)"` 应含 `+cu` 而非 `+cpu`）
3. 确认 PyTorch 的 CUDA 版本 <= nvidia-smi 显示的 CUDA 版本
4. 按 2.2 阶段二升级 GPU 版本

### 问题：用户名含中文或特殊字符导致路径错误
**表现**：`/c/Users/用户名/` 路径中有空格或中文，`xargs dirname` 等命令执行失败
**解决**：手动指定 Python 和 ffmpeg 的绝对路径，避免依赖动态检测

### 问题：Whisper 模型下载慢
**原因**：首次执行自动下载模型文件（small 约 244MB）
**解决**：提前手动下载模型：
```bash
python -c "import whisper; model = whisper.load_model('small')"
```

### 问题：转录卡住
**解决**：
1. 检查音频文件是否损坏
2. 尝试更小的模型
3. 检查磁盘空间

---

## 本次工作记录

> 执行转录后由 AI 自动填写以下内容

| 项目 | 内容 |
|------|------|
| 音频文件 | （待填写） |
| 文件大小 | （待填写） |
| 音频时长 | （待填写） |
| 使用模型 | （待填写） |
| 计算设备 | （待填写，如：RTX 3060 Laptop GPU / CPU） |
| 转录耗时 | （待填写） |
| 输出文件 | （待填写） |

### 关键命令记录

```bash
# 环境准备（根据实际路径填写）
export PATH="<ffmpeg路径>:<Python路径>:<Python Scripts路径>:$PATH"

# GPU 验证
python -c "import torch; print(f'CUDA可用: {torch.cuda.is_available()}')"

# 执行转录（根据 GPU 情况选择 --device cuda 或 --device cpu）
whisper "<音频文件名>" --model <模型> --language <语言> --output_format txt --device <cuda|cpu>
```

---

*本文档针对 AI 执行优化，简化人类阅读内容，专注执行效率*
*更新时间：2026年4月20日*
