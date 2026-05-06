# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build -o readx ./cmd/readx    # build binary
go test ./...                     # run all tests
go test ./internal/service/ -v    # run pagination tests with verbose output
go vet ./...                      # static analysis
```

## Architecture

readx is a terminal EPUB/TXT reader using the [Bubble Tea](https://github.com/charmbracelet/bubbletea) Elm-like TUI framework. It has two views managed by a state machine.

### State machine (RootModel)

`RootModel` in `internal/ui/root.go` owns two child models and routes messages:

```
LibraryState ──(OpenBookMsg)──▶ ReaderState
     ◀──(SwitchToLibraryMsg)──
```

- `RootModel.Update` intercepts `WindowSizeMsg` and forwards it to the active child — without this, the reader never gets terminal dimensions and pagination fails.
- `RootModel.Cleanup()` delegates to `ReaderModel.Cleanup()` which persists progress. Called via `defer` in `main.go` to catch ctrl+c.
- On `SwitchToLibraryMsg`, the reader is cleaned up and set to `nil`, then `LibraryModel` is rebuilt from config.

### Domain-driven layers

```
cmd/readx/main.go          # entry point, flag parsing, RootModel wiring
internal/
  domain/                  # Reader interface, Book, Chapter, Page, LibraryEntry
  adapters/                # TxtAdapter, EpubAdapter (implements domain.Reader)
  persistence/             # JSON config read/write at ~/.config/readx/config.json
  service/                 # Pagination engine (stateless)
  ui/                      # All Bubble Tea models and views
```

- `domain.Reader` is the central interface — adapters implement it, UI consumes it.
- Adapters use `golang.org/x/text` for multi-encoding TXT (GBK, Big5, Shift-JIS) and `chardet` for auto-detection.

### Pagination pipeline

`ContentArea(termWidth, termHeight)` → `wrapText(rawContent, contentWidth)` → `Paginate()` → `PageCache`

Key behaviors to preserve:
- **`wrapText` prepends `　　`** (two full-width CJK spaces) to each non-empty paragraph for first-line indent. Empty paragraphs are preserved.
- **`ContentArea` subtracts 4** from `termWidth` to account for `ContentStyle.Padding(0,2)` — otherwise wrapped text overflows the padded rendering box.
- **Pagination is async**: `paginateChapterCmd` returns a `tea.Cmd` that runs `PaginateOrCache` in a goroutine and posts `paginateDoneMsg` back with the result.
- **Cache key is chapter index only**: `PageCache` is a `map[int][]domain.Page`. Only `EvictExcept` is used (the singular `Evict` was removed).

### Key handling tiers (ReaderModel)

Input priority in `handleKey()`:
1. Popup (`showPopup`) — Y/N only
2. Command mode (`commandMode`) — enter/esc + textinput passthrough
3. Chapter modal (`showChapters`) — j/k navigation, enter to jump
4. Normal reading — q, j/k page, h/l chapter, tab (chapter list), / (command mode)

### CJK width handling

`go-runewidth.RuneWidth()` is the single source of truth for character display width. CJK characters return 2, ASCII returns 1. Used in:
- `wrapSingleLine` for line-breaking
- `truncateToWidth` in `util.go` for sidebar/modal string truncation with `…` suffix

### Footer layout

`FooterView` renders a three-column bar:
- **Normal**: `[阅读]  [████░░] 80%    第3/10页  tab目录 q退出`
- **Command**: `:goto ▐                          第3/10页  enter执行 esc取消`

Progress is `calcProgress()` — weighted average: `chapterN * (100/totalChapters) + (page/totalPages) * (100/totalChapters)`.

### Styles

Defined in `internal/ui/styles.go` as package-level `var` (not `const`, because lipgloss styles are mutable). Colors use a warm cream/parchment palette: `#F5E6D3` primary, `#C8A96E` gold accent, `#2C2416` dark background.
