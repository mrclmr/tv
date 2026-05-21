# German TV CLI

Open german live TV channels in VLC from the terminal.

## Requirements

- [Go](https://go.dev/dl/)
- [VLC](https://www.videolan.org/vlc/)

## Install

```bash
go install github.com/mrclmr/tv@main
```

## Usage

```bash
tv --list
```

```bash
tv <channel>
```

## Bash completions

**Linux:**

```bash
tv completion bash > ~/.local/share/bash-completion/completions/tv
```

**macOS (Homebrew):**

```bash
tv completion bash > $(brew --prefix)/etc/bash_completion.d/tv
```

Or add to `~/.bashrc` on any system:

```bash
source <(tv completion bash)
```
