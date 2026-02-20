# pulumi-ccstatusline

Shows Pulumi stack info in your Claude Code status line. If you're in a directory with a `Pulumi.yaml`, you'll see something like:

```
☁ dev | 8 resources | ✓ succeeded | 10d ago
```

Stack name is magenta, resources cyan, status green or red, time yellow.

## Install

### Homebrew

```bash
brew tap dirien/homebrew-dirien
brew install pulumi-ccstatusline
```

### From releases

Grab a binary from the [releases page](https://github.com/dirien/pulumi-ccstatusline/releases) and drop it in your PATH.

### From source

```bash
git clone https://github.com/dirien/pulumi-ccstatusline.git
cd pulumi-ccstatusline
make build
# Binary is at bin/pulumi-ccstatusline
```

## Usage

### With ccstatusline

[ccstatusline](https://github.com/sirmalloc/ccstatusline) gives you a TUI to configure built-in widgets (model, git, context, tokens). You add `pulumi-ccstatusline` as a Custom Command widget next to them.

**1. Set ccstatusline as your status line**

In `~/.claude/settings.json`:

```json
{
  "statusLine": "npx -y ccstatusline@latest"
}
```

Or run `npx ccstatusline@latest` and pick "Install to Claude Code" from the menu.

**2. Add the Pulumi widget**

Open the TUI with `npx ccstatusline@latest`:

1. Go to "Edit Lines"
2. Press `a` to add a widget
3. Use `←`/`→` to cycle to "Custom Command"
4. Set the command path to the binary (`pulumi-ccstatusline` if it's in PATH, or the full path)
5. Press `t` to set timeout to `5000`
6. Press `p` to enable "preserve colors"
7. `Ctrl+S` to save

You can also edit `~/.config/ccstatusline/settings.json` directly:

```json
{
  "id": "pulumi",
  "type": "custom-command",
  "commandPath": "pulumi-ccstatusline",
  "timeout": 5000,
  "preserveColors": true
}
```

### Standalone

You don't need ccstatusline. Point Claude Code at the binary directly:

```json
{
  "statusLine": {
    "type": "command",
    "command": "pulumi-ccstatusline"
  }
}
```

Or mix it into a shell script with your own segments:

```bash
#!/usr/bin/env bash
input=$(cat)

# Your other segments here...

# Pulumi info
pulumi_info=$(pulumi-ccstatusline <<< "$input" 2>/dev/null)
if [ -n "$pulumi_info" ]; then
  printf "  %s" "$pulumi_info"
fi

printf "\n"
```

## How it works

The binary reads Claude Code's JSON from stdin, pulls out the working directory, and checks for `Pulumi.yaml`. If it finds one, it calls `pulumi stack ls --json` and `pulumi stack history --json` to get the current stack's name, resource count, status, and last update time.

Results are cached for 30 seconds. The cache also watches the Pulumi workspace file in `~/.pulumi/workspaces/`, so switching stacks with `pulumi stack select` or deleting one with `pulumi stack rm` invalidates the cache immediately. No stale data.

If there's no Pulumi project in the directory, the binary exits silently and the widget stays hidden.

## Requirements

- [Pulumi CLI](https://www.pulumi.com/docs/install/) installed and authenticated
- A directory with `Pulumi.yaml`

## Development

```bash
make build      # Build binary to bin/
make test       # Run tests with race detector
make lint       # Run golangci-lint
make fmt        # Auto-fix lint issues
make snapshot   # Test goreleaser build locally
make clean      # Remove build artifacts
```

## License

Apache-2.0
