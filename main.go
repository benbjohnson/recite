package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	boldStyle    = lipgloss.NewStyle().Bold(true)
	greenStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	redStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	dimStyle     = lipgloss.NewStyle().Faint(true)
	headerStyle = lipgloss.NewStyle().Bold(true).Underline(true)
)

type state int

const (
	stateModeSelect state = iota
	stateSectionSelect
	stateTyping
	stateResult
)

type section struct {
	name     string
	startIdx int // inclusive
	endIdx   int // exclusive
}

type mode int

const (
	modePractice mode = iota
	modeMemory
)

type model struct {
	allLines        []string  // all lines from the file
	lines           []string  // lines to practice (filtered by section)
	lineIndices     []int     // maps filtered line indices to allLines indices
	sections        []section // parsed sections
	selectedSection int       // -1 for all sections
	currentLine     int
	input           string
	results         []bool
	state           state
	mode            mode
}

func isComment(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "#")
}

// headerText strips the leading '#' and whitespace from a comment line
func headerText(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "#")
	return strings.TrimSpace(line)
}

// normalize removes all punctuation and spaces, and lowercases the string
// for forgiving comparison of user input to expected lyrics
func normalize(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// parseSections extracts sections from lines. Each section starts with a # comment.
// Lines before the first # are grouped into an "Intro" section if present.
func parseSections(lines []string) []section {
	var sections []section
	var currentSection *section

	for i, line := range lines {
		if isComment(line) {
			// Close previous section
			if currentSection != nil {
				currentSection.endIdx = i
				sections = append(sections, *currentSection)
			}
			// Start new section
			currentSection = &section{
				name:     headerText(line),
				startIdx: i,
			}
		} else if currentSection == nil {
			// Lines before first section header
			currentSection = &section{
				name:     "Intro",
				startIdx: 0,
			}
		}
	}

	// Close final section
	if currentSection != nil {
		currentSection.endIdx = len(lines)
		sections = append(sections, *currentSection)
	}

	return sections
}

func initialModel(lines []string) model {
	sections := parseSections(lines)

	return model{
		allLines:        lines,
		lines:           lines,
		sections:        sections,
		selectedSection: -1, // -1 means all sections
		results:         make([]bool, len(lines)),
		state:           stateModeSelect,
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
		case stateSectionSelect:
			return m.handleSectionSelectInput(msg)
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
			m.state = stateSectionSelect
			return m, nil
		} else if key == "2" {
			m.mode = modeMemory
			m.state = stateSectionSelect
			return m, nil
		}
	}

	return m, nil
}

// selectSection filters lines based on the selected section index.
// Pass -1 to select all sections.
func (m *model) selectSection(sectionIdx int) {
	m.selectedSection = sectionIdx

	if sectionIdx < 0 || sectionIdx >= len(m.sections) {
		// All sections
		m.lines = m.allLines
		m.lineIndices = nil
		m.results = make([]bool, len(m.lines))
	} else {
		// Specific section
		sec := m.sections[sectionIdx]
		m.lines = m.allLines[sec.startIdx:sec.endIdx]
		m.lineIndices = make([]int, len(m.lines))
		for i := range m.lines {
			m.lineIndices[i] = sec.startIdx + i
		}
		m.results = make([]bool, len(m.lines))
	}

	m.currentLine = 0
	m.input = ""
}

func (m model) handleSectionSelectInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit

	case tea.KeyRunes:
		key := string(msg.Runes)
		// "a" or "A" selects all sections
		if key == "a" || key == "A" {
			m.selectSection(-1)
			m.state = stateTyping
			m.skipComments()
			return m, nil
		}

		// Number keys 1-9 select specific sections
		if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
			idx := int(key[0] - '1') // Convert '1' to 0, '2' to 1, etc.
			if idx < len(m.sections) {
				m.selectSection(idx)
				m.state = stateTyping
				m.skipComments()
				return m, nil
			}
		}
	}

	return m, nil
}

func (m model) handleTypingInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit

	case tea.KeyEnter:
		// Check if input matches current line (ignoring punctuation, spaces, and case)
		m.results[m.currentLine] = normalize(m.input) == normalize(m.lines[m.currentLine])
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

	case stateSectionSelect:
		b.WriteString("\n")
		b.WriteString(boldStyle.Render("Select Section:"))
		b.WriteString("\n\n")
		b.WriteString("  a. All sections\n")
		for i, sec := range m.sections {
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, sec.name))
		}
		b.WriteString("\n")
		b.WriteString("Press a or 1-9 to select: ")

	case stateTyping:
		// Show previous lines with results
		for i := 0; i < m.currentLine; i++ {
			if isComment(m.lines[i]) {
				b.WriteString("\n")
				b.WriteString(headerStyle.Render(headerText(m.lines[i])))
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
		for i, line := range m.lines {
			if isComment(line) {
				b.WriteString("\n")
				b.WriteString(headerStyle.Render(headerText(line)))
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
