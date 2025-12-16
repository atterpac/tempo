# Theme Switcher Implementation Plan

## Overview

Implement a runtime theme switching system with YAML configuration and 11 built-in themes.

## Built-in Themes

### TokyoNight Variants
| Color | Night | Storm | Moon | Day |
|-------|-------|-------|------|-----|
| bg | `#1a1b26` | `#24283b` | `#222436` | `#e1e2e7` |
| bg_dark | `#16161e` | `#1f2335` | `#1e2030` | `#d0d5e3` |
| fg | `#c0caf5` | `#c0caf5` | `#c8d3f5` | `#3760bf` |
| fg_dim | `#565f89` | `#565f89` | `#636da6` | `#848cb5` |
| border | `#15161e` | `#1d202f` | `#1b1d2b` | `#b4b5b9` |
| highlight | `#283457` | `#292e42` | `#2f334d` | `#b7c1e3` |
| accent | `#7aa2f7` | `#7aa2f7` | `#82aaff` | `#2e7de9` |
| accent_dim | `#bb9af7` | `#bb9af7` | `#c099ff` | `#9854f1` |
| green | `#9ece6a` | `#9ece6a` | `#c3e88d` | `#587539` |
| yellow | `#e0af68` | `#e0af68` | `#ffc777` | `#8c6c3e` |
| red | `#f7768e` | `#f7768e` | `#ff757f` | `#f52a65` |
| orange | `#ff9e64` | `#ff9e64` | `#ff966c` | `#b15c00` |
| purple | `#9d7cd8` | `#9d7cd8` | `#c099ff` | `#7847bd` |
| cyan | `#7dcfff` | `#7dcfff` | `#86e1fc` | `#007197` |

### Catppuccin Variants
| Color | Mocha | Macchiato | Frappe | Latte |
|-------|-------|-----------|--------|-------|
| base | `#1e1e2e` | `#24273a` | `#303446` | `#eff1f5` |
| mantle | `#181825` | `#1e2030` | `#292c3c` | `#e6e9ef` |
| crust | `#11111b` | `#181926` | `#232634` | `#dce0e8` |
| text | `#cdd6f4` | `#cad3f5` | `#c6d0f5` | `#4c4f69` |
| subtext0 | `#a6adc8` | `#a5adcb` | `#a5adce` | `#6c6f85` |
| overlay0 | `#6c7086` | `#6e738d` | `#737994` | `#9ca0b0` |
| surface0 | `#313244` | `#363a4f` | `#414559` | `#ccd0da` |
| surface1 | `#45475a` | `#494d64` | `#51576d` | `#bcc0cc` |
| surface2 | `#585b70` | `#5b6078` | `#626880` | `#acb0be` |
| pink | `#f5c2e7` | `#f5bde6` | `#f4b8e4` | `#ea76cb` |
| mauve | `#cba6f7` | `#c6a0f6` | `#ca9ee6` | `#8839ef` |
| red | `#f38ba8` | `#ed8796` | `#e78284` | `#d20f39` |
| peach | `#fab387` | `#f5a97f` | `#ef9f76` | `#fe640b` |
| yellow | `#f9e2af` | `#eed49f` | `#e5c890` | `#df8e1d` |
| green | `#a6e3a1` | `#a6da95` | `#a6d189` | `#40a02b` |
| teal | `#94e2d5` | `#8bd5ca` | `#81c8be` | `#179299` |
| blue | `#89b4fa` | `#8aadf4` | `#8caaee` | `#1e66f5` |

### Dracula
| Color | Dark | Light (Alucard) |
|-------|------|-----------------|
| bg | `#282a36` | `#fffbeb` |
| bg_light | `#44475a` | `#cfcfde` |
| fg | `#f8f8f2` | `#1f1f1f` |
| fg_dim | `#6272a4` | `#6c664b` |
| purple | `#bd93f9` | `#644ac9` |
| pink | `#ff79c6` | `#a3144d` |
| green | `#50fa7b` | `#14710a` |
| yellow | `#f1fa8c` | `#846e15` |
| red | `#ff5555` | `#cb3a2a` |
| orange | `#ffb86c` | `#a34d14` |
| cyan | `#8be9fd` | `#036a96` |

