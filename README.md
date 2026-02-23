<p align="center">
  <img src="assets/vecna.jpg" alt="Vecna" width="120" />
</p>

<p align="center">Minimalistic SSH TUI.</p>

## Install

```bash
curl -sSL https://raw.githubusercontent.com/shravan20/vecna/master/scripts/install.sh | sh
```

- **Go:** `go install github.com/shravan20/vecna@latest`
- **From source:**

```bash
make build
./bin/vecna
```

  `make install` also copies `config.example.yaml` to `~/.config/vecna/config.yaml` if that file doesn’t exist (so Run command has default commands).

## Usage

```bash
vecna          # Launch TUI
vecna version  # Print version
```

## Config

**Path:** `~/.config/vecna/config.yaml`

An example config is included in the repo. Copy it and edit (or run `make install`, which does this for you if no config exists yet):

```bash
mkdir -p ~/.config/vecna
cp config.example.yaml ~/.config/vecna/config.yaml
```

If **Run command** (r) shows “No saved commands”, add a `commands` block to your config (see example below) or replace your config with the example.

Example contents:

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

Idea: centralize control, reduce friction, make SSH-driven workflows better and more structured.

It is still in a very early stage, mostly ideation and foundational building, but if this problem space resonates with you, feel free to explore the repo.

</details>

## License

MIT
