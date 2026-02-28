<samp>

<p align="center">
  <img src="assets/vecna.jpg" alt="Vecna" width="400" />
</p>

<h2 align="center">Minimalistic TUI for SSH mgmt</h2>

<br>
<br>
<br>

| | |
|:---:|:---:|
| <img src="assets/screenshot-1.png" alt="Screenshot 1" width="600"/> | <img src="assets/screenshot-2.png" alt="Screenshot 2" width="600"/> |

## Install

**Linux / macOS:**

```bash
curl -sSL https://raw.githubusercontent.com/shravan20/vecna/master/scripts/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/shravan20/vecna/master/scripts/install.ps1 | iex
```

- **Go:** `go install github.com/shravan20/vecna@latest`
- **From source:**

```bash
make build
./bin/vecna   # or bin\vecna.exe on Windows
```

  `make install` installs the binary; config is created on first run with default run commands.

**Windows:** Config lives at `%USERPROFILE%\.config\vecna\config.yaml`. File transfer and “Open SFTP in new terminal” work when the [OpenSSH client](https://learn.microsoft.com/en-us/windows-server/administration/openssh/openssh_install_firstuse) is installed (default on Windows 10/11).

## Usage

```bash
vecna          # Launch TUI
vecna version  # Print version
```

## Config

**Path:** `~/.config/vecna/config.yaml`

Created on first run with default run commands. Edit to add hosts and more commands. Example:

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

---

## Contributing

Issues and PRs welcome. Tag releases with semver (`v1.0.0`); CI builds and publishes.

<details>
<summary><strong>About</strong></summary>

I spend a lot of time on servers and bare metals for deployments, containerizing dev environments, self-hosted setups. That means a lot of SSH sessions. Jumping between servers. Moving files across machines. Running repetitive ops commands.

Even though I do maintain aliases and scripts, but it still takes time and cognitive energy. I have to remember flags, paths, hostnames, sequences. Small friction that compounds.

I revisited my SSH workflows and started thinking about how much friction I quietly tolerate every day. I have to memorize commands, jumping between terminals, relying on aliases, constant context switching.

So instead of memorizing commands, why not solve it to that reduce cognitive load?

That led to Vecna (yes, Stranger Things reference, lol yep xD).

_**Note: So the idea is centralize control, reduce friction, make SSH-driven workflows better and more structured. It is still in a very early stage, mostly ideation and foundational building, but if this problem space resonates with you, feel free to explore the repo.**_

</details>

## License

MIT

<samp>
