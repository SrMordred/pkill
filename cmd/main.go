package main

//lint:file-ignore ST1006 This is BS

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const NameColIndex = 1

var (
	// primary_color = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	header_style = lipgloss.NewStyle().
			BorderForeground(lipgloss.Color("238")).
			Align(lipgloss.Center)

	center_style = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(lipgloss.Color("238"))

	footer_style = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(lipgloss.Color("238"))
)

type Model struct {
	timer chan struct{}

	text_input  textinput.Model
	text_string string

	process_table BetterTable

	shortcuts Shortcuts
	help      help.Model
	err       error
}

type Shortcuts struct {
	killProcess      key.Binding
	killAllProcesses key.Binding
	up               key.Binding
	down             key.Binding
	left             key.Binding
	right            key.Binding
	tab              key.Binding
}

func (k Shortcuts) ShortHelp() []key.Binding {
	return []key.Binding{k.killProcess, k.killAllProcesses, k.up, k.down, k.left, k.right}
}

func (k Shortcuts) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.up, k.down, k.left, k.right},
		{k.killProcess, k.killAllProcesses},
	}
}

type TimerTick struct{}

func waitForTimer(timer chan struct{}) tea.Cmd {
	return func() tea.Msg {
		return TimerTick(<-timer)
	}
}

func createModel() Model {
	text_input := textinput.New()
	text_input.Placeholder = "Process Name"
	text_input.Focus()

	shortcuts := Shortcuts{
		killProcess: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "Kill Process"),
		),

		killAllProcesses: key.NewBinding(
			key.WithKeys("ctrl+j"), // This is ctrl+enter, don't ask me.
			key.WithHelp("ctrl+enter", "Kill All Processes with same name"),
		),

		up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "Table Up"),
		),

		down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "Table Down"),
		),

		left: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "Change Ordering Column Left"),
		),

		right: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "Change Ordering Column Right"),
		),

		tab: key.NewBinding(
			key.WithKeys("tab"),
		),
	}

	columns := []table.Column{
		{Title: "PID", Width: 7},
		{Title: "Name", Width: 15},
		{Title: "CPU%", Width: 7},
		{Title: "MEM%", Width: 7},
	}

	process_list := GetProcessList()

	better_table := MakeBetterTable()

	better_table.SetCols(columns)
	better_table.SetRows(process_list)
	better_table.SortBy("Name")

	timer := make(chan struct{})
	go func(timer chan struct{}) {
		for {
			time.Sleep(time.Second * 2)
			timer <- struct{}{}
		}
	}(timer)

	help := help.New()
	help.ShowAll = true

	return Model{
		text_input:    text_input,
		shortcuts:     shortcuts,
		process_table: better_table,
		timer:         timer,
		help:          help,
		err:           nil,
	}
}

func (self Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tea.EnterAltScreen, waitForTimer(self.timer))
}

func (self Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, self.shortcuts.up):
			self.process_table.MoveUp(1)
		case key.Matches(msg, self.shortcuts.down):
			self.process_table.MoveDown(1)

		case key.Matches(msg, self.shortcuts.left):
			self.process_table.SortByNext(-1)
		case key.Matches(msg, self.shortcuts.right):
			self.process_table.SortByNext(1)

		case key.Matches(msg, self.shortcuts.killProcess):
			row := self.process_table.GetSelected()
			_ = KillProcessByName(row[0])
			// Dont care if fail to kill error

		case key.Matches(msg, self.shortcuts.killAllProcesses):
			row := self.process_table.GetSelected()
			process_name := row[1]

			rows := self.process_table.GetAllRowsWithValue(NameColIndex, process_name)
			for _, row := range rows {
				_ = KillProcessByName(row[0])
			}

		}

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return self, tea.Quit
		}

	// Dont work on windows! :(
	case tea.WindowSizeMsg:
		self.setWidth(msg.Width)

	case TimerTick:
		self.process_table.SetRows(GetProcessList())
		return self, waitForTimer(self.timer)
	case error:
		self.err = msg
		return self, nil
	}

	// TODO: should create a custom Text Type
	var cmd1, cmd2 tea.Cmd
	self.text_input, cmd1 = self.text_input.Update(msg)
	text_string := self.text_input.Value()
	if strings.Compare(self.text_string, text_string) != 0 {
		self.text_string = text_string
		if len(self.text_string) == 0 {
			self.process_table.ClearSearch()
		} else {
			self.process_table.Search(text_string)
			self.process_table.ResetPosition()
			// self.header = text_string + " | " + strconv.Itoa(result)
		}
	}

	self.process_table.table, cmd2 = self.process_table.table.Update(msg)
	return self, tea.Batch(cmd1, cmd2)
}

func (self Model) setWidth(width int) {
	header_style.Width(width - 2)
	center_style.Width(width - 2)
	footer_style.Width(width - 2)
}

// VIEW

func (self Model) FooterView() string {
	helpView := self.help.View(self.shortcuts)

	return footer_style.Render(self.text_input.View()) + "\n\n" + helpView
}

func (self Model) View() string {

	header := "Process Killer"
	center := self.process_table.View()

	return lipgloss.JoinVertical(
		lipgloss.Top,
		header_style.Render(header),
		center_style.Render(center),
		self.FooterView(),
	)
}

func main() {
	p := tea.NewProgram(createModel())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
