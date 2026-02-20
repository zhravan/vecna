# Vecna

Simplify ssh-ing with minimalistic TUI.

## Install

```bash
go install github.com/shravan20/vecna@latest
```

Or build from source:

```bash
make build
./bin/vecna
```

## Usage

```bash
vecna          # Launch TUI
vecna version  # Print version
```

## Config

Config lives at `~/.config/vecna/config.yaml`

```yaml
hosts:
  - name: prod
    hostname: 192.168.1.100
    user: admin
    port: 22
    identity_file: ~/.ssh/id_rsa
    tags: [production]
```

## Keybindings

| Key | Action |
|-----|--------|
| `q` | Quit |
| `↑/k` | Up |
| `↓/j` | Down |
| `⏎` | Select |
| `esc` | Back |
| `?` | Help |

## License

MIT
