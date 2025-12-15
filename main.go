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
	boldStyle      = lipgloss.NewStyle().Bold(true)
	greenStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	redStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	dimStyle       = lipgloss.NewStyle().Faint(true)
)

type state int

const (
	stateTyping state = iota
	stateResult
)

type model struct {
	lines       []string
	currentLine int
	input       string
	results     []bool
	state       state
}

func initialModel(lines []string) model {
	return model{
		lines:   lines,
		results: make([]bool, len(lines)),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateTyping:
			return m.handleTypingInput(msg)
		case stateResult:
			return m.handleResultInput(msg)
		}
	}
	return m, nil
}

func (m model) handleTypingInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit

	case tea.KeyEnter:
		// Check if input matches current line
		m.results[m.currentLine] = strings.TrimSpace(m.input) == strings.TrimSpace(m.lines[m.currentLine])
		m.currentLine++
		m.input = ""

		// If we've finished all lines, show results
		if m.currentLine >= len(m.lines) {
			m.state = stateResult
		}
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
	case stateTyping:
		// Show previous lines with results
		for i := 0; i < m.currentLine; i++ {
			if m.results[i] {
				b.WriteString(greenStyle.Render("✓ "))
			} else {
				b.WriteString(redStyle.Render("✗ "))
			}
			b.WriteString(dimStyle.Render(m.lines[i]))
			b.WriteString("\n")
		}

		// Show current line to type (bold)
		b.WriteString("\n")
		b.WriteString(boldStyle.Render(m.lines[m.currentLine]))
		b.WriteString("\n")

		// Show user input
		b.WriteString(m.input)
		b.WriteString("_") // Cursor
		b.WriteString("\n")

	case stateResult:
		// Show all lines with results
		b.WriteString("\n")
		for i, line := range m.lines {
			if m.results[i] {
				b.WriteString(greenStyle.Render("✓ "))
			} else {
				b.WriteString(redStyle.Render("✗ "))
			}
			b.WriteString(line)
			b.WriteString("\n")
		}

		// Calculate score
		correct := 0
		for _, r := range m.results {
			if r {
				correct++
			}
		}

		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("Score: %d/%d\n", correct, len(m.lines)))
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
