package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	boldStyle    = lipgloss.NewStyle().Bold(true)
	greenStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	redStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	dimStyle     = lipgloss.NewStyle().Faint(true)
	commentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Gray
)

type state int

const (
	stateModeSelect state = iota
	stateTyping
	stateResult
)

type mode int

const (
	modePractice mode = iota
	modeMemory
)

type model struct {
	lines       []string
	currentLine int
	input       string
	results     []bool
	state       state
	mode        mode
}

func isComment(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "#")
}

func initialModel(lines []string) model {
	return model{
		lines:   lines,
		results: make([]bool, len(lines)),
		state:   stateModeSelect,
	}
}

// skipComments advances currentLine past any comment lines
func (m *model) skipComments() {
	for m.currentLine < len(m.lines) && isComment(m.lines[m.currentLine]) {
		m.results[m.currentLine] = true // Comments are always "correct"
		m.currentLine++
	}
	if m.currentLine >= len(m.lines) {
		m.state = stateResult
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateModeSelect:
			return m.handleModeSelectInput(msg)
		case stateTyping:
			return m.handleTypingInput(msg)
		case stateResult:
			return m.handleResultInput(msg)
		}
	}
	return m, nil
}

func (m model) handleModeSelectInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit

	case tea.KeyRunes:
		key := string(msg.Runes)
		if key == "1" {
			m.mode = modePractice
			m.state = stateTyping
			m.skipComments()
			return m, nil
		} else if key == "2" {
			m.mode = modeMemory
			m.state = stateTyping
			m.skipComments()
			return m, nil
		}
	}

	return m, nil
}

func (m model) handleTypingInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit

	case tea.KeyEnter:
		// Check if input matches current line (case insensitive)
		m.results[m.currentLine] = strings.EqualFold(strings.TrimSpace(m.input), strings.TrimSpace(m.lines[m.currentLine]))
		m.currentLine++
		m.input = ""

		// Skip any comment lines
		m.skipComments()
		return m, nil

	case tea.KeyBackspace:
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
		return m, nil

	case tea.KeyRunes:
		m.input += string(msg.Runes)
		return m, nil

	case tea.KeySpace:
		m.input += " "
		return m, nil
	}

	return m, nil
}

func (m model) handleResultInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit

	case tea.KeyRunes:
		key := string(msg.Runes)
		if key == "y" || key == "Y" {
			// Restart
			m.currentLine = 0
			m.input = ""
			m.results = make([]bool, len(m.lines))
			m.state = stateTyping
			m.skipComments()
			return m, nil
		} else if key == "n" || key == "N" {
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	switch m.state {
	case stateModeSelect:
		b.WriteString("\n")
		b.WriteString(boldStyle.Render("Select Mode:"))
		b.WriteString("\n\n")
		b.WriteString("  1. Practice - see the line, then type it\n")
		b.WriteString("  2. Memory - type from memory\n")
		b.WriteString("\n")
		b.WriteString("Press 1 or 2 to select: ")

	case stateTyping:
		// Show previous lines with results
		for i := 0; i < m.currentLine; i++ {
			if isComment(m.lines[i]) {
				b.WriteString(commentStyle.Render(m.lines[i]))
			} else if m.results[i] {
				b.WriteString(greenStyle.Render("✓ "))
				b.WriteString(dimStyle.Render(m.lines[i]))
			} else {
				b.WriteString(redStyle.Render("✗ "))
				b.WriteString(dimStyle.Render(m.lines[i]))
			}
			b.WriteString("\n")
		}

		// Show current line to type (bold) - only in practice mode
		b.WriteString("\n")
		if m.mode == modePractice {
			b.WriteString(boldStyle.Render(m.lines[m.currentLine]))
		} else {
			b.WriteString(dimStyle.Render("(type from memory)"))
		}
		b.WriteString("\n")

		// Show user input
		b.WriteString(m.input)
		b.WriteString("_") // Cursor
		b.WriteString("\n")

	case stateResult:
		// Show all lines with results
		b.WriteString("\n")
		for i, line := range m.lines {
			if isComment(line) {
				b.WriteString(commentStyle.Render(line))
			} else if m.results[i] {
				b.WriteString(greenStyle.Render("✓ "))
				b.WriteString(line)
			} else {
				b.WriteString(redStyle.Render("✗ "))
				b.WriteString(line)
			}
			b.WriteString("\n")
		}

		// Calculate score (excluding comments)
		correct := 0
		total := 0
		for i, line := range m.lines {
			if !isComment(line) {
				total++
				if m.results[i] {
					correct++
				}
			}
		}

		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("Score: %d/%d\n", correct, total))
		b.WriteString("\n")
		b.WriteString("Try again? (y/n) ")
	}

	return b.String()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: recite <lyrics-file>")
		os.Exit(1)
	}

	filename := os.Args[1]
	lines, err := readLines(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	if len(lines) == 0 {
		fmt.Fprintln(os.Stderr, "Error: file is empty")
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel(lines))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

func readLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Skip empty lines
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}

	return lines, scanner.Err()
}
