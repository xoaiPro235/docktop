# docktop

A fast, lightweight, and modern Terminal UI (TUI) for Docker, built with Go, Bubble Tea, and Lipgloss.

Manage and monitor your Docker containers, images, volumes, and networks with keyboard-driven ease.

![docktop preview](assets/demo.gif)

---


## Installation

### Option 1: Download Pre-built Binary
Download the latest binary for your operating system (Linux, macOS, Windows) from the [Releases](https://github.com/xoaiPro235/docktop/releases) page.

```bash
# Example for Linux (x86_64)
tar -xzf docktop_*_linux_amd64.tar.gz
sudo mv docktop /usr/local/bin/
```

### Option 2: Using `go install`
```bash
go install github.com/xoaiPro235/docktop/cmd/docktop@latest
```

### Option 3: Build from Source
```bash
git clone https://github.com/xoaiPro235/docktop.git
cd docktop
make build
./docktop
```

---

## Configuration & Themes

`docktop` reads its configuration from `~/.config/docktop/config.yaml` (or the equivalent OS config directory). If the file does not exist, `docktop` automatically creates one with default settings upon first launch.

### Example `~/.config/docktop/config.yaml`:

```yaml
# docktop configuration file
# Location: ~/.config/docktop/config.yaml

# UI Theme:
# Available presets:
#   - tokyo-night : Default vibrant Tokyo Night / Neon aesthetic
#   - terminal    : Adaptive 16 ANSI colors (automatically matches your terminal emulator theme: Kitty, Alacritty, iTerm2, WezTerm)
#   - dracula     : Classic Dracula dark theme
#   - catppuccin  : Soothing Catppuccin Mocha theme
#   - nord        : Arctic, north-bluish palette
theme: "terminal"

# Docker Host:
# Optional Docker daemon endpoint or remote socket.
# host: "ssh://user@remote-vps:22"
```

> **Tip:** Setting `theme: "terminal"` will make `docktop` automatically inherit whatever colors your terminal emulator (Kitty, Alacritty, iTerm2, WezTerm) is configured with!
>
> **Remote Docker / SSH:** You can specify a remote Docker endpoint via `-H / --host`, the `host` config option, or `DOCKER_HOST`:
> ```bash
> docktop -H ssh://user@remote-vps
> ```
---
## 🙏 Acknowledgements

`docktop` is inspired by and learns from amazing open-source projects:

- **[btop](https://github.com/aristocratos/btop)** by aristocratos – Visual inspiration for Braille graph resource monitoring and mirrored layouts.
- **[lazydocker](https://github.com/jesseduffield/lazydocker)** by Jesse Duffield – Workflow inspiration for container lifecycle management and compose tree structure.
- **[Charmbracelet (Bubble Tea & Lipgloss)](https://github.com/charmbracelet)** – The wonderful TUI framework and styling toolkit powering this project.

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).
