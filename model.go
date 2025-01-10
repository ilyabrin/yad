package main

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	step   int
	name   string
	age    string
	path   string
	input  string
	cursor int //  field to track cursor position
	done   bool
	err    error
	errMsg string
}

func initialModel() Model {
	return Model{
		step:   0,
		done:   false,
		cursor: 0,
	}
}

func (m Model) Init() tea.Cmd {
	return nil // tea.SetWindowTitle("Bubble Tea Example")
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}
	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyLeft:
		return m.moveCursorLeft(), nil
	case tea.KeyRight:
		return m.moveCursorRight(), nil
	case tea.KeyHome:
		return m.moveCursorToStart(), nil
	case tea.KeyEnd:
		return m.moveCursorToEnd(), nil
	case tea.KeyBackspace:
		return m.handleBackspace(), nil
	case tea.KeyDelete:
		return m.handleDelete(), nil
	case tea.KeyEnter:
		return m.handleEnter()
	default:
		return m.handleDefaultKey(msg)
	}
}

func (m Model) moveCursorLeft() Model {
	if m.cursor > 0 {
		m.cursor--
	}
	return m
}

func (m Model) moveCursorRight() Model {
	if m.cursor < len(m.input) {
		m.cursor++
	}
	return m
}

func (m Model) moveCursorToStart() Model {
	m.cursor = 0
	return m
}

func (m Model) moveCursorToEnd() Model {
	m.cursor = len(m.input)
	return m
}

func (m Model) handleBackspace() Model {
	if m.cursor > 0 {
		m.input = m.input[:m.cursor-1] + m.input[m.cursor:]
		m.cursor--
	}
	return m
}

func (m Model) handleDelete() Model {
	if m.cursor < len(m.input) {
		m.input = m.input[:m.cursor] + m.input[m.cursor+1:]
	}
	return m
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	if m.done {
		return m, tea.Quit
	}

	switch m.step {
	case 0:
		return m.handleNameInput()
	case 1:
		return m.handleAgeInput()
	case 2:
		return m.handlePathInput()
	}

	return m, nil
}

func (m Model) handleNameInput() (tea.Model, tea.Cmd) {
	if len(m.input) == 0 {
		m.errMsg = "Name cannot be empty"
		return m, nil
	}
	m.name = m.input
	return m.nextStep(), nil
}

func (m Model) handleAgeInput() (tea.Model, tea.Cmd) {
	age, err := strconv.Atoi(m.input)
	if err != nil || age < 1 || age > 100 {
		m.errMsg = "Age must be a number between 1 and 100"
		return m, nil
	}
	m.age = m.input
	return m.nextStep(), nil
}

func (m Model) handlePathInput() (tea.Model, tea.Cmd) {
	if len(m.input) == 0 {
		m.errMsg = "Path cannot be empty"
		return m, nil
	}
	m.path = m.input
	m.done = true
	fmt.Printf("\n\nProject initialized successfully!\n Name: %s\n Age: %s\n Path: %s\n\n", m.name, m.age, m.path)
	return m.nextStep(), nil
}

func (m Model) nextStep() Model {
	m.input = ""
	m.cursor = 0
	m.errMsg = ""
	m.step++
	return m
}

func (m Model) handleDefaultKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if !m.done && msg.String() != "" {
		m.input = m.input[:m.cursor] + msg.String() + m.input[m.cursor:]
		m.cursor++
	}
	return m, nil
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	if m.done {
		return "Press enter to exit."
	}

	var sb strings.Builder

	if m.errMsg != "" {
		sb.WriteString(fmt.Sprintf("Error: %s\n", m.errMsg))
	}

	// TODO: testing purpose only
	switch m.step {
	case 0:
		sb.WriteString("Enter your name: ")
	case 1:
		sb.WriteString(fmt.Sprintf("Name: %s\n", m.name))
		sb.WriteString("Enter your age (1-100): ")
	case 2:
		sb.WriteString(fmt.Sprintf("Name: %s\n", m.name))
		sb.WriteString(fmt.Sprintf("Age: %s\n", m.age))
		sb.WriteString("Enter project path: ")
	}

	// Display input with cursor
	beforeCursor := m.input[:m.cursor]
	afterCursor := m.input[m.cursor:]
	cursor := "█" // You can change this to any cursor character you prefer

	sb.WriteString(beforeCursor)
	sb.WriteString(cursor)
	sb.WriteString(afterCursor)

	return sb.String()
}
