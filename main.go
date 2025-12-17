package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
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
	stateIntro state = iota
	stateSectionSelect
	stateTyping
	stateResult
)

// metadata holds song information from YAML front matter
type metadata struct {
	Title  string `yaml:"title"`
	Artist string `yaml:"artist"`
}

type section struct {
	name     string
	startIdx int // inclusive
	endIdx   int // exclusive
}

type model struct {
	meta            metadata  // song metadata from front matter
	allLines        []string  // all lines from the file
	lines           []string  // lines to practice (filtered by section)
	lineIndices     []int     // maps filtered line indices to allLines indices
	sections        []section // parsed sections
	selectedSection int       // -1 for all sections
	currentLine     int
	input           string
	results         []bool
	state           state
	hint            string // current hint to display (next word or full line)
	hintLevel       int    // 0 = no hint, 1 = word hint, 2 = full line hint
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

// getNextWordHint returns a hint for the next word the user should type.
// It looks at what the user has typed so far and returns the next word from the expected line.
func getNextWordHint(input, expected string) string {
	expectedWords := strings.Fields(expected)
	inputWords := strings.Fields(input)

	// If user is in the middle of typing a word (no trailing space), show that word
	if len(input) > 0 && !strings.HasSuffix(input, " ") {
		wordIdx := len(inputWords) - 1
		if wordIdx < len(expectedWords) {
			return expectedWords[wordIdx]
		}
		return ""
	}

	// User finished a word (trailing space or empty), show next word
	wordIdx := len(inputWords)
	if wordIdx < len(expectedWords) {
		return expectedWords[wordIdx]
	}
	return ""
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

func initialModel(meta metadata, lines []string) model {
	sections := parseSections(lines)

	// Skip intro screen if no metadata is set
	initialState := stateIntro
	if meta.Title == "" && meta.Artist == "" {
		initialState = stateSectionSelect
	}

	return model{
		meta:            meta,
		allLines:        lines,
		lines:           lines,
		sections:        sections,
		selectedSection: -1, // -1 means all sections
		results:         make([]bool, len(lines)),
		state:           initialState,
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
		case stateIntro:
			return m.handleIntroInput(msg)
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

func (m model) handleIntroInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit
	case tea.KeyEnter, tea.KeySpace:
		m.state = stateSectionSelect
		return m, nil
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
		m.hint = ""
		m.hintLevel = 0

		// Skip any comment lines
		m.skipComments()
		return m, nil

	case tea.KeyTab:
		// First tab: show next word, second tab: show full line
		if m.hintLevel == 0 {
			m.hint = getNextWordHint(m.input, m.lines[m.currentLine])
			m.hintLevel = 1
		} else if m.hintLevel == 1 {
			m.hint = m.lines[m.currentLine]
			m.hintLevel = 2
		}
		return m, nil

	case tea.KeyBackspace:
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
		m.hint = ""
		m.hintLevel = 0
		return m, nil

	case tea.KeyRunes:
		m.input += string(msg.Runes)
		m.hint = ""
		m.hintLevel = 0
		return m, nil

	case tea.KeySpace:
		m.input += " "
		m.hint = ""
		m.hintLevel = 0
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
	case stateIntro:
		b.WriteString("\n")
		if m.meta.Title != "" {
			b.WriteString(boldStyle.Render(m.meta.Title))
			b.WriteString("\n")
		}
		if m.meta.Artist != "" {
			b.WriteString(dimStyle.Render("by " + m.meta.Artist))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("Press Enter to continue..."))
		b.WriteString("\n")

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

		b.WriteString("\n")

		// Show user input
		b.WriteString(m.input)
		b.WriteString("_") // Cursor
		b.WriteString("\n")

		// Show hint if available
		if m.hint != "" {
			b.WriteString(dimStyle.Render("Hint: " + m.hint))
			b.WriteString("\n")
		}

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
	meta, lines, err := readFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	if len(lines) == 0 {
		fmt.Fprintln(os.Stderr, "Error: file is empty")
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel(meta, lines))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

func readFile(filename string) (metadata, []string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return metadata{}, nil, err
	}
	defer file.Close()

	var allContent []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allContent = append(allContent, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return metadata{}, nil, err
	}

	var meta metadata
	var lines []string
	startIdx := 0

	// Check for YAML front matter
	if len(allContent) > 0 && strings.TrimSpace(allContent[0]) == "---" {
		// Find closing ---
		endIdx := -1
		for i := 1; i < len(allContent); i++ {
			if strings.TrimSpace(allContent[i]) == "---" {
				endIdx = i
				break
			}
		}

		if endIdx > 0 {
			// Parse YAML between the delimiters
			yamlContent := strings.Join(allContent[1:endIdx], "\n")
			if err := yaml.Unmarshal([]byte(yamlContent), &meta); err != nil {
				return metadata{}, nil, fmt.Errorf("invalid YAML front matter: %w", err)
			}
			startIdx = endIdx + 1
		}
	}

	// Collect non-empty lines after front matter
	for i := startIdx; i < len(allContent); i++ {
		if strings.TrimSpace(allContent[i]) != "" {
			lines = append(lines, allContent[i])
		}
	}

	return meta, lines, nil
}
