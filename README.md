<!--
  Vecna — a terminal-first SSH workspace.
  Keep this README aligned with the actual CLI, configuration, and release workflow.
-->

<div align="center">
  <img src="assets/vecna.jpg" alt="Vecna" width="260" />

  <h1>Vecna</h1>
  <p><strong>A minimal, keyboard-driven SSH workspace for managing servers from your terminal.</strong></p>

  <p>
    Connect to hosts, work across SSH sessions, transfer files, and run routine operations
    without juggling a collection of terminal windows and shell aliases.
  </p>
</div>

<p align="center">
  <img src="assets/screenshot-1.png" alt="Vecna host dashboard" width="49%" />
  <img src="assets/screenshot-2.png" alt="Vecna terminal workspace" width="49%" />
</p>

---

## Why Vecna?

Server work often means switching between SSH sessions, remembering host aliases, copying files, setting up tunnels, and repeating the same operational commands.

Vecna brings those workflows into one terminal user interface (TUI). It is designed to keep common server tasks discoverable and keyboard-accessible while still using standard SSH infrastructure.

## Features

| Area | What you can do |
| --- | --- |
| **Host management** | Add, edit, remove, search, tag, pin, and annotate SSH hosts. |
| **SSH connections** | Open interactive SSH sessions in tabs, reconnect, and switch between sessions. |
| **Authentication** | Use SSH private keys, an SSH agent, or a password. Configure a specific identity file per host. |
| **SSH configuration** | Import host aliases from your `~/.ssh/config`, including common connection settings such as user, port, identity file, and ProxyJump. |
| **Jump hosts** | Connect through a configured bastion using `ProxyJump`. |
| **File operations** | Browse remote files and transfer files or directories. Recursive transfers report progress and support concurrent workers. |
| **Port forwarding** | Create local, remote, and dynamic SOCKS5 forwarding through an SSH connection. |
| **Command execution** | Run saved commands on a host or execute a command across selected hosts. |
| **Server visibility** | View live CPU, memory, disk, network throughput, and active SSH session information when available. |
| **Workflow conveniences** | Use command history, a command palette, searchable host lists, and keyboard-driven navigation. |
| **Host-key checks** | Review unknown or changed SSH host-key fingerprints before trusting or replacing a key. |

Vecna connects to your existing SSH servers. It does not require a Vecna agent to be installed on those servers.

## Installation

### Linux and macOS

The installer detects the operating system and CPU architecture, downloads the matching release, verifies its SHA-256 checksum, and installs the binary to `~/.local/bin` by default.

```bash
curl -fsSL https://raw.githubusercontent.com/zhravan/vecna/main/scripts/install.sh | sh
```

### Windows

Run the following in PowerShell. The installer downloads the matching Windows build, verifies its SHA-256 checksum, installs it under your local application data directory, and adds that directory to your user PATH.

```powershell
irm https://raw.githubusercontent.com/zhravan/vecna/main/scripts/install.ps1 | iex
```

The Windows SFTP and SCP integrations use the OpenSSH client. On Windows 10/11, the OpenSSH Client optional feature is generally available; install or enable it if those operations are unavailable.

### Install with Go

Requires Go 1.24.3 or newer.

```bash
go install github.com/zhravan/vecna@latest
```

Make sure `$(go env GOPATH)/bin` is on your PATH.

### Build from source

```bash
git clone https://github.com/zhravan/vecna.git
cd vecna

make build
./bin/vecna
```

To install the binary into your Go bin directory instead:

```bash
make install
```

### Supported release targets

Prebuilt releases are published for:

| Operating system | Architectures |
| --- | --- |
| Linux | `amd64`, `arm64` |
| macOS | `amd64` (Intel), `arm64` (Apple Silicon) |
| Windows | `amd64`, `arm64` |

