package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type orgPicker struct {
	orgs    []string
	cursor  int
	chosen  string
}

func newOrgPicker(orgs []string) orgPicker {
	return orgPicker{orgs: orgs}
}

func (o orgPicker) Update(msg tea.Msg) (orgPicker, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "enter":
			o.chosen = o.orgs[o.cursor]
			return o, nil
		case "up", "k":
			if o.cursor > 0 {
				o.cursor--
			}
		case "down", "j":
			if o.cursor < len(o.orgs)-1 {
				o.cursor++
			}
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(msg.String()[0]-'0') - 1
			if idx < len(o.orgs) {
				o.chosen = o.orgs[idx]
			}
		case "0":
			idx := 9
			if idx < len(o.orgs) {
				o.chosen = o.orgs[idx]
			}
		}
	}
	return o, nil
}

func (o orgPicker) View() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Select org"))
	b.WriteString("\n\n")

	for i, org := range o.orgs {
		num := i + 1
		if num == 10 {
			num = 0
		}

		numStr := dimStyle.Render(fmt.Sprintf("%d ", num))
		name := org
		cursor := "  "
		if i == o.cursor {
			cursor = selectedStyle.Render("> ")
			name = selectedStyle.Render(org)
		}

		b.WriteString(fmt.Sprintf("  %s%s%s\n", numStr, cursor, name))

		if i >= 9 {
			break
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  1-9/0 select • ↑/↓ navigate • enter confirm • q quit"))
	return b.String()
}
