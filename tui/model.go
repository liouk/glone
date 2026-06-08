package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenOrg screen = iota
	screenRepo
	screenResult
)

type reposCachedMsg struct {
	repos []repoItem
}

type reposLoadedMsg struct {
	repos []repoItem
}

type reposErrorMsg struct {
	err error
}

type cloneDoneMsg struct {
	path       string
	repoName   string
	message    string
	err        error
	openEditor bool
}

type editorOpenedMsg struct {
	path    string
	message string
	err     error
}

type Org struct {
	Name          string
	CloneDir      string
	ForkCloneDirs map[string]string
}

type Model struct {
	screen     screen
	orgPicker  orgPicker
	repoPicker repoPicker
	result     resultScreen
	orgs       []Org
	editor     string
	quitting   bool
	resultText string
}

func New(orgs []Org, editor string) Model {
	m := Model{
		orgs:   orgs,
		editor: editor,
	}

	if len(orgs) == 1 {
		m.screen = screenRepo
		m.repoPicker = newRepoPicker()
		m.repoPicker.org = orgs[0].Name
		m.repoPicker.cloneDir = orgs[0].CloneDir
		m.repoPicker.forkCloneDirs = orgs[0].ForkCloneDirs
	} else {
		names := make([]string, len(orgs))
		for i, o := range orgs {
			names[i] = o.Name
		}
		m.screen = screenOrg
		m.orgPicker = newOrgPicker(names)
	}

	return m
}

func (m Model) Result() string {
	return m.resultText
}

func (m Model) Init() tea.Cmd {
	if m.screen == screenRepo {
		return tea.Batch(
			m.repoPicker.spinner.Tick,
			loadCachedRepos(m.orgs[0].Name, m.orgs[0].CloneDir, m.orgs[0].ForkCloneDirs),
			fetchRepos(m.orgs[0].Name, m.orgs[0].CloneDir, m.orgs[0].ForkCloneDirs),
		)
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			m.quitting = true
			return m, tea.Quit
		case "q":
			if m.screen != screenRepo {
				m.quitting = true
				return m, tea.Quit
			}
		}

	case reposCachedMsg:
		m.repoPicker.allItems = msg.repos
		m.repoPicker.loading = false
		m.repoPicker.refreshing = true
		m.repoPicker.applyFilter()
		return m, nil

	case reposLoadedMsg:
		m.repoPicker.allItems = msg.repos
		m.repoPicker.loading = false
		m.repoPicker.refreshing = false
		m.repoPicker.applyFilter()
		return m, nil

	case reposErrorMsg:
		if len(m.repoPicker.allItems) > 0 {
			m.repoPicker.refreshing = false
			return m, nil
		}
		m.screen = screenResult
		var cmd tea.Cmd
		m.result, cmd = newResult(msg.err.Error(), true)
		return m, cmd

	case cloneDoneMsg:
		if msg.err != nil {
			m.screen = screenResult
			var cmd tea.Cmd
			m.result, cmd = newResult(msg.err.Error(), true)
			return m, cmd
		}
		if msg.openEditor && m.editor != "" {
			return m, m.openEditor(repoItem{name: msg.repoName}, msg.path)
		}
		m.screen = screenResult
		m.resultText = msg.path
		var cmd tea.Cmd
		m.result, cmd = newResult(msg.message, false)
		return m, cmd

	case editorOpenedMsg:
		if msg.err != nil {
			m.screen = screenResult
			var cmd tea.Cmd
			m.result, cmd = newResult(msg.err.Error(), true)
			return m, cmd
		}
		m.screen = screenResult
		m.resultText = msg.path
		var cmd tea.Cmd
		m.result, cmd = newResult(msg.message, false)
		return m, cmd

	case autoDismissMsg:
		m.quitting = true
		return m, tea.Quit
	}

	switch m.screen {
	case screenOrg:
		return m.updateOrg(msg)
	case screenRepo:
		return m.updateRepo(msg)
	case screenResult:
		return m, nil
	}

	return m, nil
}

func (m Model) updateOrg(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.orgPicker, cmd = m.orgPicker.Update(msg)

	if m.orgPicker.chosen != "" {
		var chosenOrg Org
		for _, o := range m.orgs {
			if o.Name == m.orgPicker.chosen {
				chosenOrg = o
				break
			}
		}
		m.screen = screenRepo
		m.repoPicker = newRepoPicker()
		m.repoPicker.org = chosenOrg.Name
		m.repoPicker.cloneDir = chosenOrg.CloneDir
		m.repoPicker.forkCloneDirs = chosenOrg.ForkCloneDirs
		return m, tea.Batch(
			m.repoPicker.spinner.Tick,
			loadCachedRepos(chosenOrg.Name, chosenOrg.CloneDir, chosenOrg.ForkCloneDirs),
			fetchRepos(chosenOrg.Name, chosenOrg.CloneDir, chosenOrg.ForkCloneDirs),
		)
	}

	return m, cmd
}

