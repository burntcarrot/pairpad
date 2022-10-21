package main

import (
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func UI() {
	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		log.Fatal(err)
	}
}

type (
	errMsg error
)

type model struct {
	textInput textinput.Model
	textarea  textarea.Model
	err       error
	Quitting  bool
	LoggedIn  bool
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Username"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	ta := textarea.New()
	ta.Placeholder = "Write some text here..."
	// ta.Focus()

	return model{
		textInput: ti,
		textarea:  ta,
		err:       nil,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.Quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			m.LoggedIn = true
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	if !m.LoggedIn {
		m.textInput, cmd = m.textInput.Update(msg)
	} else {
		m.textarea, cmd = m.textarea.Update(msg)
		m.textarea.Focus()
	}

	return m, cmd
}

func loginView(m model) string {
	return fmt.Sprintf(
		"Enter username:\n\n%s\n\n%s",
		m.textInput.View(),
		"(esc to quit)",
	) + "\n"
}

func editorView(m model) string {
	return fmt.Sprintf(
		"Username: %s\n\n%s\n\n%s",
		m.textInput.Value(),
		m.textarea.View(),
		"(ctrl+c to quit)",
	) + "\n\n"
}

func (m model) View() string {
	var s string
	if m.Quitting {
		return "\n  See you later!\n\n"
	}
	if !m.LoggedIn {
		s = loginView(m)
	} else {
		s = editorView(m)
	}
	return s
}