See [GitHub Releases](https://github.com/zhravan/vecna/releases) for downloadable archives and checksums.

## Quick start

Launch Vecna:

```bash
vecna
```

On first launch, Vecna creates its configuration directory and a default configuration file. Add a host from the interface with `a`, select it, and press `c` to connect.

Print the installed version:

```bash
vecna version
```

You can also supply a non-default configuration file:

```bash
vecna --config /path/to/config.yaml
```

## Configuration

By default, Vecna reads:

- **Linux / macOS:** `~/.config/vecna/config.yaml`
- **Windows:** `%USERPROFILE%\.config\vecna\config.yaml`

The file is created on first launch with an empty host list and a set of example operational commands.

### Example

```yaml
hosts:
  - name: production
    hostname: 203.0.113.10
    user: deploy
    port: 22
    identity_file: ~/.ssh/id_ed25519
    tags:
      - production
      - api
    proxy_jump: bastion
    connect_timeout_seconds: 10
    keep_alive_seconds: 30
    pinned: true
    notes: Primary production API server

  - name: bastion
    hostname: bastion.example.com
    user: ubuntu
    port: 22
    identity_file: ~/.ssh/id_ed25519
    tags:
      - infrastructure

commands:
  - label: disk usage
    command: df -h

  - label: memory
    command: free -m

  - label: docker containers
    command: docker ps
```

### Host fields

| Field | Description |
| --- | --- |
| `name` | Friendly name used to identify the host inside Vecna. |
| `hostname` | Server hostname or IP address. |
| `user` | SSH username. |
| `port` | SSH port; typically `22`. |
| `identity_file` | Optional path to the SSH private key. Supports `~` paths. |
| `tags` | Optional labels for organizing and filtering hosts. |
| `proxy_jump` | Optional name of another configured Vecna host to use as a jump host. |
| `connect_timeout_seconds` | Optional connection timeout. |
| `keep_alive_seconds` | Optional keep-alive interval. |
| `pinned` | Pin a frequently used host in the list. |
| `notes` | Optional notes displayed in the host details panel. |
| `use_agent` | Set to `true` to explicitly try the SSH agent for this host. |

The `commands` list defines shortcuts available in the command runner. Each entry has a display `label` and the shell `command` to execute.

### Import existing SSH hosts

Press `i` in the host list to import aliases from your OpenSSH configuration, usually `~/.ssh/config`. Review the imported entries and their connection settings before connecting.

## Keyboard shortcuts

Vecna is designed to be used without a mouse. The in-app help (`?`) shows context-sensitive shortcuts.

| Key | Action |
| --- | --- |
| `↑` / `k`, `↓` / `j` | Move through the host list |
| `/` | Search or filter hosts |
| `a` | Add a host |
| `e` | Edit the selected host |
| `d` | Delete the selected host |
| `i` | Import from OpenSSH config |
| `P` | Pin or unpin a host |
| `c` | Connect to the selected host |
| `r` | Run a command |
| `f` | Open the SFTP browser |
| `p` | Manage port forwarding |
| `t` | Open file transfer |
| `Space` | Select hosts for multi-host actions |
| `Tab` / `Ctrl+Tab` | Move between fields or tabs, depending on the view |
| `1`–`9` | Jump to a tab from the host list |
| `?` | Show help |
| `q` | Quit or leave the current view |

While connected, use the tab controls shown in the status bar to move between SSH sessions. Key combinations can vary by terminal emulator.

## Development

### Requirements

- Go 1.24.3 or newer
- Git
- A terminal emulator with interactive TUI support

### Useful commands

```bash
make build     # Build ./bin/vecna
make run       # Run from source
make test      # Run all Go tests
make lint      # Run golangci-lint (install separately)
make clean     # Remove build output
make version   # Print version.txt
```

You can also run the standard Go checks directly:

```bash
go test ./...
go vet ./...
```

### Project structure

```text
.
├── cmd/                 # CLI commands and entry point
├── internal/
│   ├── config/          # YAML configuration and defaults
│   ├── history/         # Per-host command history
│   ├── monitoring/      # Remote system-stat parsing
│   ├── safety/          # Potentially destructive command checks
│   ├── search/          # Host and command search helpers
│   ├── sftp/            # File browsing and transfer helpers
│   ├── ssh/             # SSH sessions, auth, forwarding, and retries
│   ├── sshconfig/       # OpenSSH config import
│   ├── state/           # TUI state persistence
│   ├── terminal/        # Terminal output helpers
│   ├── tui/             # Bubble Tea user interface
│   └── update/          # Update support
├── scripts/             # Linux/macOS and Windows installers
├── assets/              # Project artwork and screenshots
├── .github/workflows/   # Tests, formatting, and release automation
├── Makefile
├── go.mod
└── version.txt
```

## Releases and versioning

The release workflow builds and tests the project for the supported operating systems and architectures, runs `go vet`, packages the binaries, and publishes a GitHub Release with SHA-256 checksums.

The release version is read from `version.txt` and must use semantic versioning (`MAJOR.MINOR.PATCH`). Update the version before pushing a release-worthy change to `main`.

## Security

- Verify SSH host-key fingerprints independently before trusting a new host or replacing a changed key.
- Prefer SSH keys or your SSH agent over storing reusable passwords in configuration.
- Keep your Vecna configuration, SSH private keys, and other credentials out of source control.
- Review commands before executing them, especially when running them across multiple hosts.

Please report security issues privately as described in [SECURITY.md](SECURITY.md), rather than opening a public issue.

## Contributing

Contributions, bug reports, and feature ideas are welcome.

1. Read [CONTRIBUTING.md](CONTRIBUTING.md).
2. Open an issue to discuss substantial changes.
3. Create a focused branch and submit a pull request.
4. Include tests for behavior changes where practical.

## License

Vecna is released under the [MIT License](LICENSE).

---

<div align="center">
  <sub>Built for people who spend a lot of time on servers.</sub>
</div>