func (m Model) updateRepo(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.repoPicker, cmd = m.repoPicker.Update(msg)

	if m.repoPicker.action.kind != actionNone {
		action := m.repoPicker.action
		m.repoPicker.action = repoAction{}

		switch action.kind {
		case actionBrowser:
			return m, m.openBrowser(action.item)
		case actionDeepClone, actionShallowClone:
			m.repoPicker.loading = true
			m.repoPicker.loadingMsg = fmt.Sprintf("Cloning %s…", action.item.name)
			return m, tea.Batch(
				m.repoPicker.spinner.Tick,
				m.doClone(action),
			)
		case actionOpen:
			return m, m.openEditor(action.item, "")
		}
	}

	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	switch m.screen {
	case screenOrg:
		return m.orgPicker.View()
	case screenRepo:
		return m.repoPicker.View()
	case screenResult:
		return m.result.View()
	}
	return ""
}

func cachedReposToItems(repos []cachedRepo, cloneDir string, forkCloneDirs map[string]string) []repoItem {
	items := make([]repoItem, len(repos))
	for i, r := range repos {
		dir := resolveCloneDir(cloneDir, forkCloneDirs, r.ParentOrg)
		items[i] = repoItem{
			name:        r.Name,
			url:         r.URL,
			description: r.Description,
			parentOrg:   r.ParentOrg,
			cloned:      isRepoCloned(dir, r.Name),
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].name < items[j].name
	})
	return items
}

func loadCachedRepos(org, cloneDir string, forkCloneDirs map[string]string) tea.Cmd {
	return func() tea.Msg {
		repos, err := readCache(org)
		if err != nil || len(repos) == 0 {
			return nil
		}
		return reposCachedMsg{repos: cachedReposToItems(repos, cloneDir, forkCloneDirs)}
	}
}

func fetchRepos(org, cloneDir string, forkCloneDirs map[string]string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("gh", "repo", "list", org,
			"--json", "name,url,description,isFork,parent",
			"--limit", "1000",
		)
		out, err := cmd.Output()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return reposErrorMsg{err: fmt.Errorf("gh: %s", string(exitErr.Stderr))}
			}
			return reposErrorMsg{err: err}
		}

		type ghRepo struct {
			Name        string `json:"name"`
			URL         string `json:"url"`
			Description string `json:"description"`
			IsFork      bool   `json:"isFork"`
			Parent      *struct {
				Owner struct {
					Login string `json:"login"`
				} `json:"owner"`
			} `json:"parent"`
		}

		var repos []ghRepo
		if err := jsonUnmarshal(out, &repos); err != nil {
			return reposErrorMsg{err: err}
		}

		cached := make([]cachedRepo, len(repos))
		for i, r := range repos {
			parentOrg := ""
			if r.IsFork && r.Parent != nil {
				parentOrg = r.Parent.Owner.Login
			}
			cached[i] = cachedRepo{
				Name:        r.Name,
				URL:         r.URL,
				Description: r.Description,
				IsFork:      r.IsFork,
				ParentOrg:   parentOrg,
			}
		}
		writeCache(org, cached)

		return reposLoadedMsg{repos: cachedReposToItems(cached, cloneDir, forkCloneDirs)}
	}
}

func resolveCloneDir(cloneDir string, forkCloneDirs map[string]string, parentOrg string) string {
	if parentOrg != "" {
		if dir, ok := forkCloneDirs[parentOrg]; ok {
			return dir
		}
	}
	return cloneDir
}

func (m Model) cloneDirFor(item repoItem) string {
	return resolveCloneDir(m.repoPicker.cloneDir, m.repoPicker.forkCloneDirs, item.parentOrg)
}

func (m Model) doClone(action repoAction) tea.Cmd {
	return func() tea.Msg {
		shallow := action.kind == actionShallowClone
		dir := m.cloneDirFor(action.item)
		path, err := cloneRepoCmd(action.item.url, dir, action.item.name, shallow)
		if err != nil {
			return cloneDoneMsg{err: err}
		}
		kind := "cloned"
		if shallow {
			kind = "shallow cloned"
		}
		return cloneDoneMsg{
			path:       path,
			repoName:   action.item.name,
			message:    fmt.Sprintf("%s to %s", kind, path),
			openEditor: true,
		}
	}
}

func (m Model) openBrowser(item repoItem) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("gh", "repo", "view", "--web", m.repoPicker.org+"/"+item.name)
		if err := cmd.Run(); err != nil {
			return cloneDoneMsg{err: fmt.Errorf("could not open browser: %w", err)}
		}
		return cloneDoneMsg{message: "opened in browser"}
	}
}

func (m Model) openEditor(item repoItem, path string) tea.Cmd {
	return func() tea.Msg {
		if path == "" {
			dir := m.cloneDirFor(item)
			path = filepath.Join(dir, item.name)
		}
		_, err := openEditorCmd(m.editor, filepath.Dir(path), filepath.Base(path))
		if err != nil {
			return editorOpenedMsg{err: err}
		}
		return editorOpenedMsg{
			path:    path,
			message: fmt.Sprintf("opened %s", path),
		}
	}
}
