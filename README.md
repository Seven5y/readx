<p align="center">
  <img src="assets/readx_logo.svg" alt="readx" width="120" />
</p>

<h1 align="center">readx</h1>

<p align="center">
  <em>A terminal-first reading environment. Built for immersion, speed, and the keyboard.</em>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Built_with-AI_Agent_%26_Go-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Built with AI Agent & Go" />
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#installation">Installation</a> •
  <a href="#usage">Usage</a> •
  <a href="#keyboard-shortcuts">Shortcuts</a> •
  <a href="#built-with">Built With</a> •
  <a href="README_ZH.md">中文</a>
</p>

---

## Why readx?

Most ebook readers are GUI-first, heavy, and mouse-driven. **readx** brings reading back to the terminal — fast startup, zero distractions, fully keyboard-operable. It handles CJK typography correctly (indent, character-width, line-wrapping) and keeps a smart library shelf so you can jump between books without losing your place.

---

## Features

- **Multi-format support** — EPUB (chapter-aware parsing) and TXT (auto encoding detection for GBK, Big5, Shift-JIS, UTF-8).
- **Typography engine** — First-line indent with full-width spaces, CJK-aware line wrapping via `go-runewidth`, and proper paragraph boundaries even in EPUBs.
- **Vim-style command mode** — Press `/` to open a command prompt: `/list` returns to the shelf, `/goto 5` jumps to page 5, `/q` saves and quits.
- **Smart library shelf** — Tracks reading progress (%), last-read time, and book metadata. Open a book and it auto-adds to your shelf.
- **Chapter navigation** — Modal chapter list with fuzzy scrolling, press `Tab` to open, `j`/`k` to move, `Enter` to jump.
- **Low footprint** — Written in Go. Single binary, no runtime, low memory. Pre-paginates chapters so even large files scroll instantly.

---

## Installation

```bash
go install github.com/Seven5y/readx/cmd/readx@latest
```

Or clone and build:

```bash
git clone https://github.com/Seven5y/readx.git
cd readx
go build -o readx ./cmd/readx
```

Requires Go 1.21+.

---

## Usage

```bash
# Open a book directly (auto-adds to your library)
readx path/to/book.epub
readx novel.txt

# Open the library shelf
readx
readx --list
```

---

## Keyboard Shortcuts

### Reading Mode

| Key | Action |
|-----|--------|
| `↑` / `↓` , `j` / `k` | Page up / down |
| `←` / `→` , `h` / `l` | Previous / next chapter |
| `Tab` | Open chapter list modal |
| `/` | Enter command mode |
| `q` | Save progress and quit |

### Chapter Modal

| Key | Action |
|-----|--------|
| `↑` / `↓` , `j` / `k` | Move cursor |
| `Enter` | Jump to selected chapter |
| `Esc` / `Tab` | Close modal |

### Command Mode

| Command | Action |
|---------|--------|
| `/list` | Save progress, return to shelf |
| `/goto <N>` | Jump to page N |
| `/q` | Save progress and quit |
| `Esc` | Exit command mode |

### Library Shelf

| Key | Action |
|-----|--------|
| `Enter` | Open selected book |
| `d` | Remove book from shelf |
| `q` | Quit |

---

## Built With

| Library | Role |
|---------|------|
| [Bubble Tea](https://github.com/charmbracelet/bubbletea) | TUI framework (Elm Architecture) |
| [Lip Gloss](https://github.com/charmbracelet/lipgloss) | Terminal styling & layout |
| [Bubbles](https://github.com/charmbracelet/bubbles) | `list` & `textinput` components |
| [go-runewidth](https://github.com/mattn/go-runewidth) | CJK character-width handling |
| [golang.org/x/text](https://pkg.go.dev/golang.org/x/text) | Multi-encoding conversion |
| [chardet](https://github.com/saintfish/chardet) | Auto encoding detection for TXT |

---

## Inspiration

readx was born from a simple question: *what does a first-class reading experience look like for the terminal?* EPUB readers are everywhere on desktop and mobile, but the terminal — where developers spend most of their time — is largely ignored. The entire project was built with [Claude Code](https://claude.ai/code), an AI-powered coding agent, as an experiment in human-AI collaborative software design. Every feature, from the modal chapter picker to the CJK typography engine, was iteratively refined through conversation.

If you find readx useful, **star the repo** and share it with someone who lives in the terminal.

---

<p align="center">
  Made with ☕ and Claude Code
</p>
