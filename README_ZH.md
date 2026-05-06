<p align="center">
  <img src="https://raw.githubusercontent.com/Seven5y/readx/main/.github/logo.svg" alt="readx" width="120" />
</p>

<h1 align="center">readx</h1>

<p align="center">
  <em>回归终端。沉浸阅读，键盘至上。</em>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Built_with-AI_Agent_%26_Go-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Built with AI Agent & Go" />
</p>

<p align="center">
  <a href="#特性">特性</a> •
  <a href="#安装">安装</a> •
  <a href="#快速上手">快速上手</a> •
  <a href="#快捷键">快捷键</a> •
  <a href="#技术栈">技术栈</a> •
  <a href="README.md">English</a>
</p>

---

## 为什么选择 readx？

市面上的阅读器大多是 GUI 图形界面，启动缓慢、功能臃肿、依赖鼠标操作。**readx** 让阅读回归终端 —— 秒启动、零干扰、全键盘操控。针对中文排版做了深度适配（首行缩进、字符宽度计算、精准换行），并内置智能书架，让你在不同书籍之间无缝切换，阅读进度永不丢失。

---

## 特性

- **多格式支持** — 完美解析 EPUB（章节识别、块级标签保留）和 TXT（自动检测 GBK / Big5 / Shift-JIS / UTF-8 编码）。
- **中文排版引擎** — 首行缩进两个全角空格、基于 `go-runewidth` 的 CJK 字符宽度换行，EPUB 也能正确保留段落边界。
- **类 Vim 指令模式** — 按 `/` 呼出指令输入框：`/list` 返回书架、`/goto 5` 跳转到第 5 页、`/q` 保存并退出。
- **智能书架** — 记录阅读进度百分比、最后阅读时间、书籍元数据。打开任意书籍自动加入书架，无需手动管理。
- **悬浮章节目录** — 按 `Tab` 呼出居中浮窗，`j` / `k` 移动光标，`Enter` 跳转。
- **高性能低开销** — Go 语言编写，单二进制文件，内存占用极低。章节预分页，大文件也能流畅翻页。

---

## 安装

```bash
go install github.com/Seven5y/readx/cmd/readx@latest
```

或克隆源码手动构建：

```bash
git clone https://github.com/Seven5y/readx.git
cd readx
go build -o readx ./cmd/readx
```

需要 Go 1.21 以上版本。

---

## 快速上手

```bash
# 直接打开书籍（自动加入书架）
readx path/to/book.epub
readx novel.txt

# 进入书架
readx
readx --list
```

---

## 快捷键

### 阅读模式

| 按键 | 功能 |
|------|------|
| `↑` / `↓` , `j` / `k` | 上 / 下翻页 |
| `←` / `→` , `h` / `l` | 上一章 / 下一章 |
| `Tab` | 打开章节目录浮窗 |
| `/` | 进入指令模式 |
| `q` | 保存进度并退出 |

### 章节目录浮窗

| 按键 | 功能 |
|------|------|
| `↑` / `↓` , `j` / `k` | 移动光标 |
| `Enter` | 跳转到选中章节 |
| `Esc` / `Tab` | 关闭浮窗 |

### 指令模式

| 指令 | 功能 |
|------|------|
| `/list` | 保存进度，返回书架 |
| `/goto <N>` | 跳转到第 N 页 |
| `/q` | 保存进度并退出 |
| `Esc` | 退出指令模式 |

### 书架

| 按键 | 功能 |
|------|------|
| `Enter` | 打开选中书籍 |
| `d` | 从书架移除（不删除源文件） |
| `q` | 退出 |

---

## 技术栈

| 库 | 用途 |
|----|------|
| [Bubble Tea](https://github.com/charmbracelet/bubbletea) | TUI 框架（Elm 架构） |
| [Lip Gloss](https://github.com/charmbracelet/lipgloss) | 终端样式与布局 |
| [Bubbles](https://github.com/charmbracelet/bubbles) | `list`、`textinput` 组件 |
| [go-runewidth](https://github.com/mattn/go-runewidth) | CJK 字符宽度处理 |
| [golang.org/x/text](https://pkg.go.dev/golang.org/x/text) | 多编码转换（GBK、Big5 等） |
| [chardet](https://github.com/saintfish/chardet) | TXT 编码自动检测 |

---

## 开发初衷

readx 始于一个朴素的问题：*终端里的第一流阅读体验应该是什么样的？* 桌面上、手机里，EPUB 阅读器数不胜数，但开发者待得最久的终端却几乎无人问津。整个项目由 [Claude Code](https://claude.ai/code) 辅助开发，是一次人机协作编程的实验：从悬浮章节浮窗到 CJK 排版引擎，每一个细节都在对话中迭代打磨。

如果你觉得 readx 有用，请 **star 这个仓库**，分享给同样生活在终端里的朋友。

---

<p align="center">
  Made with ☕ and Claude Code
</p>
