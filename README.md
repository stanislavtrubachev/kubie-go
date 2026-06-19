# Kubie-go

[![License: Zlib](https://img.shields.io/badge/License-Zlib-lightgrey.svg)](https://opensource.org/licenses/Zlib)

## Motivation

Imagine managing multiple clusters (`dev`, `test`, and `production`). You switch between them throughout the day using kubectx or kubens. The issue? These utilities modify the global Kubernetes context for your entire system. Every terminal window you have open shares the exact same cluster view.

This creates a risky workflow:
- You open a new shell and have no idea which cluster is currently active.
- You accidentally apply a critical manifest to `production` because you thought you were still on `dev`.
- You find yourself running `kubectl config current-context` every few minutes just to stay safe.

<p align="center">
  <img src="./assets/image_mascot_400x450.png" alt="Mascot Harold" width="400"/>
</p>

## Solution is Kubi-go

**Kubie-go** solves the main problem when working with multiple clusters: each of your terminals has its own life. The context and namespace switch independently in each shell, no accidental actions in the wrong cluster. Now you can:

- Switch context and namespace **independently** per terminal, no more global state.

- See the current cluster right in your shell (if you configure it), you always know exactly where you are.

- Load contexts from **multiple** configuration files* (for example, keep separate files for each cloud provider or environment).

This project is an offshoot from the original [kubie](https://github.com/kubie-org/kubie ). Since work with the source code repository has been [suspended](https://github.com/kubie-org/kubie/issues/385 ), I didn't want to lose such a handy tool. So I decided to do the maintenance myself and rewrite it in Go (*and maybe bring in something new*). It is a language that is ideal for DevOps engineers.

## How it works

When you run `kubie ctx <context>`, kubie-go:

1. Creates a temporary **kubeconfig** file containing only the selected context.
2. Spawns a new shell with `KUBECONFIG` pointing to that file.
3. Injects a shell prompt hook that prepends `[context|namespace]` to your shell.

Each shell gets its own isolated kubeconfig, so switching context in one terminal window does not affect any other.
Exiting the shell restores the previous state automatically.

## Installation

### Install script (recommended)

```sh
curl -fsSL https://raw.githubusercontent.com/stanislavtrubachev/kubie-go/main/install.sh | sh
```

Automatically detects your OS and architecture, downloads the binary from the latest release, and installs both `kubie-go` and the `kubie` alias to `/usr/local/bin`.

**To uninstall:**

```sh
curl -fsSL https://raw.githubusercontent.com/stanislavtrubachev/kubie-go/main/uninstall.sh | sh
```

### Homebrew (macOS)

If you have original `kubie` package installed, remove it before proceeding:
```sh
brew uninstall kubie
```

Then install:
```sh
brew tap stanislavtrubachev/kubie-go
brew install kubie-go
```

**Troubleshooting Installation Issues**
If you encounter the error Command Line Tools are too outdated, run:
```sh
sudo rm -rf /Library/Developer/CommandLineTools
xcode-select --install
brew update
```

### Build from source

You need Go 1.18 or later.

```sh
git clone https://github.com/stanislavtrubachev/kubie-go
cd kubie-go/src
go build -o kubie-go .
sudo mv kubie-go /usr/local/bin/kubie-go
sudo ln -sf /usr/local/bin/kubie-go /usr/local/bin/kubie
```

#### Maintaining compatibility

So that you can continue to use the familiar kubie command without having to retrain, I have kept the original interface and versioning. However, the binary file is now called `kubie-go` (so as not to conflict with the original if it is still installed).

For a seamless transition, it is enough to create an alias in your shell:


```sh
# for Bash
echo "alias kubie='kubie-go'" >> ~/.bashrc

# for Zsh
echo "alias kubie='kubie-go'" >> ~/.zshrc

# for Fish
echo "alias kubie='kubie-go'" >> ~/.config/fish/config.fish
```


After that, restart the shell configuration:
- `source ~/.bashrc` (or `source ~/.zshrc`) for Bash (or Zsh), or open a new terminal,
- for `fish` it's enough to open a new terminal.


## Auto-completion

To make working with `kubie-go` even faster and more convenient, command and flag autocompletions are supported, they suggest available contexts, namespaces, and options right in your terminal when you press Tab.

Completion scripts for Bash, Zsh, and Fish are located in the completion/ directory at the root of the repository. Copy the appropriate script to the correct location and source it in your shell configuration file.

**Bash:**
```bash
# Add to ~/.bashrc:
source /path/to/kubie/completion/kubie.bash
```

**Zsh:**
```zsh
# Add to ~/.zshrc:
source /path/to/kubie/completion/kubie.zsh

# Or install system-wide:
cp completion/kubie.zsh /usr/local/share/zsh/site-functions/_kubie
```

**Fish:**
```fish
cp completion/kubie.fish ~/.config/fish/completions/kubie.fish
```



## Advanced info

### Usage

When stdout is a terminal, `kubie ctx` and `kubie ns` open an interactive selector.
Navigate with arrow keys, press Enter to confirm, ESC or Ctrl-C to cancel (or `exit` command).
If [fzf](https://github.com/junegunn/fzf) is installed, it is used instead of the built-in selector.

---

- `kubie ctx` — display an interactive menu of contexts
- `kubie ctx <context>` — switch the current shell to the given context
- `kubie ctx -` — switch back to the previous context
- `kubie ctx <context> -r` — spawn a recursive shell in the given context
- `kubie ctx <context> -n <namespace>` — spawn a shell in the given context and namespace
- `kubie ns` — display an interactive menu of namespaces
- `kubie ns <namespace>` — switch the current shell to the given namespace
- `kubie ns -` — switch back to the previous namespace
- `kubie ns <namespace> -r` — spawn a recursive shell in the given namespace
- `kubie exec <context> <namespace> <cmd> [args...]` — execute a command in the given context and namespace
- `kubie exec <wildcard> <namespace> <cmd> [args...]` — execute in all contexts matched by the wildcard
- `kubie exec <wildcard> <namespace> --exit-early <cmd> [args...]` — same, but stop on first non-zero exit code
- `kubie export <context> <namespace>` — print the path to an isolated kubeconfig for the given context and namespace
- `kubie edit` — display an interactive menu of contexts to edit
- `kubie edit <context>` — open the file that contains this context in your editor
- `kubie edit-config` — open kubie's own config file in your editor
- `kubie delete [context]` — delete a context from its kubeconfig file
- `kubie lint` — check kubeconfig files for issues
- `kubie info ctx` — print the name of the current context
- `kubie info ns` — print the name of the current namespace
- `kubie info depth` — print the depth of recursive context nesting
- `kubie generate-completion --shell <bash|zsh|fish|xonsh|nu>` — print a completion script
- `kubie update` — check for the latest version and update the local binary if needed

### Settings

Kubie-go reads its configuration from `~/.kube/kubie.yaml`. All fields are optional.

```yaml
# Force kubie to use a particular shell instead of auto-detecting.
# Possible values: bash, fish, xonsh, zsh
# Default: auto-detect from process tree
shell: zsh

# Editor used by `kubie edit` and `kubie edit-config`.
# Default: $EDITOR or $VISUAL environment variable
default_editor: vim

# Paths where kubie looks for kubeconfig files.
configs:
    include:
        - ~/.kube/config
        - ~/.kube/*.yml
        - ~/.kube/*.yaml
        - ~/.kube/configs/*.yml
        - ~/.kube/configs/*.yaml
        - ~/.kube/kubie/*.yml
        - ~/.kube/kubie/*.yaml
    exclude:
        - ~/.kube/kubie.yaml

# Prompt settings.
prompt:
    # Disable kubie's prompt prefix inside a kubie shell.
    # Useful when your prompt theme already shows the kubernetes context.
    # Default: false
    disable: false

    # Show nesting depth when greater than 1.
    # Default: true
    show_depth: true

    # Use RPS1 (right-hand prompt) instead of PS1 in zsh.
    # Default: false
    zsh_use_rps1: false

    # Use right-hand prompt in fish.
    # Default: false
    fish_use_rprompt: false

    # Use right-hand prompt in xonsh.
    # Default: false
    xonsh_use_right_prompt: false

# Behavior settings.
behavior:
    # Namespace validation when switching with `kubie ns`.
    # Valid values:
    #   true:    Validate with `kubectl get namespaces` before switching.
    #   false:   Switch without validation.
    #   partial: Accept unique partial matches.
    # Default: true
    validate_namespaces: true

    # Whether to print "CONTEXT => ..." headers in `kubie exec` output.
    # Valid values: auto | always | never
    # Default: auto (prints only when stdout is a TTY)
    print_context_in_exec: auto

    # Parse the context argument to `kubie exec` and `kubie export` as a
    # space-delimited list of patterns.
    # Example: kubie exec 'dev-* staging-1' kube-system -- kubectl get po
    # Default: false
    allow_multiple_context_patterns: false

# Shell hooks — arbitrary shell code run at context start/stop.
hooks:
    # Run when entering a kubie context (inside the spawned shell).
    # Default: none
    start_ctx: >
        echo -en "\033]1; `kubie info ctx`|`kubie info ns` \007"

    # Run when exiting a kubie context.
    # Default: none
    stop_ctx: >
        echo -en "\033]1; $SHELL \007"

# Interactive selector settings.
# These options are passed to fzf when it is installed.
# When fzf is not installed, the built-in selector is used (arrow keys only).
fzf:
    # Enable mouse support.
    # Default: true
    mouse: true

    # Show the prompt at the top (reverse layout).
    # Default: false
    reverse: false

    # Case-insensitive search.
    # Default: false
    ignore_case: false

    # Hide the match-count info line.
    # Default: false
    info_hidden: false

    # Height of the selector. Percentage or fixed number of rows.
    # Default: unset (full screen)
    height: "50%"

    # Prompt string.
    # Default: unset
    prompt: "> "

    # Color scheme. See fzf documentation for supported values.
    # Default: unset
    color: "dark"
```

## Supported shells

| Shell  | Prompt | Completion |
|--------|--------|------------|
| zsh    | ✓      | ✓          |
| bash   | ✓      | ✓          |
| fish   | ✓      | ✓          |
| xonsh  | ✓      | —          |
