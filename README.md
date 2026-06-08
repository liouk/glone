# glone

A TUI for browsing, cloning, and opening GitHub repos from your orgs.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Install

```
go install github.com/liouk/glone@latest
```

Requires:
- Go 1.25+
- [gh CLI](https://cli.github.com) (authenticated)

## Config

Create `~/.config/glone/config.yaml`:

```yaml
orgs:
  - name: openshift
    clone_dir: ~/redhat/openshift       # repos clone to ~/redhat/openshift/<repo>
  - name: kubernetes
    clone_dir: ~/redhat/kubernetes
  - name: liouk
    clone_dir: ~/liouk                  # original repos → ~/liouk/<repo>
    fork_clone_dirs:                    # forks routed by parent org:
      openshift: ~/redhat/liouk        #   openshift forks → ~/redhat/liouk/<repo>
      kubernetes: ~/redhat/liouk       #   kubernetes forks → ~/redhat/liouk/<repo>
                                        #   other forks → ~/liouk/<repo> (default)
```

Each org requires `name` and `clone_dir`. Use `fork_clone_dirs` to route forks to different directories based on their parent org. Forks from unlisted parent orgs fall back to `clone_dir`.

## Usage

```
glone
```

If multiple orgs are configured, you'll first pick one (press `1`-`9`/`0` to quick-select). Then browse repos with fuzzy filtering.

### Keybindings

| Key | Action |
|-----|--------|
| *type* | Fuzzy filter repos |
| `enter` | Clone repo (or open if already cloned) |
| `ctrl+s` | Shallow clone (`--depth 1`) |
| `ctrl+o` | Open in browser |
| `esc` / `ctrl+c` | Quit |

Already-cloned repos are marked with ✓ and `enter` opens them in Zed (or `$EDITOR`).

### Shell integration

`glone` prints the repo path to stdout on success, so you can:

```bash
cd "$(glone)"
```