### Nord
| Color | Hex | Role |
|-------|-----|------|
| nord0 | `#2e3440` | bg |
| nord1 | `#3b4252` | bg_light |
| nord2 | `#434c5e` | highlight |
| nord3 | `#4c566a` | border |
| nord4 | `#d8dee9` | fg |
| nord5 | `#e5e9f0` | fg_bright |
| nord8 | `#88c0d0` | accent |
| nord9 | `#81a1c1` | accent_dim |
| nord11 | `#bf616a` | red |
| nord12 | `#d08770` | orange |
| nord13 | `#ebcb8b` | yellow |
| nord14 | `#a3be8c` | green |
| nord15 | `#b48ead` | purple |

## Config Format (YAML)

```yaml
# ~/.config/loom/config.yaml
theme: "catppuccin-mocha"

connection:
  address: "localhost:7233"
  namespace: "default"
  tls:
    cert: ""
    key: ""
    ca: ""
    server_name: ""
    skip_verify: false
```

## Custom Theme Format

```yaml
# ~/.config/loom/themes/custom.yaml
name: "My Theme"
type: "dark"

colors:
  bg: "#1e1e2e"
  bg_light: "#313244"
  bg_dark: "#181825"
  fg: "#cdd6f4"
  fg_dim: "#6c7086"
  border: "#45475a"
  highlight: "#585b70"
  accent: "#f5c2e7"
  accent_dim: "#cba6f7"
  running: "#f9e2af"
  completed: "#a6e3a1"
  failed: "#f38ba8"
  canceled: "#fab387"
  terminated: "#cba6f7"
  timed_out: "#f38ba8"
  header: "#181825"
  menu: "#1e1e2e"
  table_header: "#f5c2e7"
  key: "#cba6f7"
  crumb: "#f5c2e7"
  panel_border: "#585b70"
  panel_title: "#f5c2e7"
```

## Implementation Phases

### Phase 1: Config Infrastructure
- [ ] Create `internal/config/` package
- [ ] Add `gopkg.in/yaml.v3` dependency
- [ ] Implement config struct and load/save
- [ ] XDG path resolution

### Phase 2: Theme System
- [ ] Define Theme struct with all color fields
- [ ] Implement hex string to tcell.Color parsing
- [ ] Create all 11 built-in themes
- [ ] Theme validation logic

### Phase 3: Refactor styles.go
- [ ] Add activeTheme variable and mutex
- [ ] Create getter functions (Bg(), Fg(), etc.)
- [ ] Add SetTheme() function
- [ ] Add OnThemeChange() subscriber registration
- [ ] Update StatusColorTag() to use active theme

### Phase 4: Update UI Components
- [ ] internal/ui/table.go - use getters, add listener
- [ ] internal/ui/panel.go - use getters, add listener
- [ ] internal/ui/statsbar.go - use getters, add listener
- [ ] internal/ui/menu.go - use getters, add listener
- [ ] internal/ui/crumbs.go - use getters, add listener
- [ ] internal/ui/app.go - use config for init

### Phase 5: Update Views
- [ ] internal/view/workflow_list.go
- [ ] internal/view/workflow_detail.go
- [ ] internal/view/namespace_list.go
- [ ] internal/view/task_queue.go
- [ ] internal/view/event_history.go
- [ ] internal/view/app.go

### Phase 6: Runtime Switching
- [ ] Add theme command/keybinding
- [ ] Persist theme choice to config
- [ ] Theme picker UI (optional)

### Phase 7: Cleanup
- [ ] Remove duplicate style init from cmd/main.go
- [ ] Update CLI flags to use config
- [ ] Documentation

## Architecture

```
internal/
├── config/
│   ├── config.go       # Config struct, Load(), Save()
│   ├── themes.go       # Theme struct, built-in themes
│   └── xdg.go          # XDG path helpers
└── ui/
    └── styles.go       # Refactored: getters + listeners
```

## Runtime Switching Pattern

```go
// Subscribe to theme changes
ui.OnThemeChange(func(t *config.Theme) {
    component.applyTheme(t)
})

// Switch theme (notifies all subscribers)
ui.SetTheme("dracula")
```

## Dependencies

```
gopkg.in/yaml.v3 v3.0.1
```
