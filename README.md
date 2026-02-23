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
    proxy_jump: bastion   # optional: name of another host to use as jump/bastion

commands:                  # optional: saved commands for "Run command" (r)
  - label: "disk usage"
    command: "df -h"
  - label: "memory"
    command: "free -m"
```

## Keybindings

| Key | Action |
|-----|--------|
| `q` | Quit |
| `1`–`9` | Switch to tab (1=Hosts, 2–9=SSH tabs) |
| `ctrl+←` / `ctrl+→` | Previous / next tab |
| `/` | Filter hosts by name or tag |
| `↑/k` | Up |
| `↓/j` | Down |
| `⏎` | Select / Connect (opens new SSH tab) |
| `esc` | Back / close SSH tab |
| `?` | Help |
| `a` | Add host |
| `d` | Delete host |
| `e` | Edit host |
| `c` | Connect (SSH in new TUI tab) |
| `f` | Open SFTP in new terminal |
| `p` | Port forward |
| `r` | Run saved command |
| `t` | File transfer (scp push/pull) |

## License

MIT
