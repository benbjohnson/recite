package main

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSectionSelect(t *testing.T) {
	t.Run("pressing a selects all sections", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"# Verse 1", "Line one", "# Chorus", "Line two"})
		m.state = stateSectionSelect

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		m = newModel.(model)

		if m.state != stateTyping {
			t.Errorf("state = %v, want stateTyping", m.state)
		}
		if m.selectedSection != -1 {
			t.Errorf("selectedSection = %d, want -1 (all sections)", m.selectedSection)
		}
		if len(m.lines) != 4 {
			t.Errorf("len(lines) = %d, want 4", len(m.lines))
		}
	})

	t.Run("pressing 1 selects first section", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"# Verse 1", "Line one", "# Chorus", "Line two"})
		m.state = stateSectionSelect

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
		m = newModel.(model)

		if m.state != stateTyping {
			t.Errorf("state = %v, want stateTyping", m.state)
		}
		if m.selectedSection != 0 {
			t.Errorf("selectedSection = %d, want 0", m.selectedSection)
		}
		if len(m.lines) != 2 {
			t.Errorf("len(lines) = %d, want 2 (Verse 1 header + Line one)", len(m.lines))
		}
	})

	t.Run("pressing 2 selects second section", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"# Verse 1", "Line one", "# Chorus", "Line two"})
		m.state = stateSectionSelect

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
		m = newModel.(model)

		if m.state != stateTyping {
			t.Errorf("state = %v, want stateTyping", m.state)
		}
		if m.selectedSection != 1 {
			t.Errorf("selectedSection = %d, want 1", m.selectedSection)
		}
		if len(m.lines) != 2 {
			t.Errorf("len(lines) = %d, want 2 (Chorus header + Line two)", len(m.lines))
		}
	})

	t.Run("section selection skips leading comments", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"# Verse 1", "Line one"})
		m.state = stateSectionSelect

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		m = newModel.(model)

		if m.currentLine != 1 {
			t.Errorf("currentLine = %d, want 1 (should skip comment)", m.currentLine)
		}
	})

	t.Run("ctrl+c quits from section select", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"# Verse 1", "Line one"})
		m.state = stateSectionSelect
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

		if cmd == nil {
			t.Error("expected quit command")
		}
	})

	t.Run("invalid section number does nothing", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"# Verse 1", "Line one"})
		m.state = stateSectionSelect

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'9'}})
		m = newModel.(model)

		if m.state != stateSectionSelect {
			t.Errorf("state = %v, want stateSectionSelect (invalid number should do nothing)", m.state)
		}
	})
}

func TestParseSections(t *testing.T) {
	t.Run("parses multiple sections", func(t *testing.T) {
		lines := []string{"# Verse 1", "Line one", "# Chorus", "Line two"}
		sections := parseSections(lines)

		if len(sections) != 2 {
			t.Fatalf("len(sections) = %d, want 2", len(sections))
		}
		if sections[0].name != "Verse 1" {
			t.Errorf("sections[0].name = %q, want %q", sections[0].name, "Verse 1")
		}
		if sections[0].startIdx != 0 || sections[0].endIdx != 2 {
			t.Errorf("sections[0] range = [%d, %d), want [0, 2)", sections[0].startIdx, sections[0].endIdx)
		}
		if sections[1].name != "Chorus" {
			t.Errorf("sections[1].name = %q, want %q", sections[1].name, "Chorus")
		}
		if sections[1].startIdx != 2 || sections[1].endIdx != 4 {
			t.Errorf("sections[1] range = [%d, %d), want [2, 4)", sections[1].startIdx, sections[1].endIdx)
		}
	})

	t.Run("creates Intro for lines before first header", func(t *testing.T) {
		lines := []string{"Intro line", "# Verse 1", "Verse line"}
		sections := parseSections(lines)

		if len(sections) != 2 {
			t.Fatalf("len(sections) = %d, want 2", len(sections))
		}
		if sections[0].name != "Intro" {
			t.Errorf("sections[0].name = %q, want %q", sections[0].name, "Intro")
		}
		if sections[0].startIdx != 0 || sections[0].endIdx != 1 {
			t.Errorf("sections[0] range = [%d, %d), want [0, 1)", sections[0].startIdx, sections[0].endIdx)
		}
	})

	t.Run("handles file with no sections", func(t *testing.T) {
		lines := []string{"Line one", "Line two"}
		sections := parseSections(lines)

		if len(sections) != 1 {
			t.Fatalf("len(sections) = %d, want 1", len(sections))
		}
		if sections[0].name != "Intro" {
			t.Errorf("sections[0].name = %q, want %q", sections[0].name, "Intro")
		}
	})

	t.Run("handles empty file", func(t *testing.T) {
		lines := []string{}
		sections := parseSections(lines)

		if len(sections) != 0 {
			t.Errorf("len(sections) = %d, want 0", len(sections))
		}
	})
}

func TestIsComment(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{"# This is a comment", true},
		{"#Comment without space", true},
		{"  # Indented comment", true},
		{"Regular line", false},
		{"Not a # comment", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := isComment(tt.line); got != tt.expected {
				t.Errorf("isComment(%q) = %v, want %v", tt.line, got, tt.expected)
			}
		})
	}
}

func TestHeaderText(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"# Verse 1", "Verse 1"},
		{"#Chorus", "Chorus"},
		{"  # Indented header", "Indented header"},
		{"#  Multiple spaces", "Multiple spaces"},
		{"# ", ""},
		{"#", ""},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := headerText(tt.line); got != tt.expected {
				t.Errorf("headerText(%q) = %q, want %q", tt.line, got, tt.expected)
			}
		})
	}
}

func TestGetNextWordHint(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hint     string
	}{
		{"empty input returns first word", "", "hello world today", "hello"},
		{"after first word returns second", "hello ", "hello world today", "world"},
		{"mid-word returns current word", "hel", "hello world today", "hello"},
		{"after two words returns third", "hello world ", "hello world today", "today"},
		{"mid second word returns second", "hello wor", "hello world today", "world"},
		{"all words typed returns empty", "hello world today ", "hello world today", ""},
		{"beyond expected returns empty", "hello world today extra ", "hello world today", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getNextWordHint(tt.input, tt.expected); got != tt.hint {
				t.Errorf("getNextWordHint(%q, %q) = %q, want %q", tt.input, tt.expected, got, tt.hint)
			}
		})
	}
}

func TestWordsMatch(t *testing.T) {
	tests := []struct {
		a, b  string
		match bool
	}{
		{"hello", "hello", true},
		{"Hello", "hello", true},
		{"stayin", "staying", true},
		{"staying", "stayin", true},
		{"nothin", "nothing", true},
		{"nothing", "nothin", true},
		{"believin", "believing", true},
		{"runnin", "running", true},
		{"stayin'", "staying", true},
		{"hello", "world", false},
		{"sin", "sing", false}, // "sin" is a different word, not g-dropping
		{"in", "ing", false},   // too short to be g-dropping
		{"win", "wing", false}, // different words
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			if got := wordsMatch(tt.a, tt.b); got != tt.match {
				t.Errorf("wordsMatch(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.match)
			}
		})
	}
}

func TestLinesMatch(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		match    bool
	}{
		{"exact match", "hello world", "hello world", true},
		{"case insensitive", "Hello World", "hello world", true},
		{"g-dropping single word", "stayin alive", "staying alive", true},
		{"g-dropping multiple words", "keepin on movin", "keeping on moving", true},
		{"punctuation ignored", "dont stop", "don't stop", true},
		{"mixed g-dropping and punctuation", "dont stop believin", "don't stop believin'", true},
		{"wrong word count", "hello", "hello world", false},
		{"wrong words", "hello world", "goodbye world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := linesMatch(tt.input, tt.expected); got != tt.match {
				t.Errorf("linesMatch(%q, %q) = %v, want %v", tt.input, tt.expected, got, tt.match)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "helloworld"},
		{"hello world", "helloworld"},
		{"Hello, World!", "helloworld"},
		{"Don't stop", "dontstop"},
		{"It's a test", "itsatest"},
		{"  spaces  everywhere  ", "spaceseverywhere"},
		{"UPPERCASE", "uppercase"},
		{"123 numbers 456", "123numbers456"},
		{"", ""},
		{"...!!!", ""},
		{"a-b-c", "abc"},
		{"(parentheses)", "parentheses"},
		{"question?", "question"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalize(tt.input); got != tt.expected {
				t.Errorf("normalize(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestInitialModel(t *testing.T) {
	t.Run("basic initialization", func(t *testing.T) {
		lines := []string{"Line one", "Line two"}
		m := initialModel(metadata{}, lines)

		if m.currentLine != 0 {
			t.Errorf("currentLine = %d, want 0", m.currentLine)
		}
		if m.state != stateSectionSelect {
			t.Errorf("state = %v, want stateSectionSelect", m.state)
		}
		if len(m.results) != 2 {
			t.Errorf("len(results) = %d, want 2", len(m.results))
		}
	})
}

func TestHandleTypingInput(t *testing.T) {
	t.Run("correct input", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Hello world", "Second line"})
		m.state = stateTyping
		m.input = "Hello world"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if !m.results[0] {
			t.Error("first line should be marked correct")
		}
		if m.currentLine != 1 {
			t.Errorf("currentLine = %d, want 1", m.currentLine)
		}
	})

	t.Run("case insensitive comparison", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Hello World"})
		m.state = stateTyping
		m.input = "hello world"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if !m.results[0] {
			t.Error("case insensitive match should be correct")
		}
	})

	t.Run("incorrect input", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Hello world"})
		m.state = stateTyping
		m.input = "Wrong input"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if m.results[0] {
			t.Error("incorrect input should be marked wrong")
		}
	})

	t.Run("whitespace trimming", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Hello world"})
		m.state = stateTyping
		m.input = "  Hello world  "

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if !m.results[0] {
			t.Error("whitespace-trimmed match should be correct")
		}
	})

	t.Run("ignores punctuation", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Don't stop believin'"})
		m.state = stateTyping
		m.input = "dont stop believin"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if !m.results[0] {
			t.Error("punctuation-free input should match")
		}
	})

	t.Run("ignores extra spaces", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Hello world"})
		m.state = stateTyping
		m.input = "Hello    world"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if !m.results[0] {
			t.Error("input with extra spaces should match")
		}
	})

	t.Run("g-dropping tolerance", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Just a small town girl, livin' in a lonely world"})
		m.state = stateTyping
		m.input = "just a small town girl living in a lonely world"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if !m.results[0] {
			t.Error("g-dropping input should match (livin vs living)")
		}
	})

	t.Run("skips comments after enter", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"First line", "# Comment", "Third line"})
		m.state = stateTyping
		m.input = "First line"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if m.currentLine != 2 {
			t.Errorf("currentLine = %d, want 2 (should skip comment)", m.currentLine)
		}
	})

	t.Run("transitions to result after last line", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Only line"})
		m.state = stateTyping
		m.input = "Only line"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if m.state != stateResult {
			t.Errorf("state = %v, want stateResult", m.state)
		}
	})

	t.Run("backspace removes character", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Test"})
		m.state = stateTyping
		m.input = "Hello"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		m = newModel.(model)

		if m.input != "Hell" {
			t.Errorf("input = %q, want %q", m.input, "Hell")
		}
	})

	t.Run("typing adds characters", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Test"})
		m.state = stateTyping
		m.input = "He"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l', 'l'}})
		m = newModel.(model)

		if m.input != "Hell" {
			t.Errorf("input = %q, want %q", m.input, "Hell")
		}
	})

	t.Run("space adds space", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Test"})
		m.state = stateTyping
		m.input = "Hello"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
		m = newModel.(model)

		if m.input != "Hello " {
			t.Errorf("input = %q, want %q", m.input, "Hello ")
		}
	})

	t.Run("tab shows hint for next word", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Hello world today"})
		m.state = stateTyping
		m.input = "Hello "

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = newModel.(model)

		if m.hint != "world" {
			t.Errorf("hint = %q, want %q", m.hint, "world")
		}
		if m.hintLevel != 1 {
			t.Errorf("hintLevel = %d, want 1", m.hintLevel)
		}
	})

	t.Run("double tab shows full line", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Hello world today"})
		m.state = stateTyping
		m.input = "Hello "

		// First tab
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = newModel.(model)

		// Second tab
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = newModel.(model)

		if m.hint != "Hello world today" {
			t.Errorf("hint = %q, want %q", m.hint, "Hello world today")
		}
		if m.hintLevel != 2 {
			t.Errorf("hintLevel = %d, want 2", m.hintLevel)
		}
	})

	t.Run("third tab does nothing", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Hello world today"})
		m.state = stateTyping
		m.input = ""

		// First tab
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = newModel.(model)

		// Second tab
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = newModel.(model)

		// Third tab - should stay the same
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = newModel.(model)

		if m.hint != "Hello world today" {
			t.Errorf("hint = %q, want %q", m.hint, "Hello world today")
		}
		if m.hintLevel != 2 {
			t.Errorf("hintLevel = %d, want 2", m.hintLevel)
		}
	})

	t.Run("typing clears hint", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Hello world"})
		m.state = stateTyping
		m.hint = "Hello"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
		m = newModel.(model)

		if m.hint != "" {
			t.Errorf("hint = %q, want empty (should be cleared on typing)", m.hint)
		}
	})

	t.Run("backspace clears hint", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Hello world"})
		m.state = stateTyping
		m.input = "Hello"
		m.hint = "Hello"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		m = newModel.(model)

		if m.hint != "" {
			t.Errorf("hint = %q, want empty (should be cleared on backspace)", m.hint)
		}
	})

	t.Run("enter clears hint", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Hello", "World"})
		m.state = stateTyping
		m.input = "Hello"
		m.hint = "Hello"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if m.hint != "" {
			t.Errorf("hint = %q, want empty (should be cleared on enter)", m.hint)
		}
	})
}

func TestHandleResultInput(t *testing.T) {
	t.Run("y restarts", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Line one", "Line two"})
		m.state = stateResult
		m.currentLine = 2
		m.results[0] = true
		m.results[1] = false

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		m = newModel.(model)

		if m.state != stateTyping {
			t.Errorf("state = %v, want stateTyping", m.state)
		}
		if m.currentLine != 0 {
			t.Errorf("currentLine = %d, want 0", m.currentLine)
		}
		if m.results[0] || m.results[1] {
			t.Error("results should be reset")
		}
	})

	t.Run("Y restarts (uppercase)", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Line one"})
		m.state = stateResult
		m.currentLine = 1

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
		m = newModel.(model)

		if m.state != stateTyping {
			t.Errorf("state = %v, want stateTyping", m.state)
		}
	})

	t.Run("restart skips leading comments", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"# Comment", "Real line"})
		m.state = stateResult
		m.currentLine = 2

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		m = newModel.(model)

		if m.currentLine != 1 {
			t.Errorf("currentLine = %d, want 1 (should skip comment)", m.currentLine)
		}
	})

	t.Run("n quits", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Line one"})
		m.state = stateResult

		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

		if cmd == nil {
			t.Error("expected quit command")
		}
	})

	t.Run("N quits (uppercase)", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Line one"})
		m.state = stateResult

		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})

		if cmd == nil {
			t.Error("expected quit command")
		}
	})
}

func TestView(t *testing.T) {
	t.Run("section select shows sections", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"# Verse 1", "Line one", "# Chorus", "Line two"})
		m.state = stateSectionSelect
		view := m.View()

		if !strings.Contains(view, "Select Section:") {
			t.Error("view should show section selection header")
		}
		if !strings.Contains(view, "a. All sections") {
			t.Error("view should show all sections option")
		}
		if !strings.Contains(view, "1. Verse 1") {
			t.Error("view should show Verse 1 section")
		}
		if !strings.Contains(view, "2. Chorus") {
			t.Error("view should show Chorus section")
		}
	})

	t.Run("section select shows title and artist when set", func(t *testing.T) {
		meta := metadata{Title: "Amazing Grace", Artist: "John Newton"}
		m := initialModel(meta, []string{"Line one"})
		view := m.View()

		if !strings.Contains(view, "Amazing Grace") {
			t.Error("section select should show title")
		}
		if !strings.Contains(view, "John Newton") {
			t.Error("section select should show artist")
		}
	})

	t.Run("typing state shows cursor", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Test line"})
		m.state = stateTyping
		view := m.View()

		if !strings.Contains(view, "_") {
			t.Error("view should contain cursor")
		}
	})

	t.Run("result state shows score excluding comments", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"# Comment", "Line one", "Line two"})
		m.state = stateResult
		m.currentLine = 3
		m.results[0] = true // comment
		m.results[1] = true // correct
		m.results[2] = false // incorrect

		view := m.View()

		// Score should be 1/2, not 2/3 (comment excluded)
		if !strings.Contains(view, "Score: 1/2") {
			t.Errorf("view should show 'Score: 1/2', got: %s", view)
		}
	})

	t.Run("result state shows try again prompt", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Line one"})
		m.state = stateResult
		m.currentLine = 1

		view := m.View()

		if !strings.Contains(view, "Try again? (y/n)") {
			t.Error("view should contain try again prompt")
		}
	})

	t.Run("shows checkmark for correct lines", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Line one", "Line two"})
		m.state = stateTyping
		m.currentLine = 1
		m.results[0] = true

		view := m.View()

		if !strings.Contains(view, "✓") {
			t.Error("view should contain checkmark for correct line")
		}
	})

	t.Run("shows X for incorrect lines", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Line one", "Line two"})
		m.state = stateTyping
		m.currentLine = 1
		m.results[0] = false

		view := m.View()

		if !strings.Contains(view, "✗") {
			t.Error("view should contain X for incorrect line")
		}
	})

	t.Run("typing state shows header text without hash prefix", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"# Verse 1", "Line one"})
		m.state = stateTyping
		m.currentLine = 1
		m.results[0] = true

		view := m.View()

		if !strings.Contains(view, "Verse 1") {
			t.Error("view should show header text")
		}
		if strings.Contains(view, "# Verse 1") {
			t.Error("view should not show hash prefix in header")
		}
	})

	t.Run("result state shows header text without hash prefix", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"# Chorus", "Line one"})
		m.state = stateResult
		m.currentLine = 2
		m.results[0] = true
		m.results[1] = true

		view := m.View()

		if !strings.Contains(view, "Chorus") {
			t.Error("view should show header text")
		}
		if strings.Contains(view, "# Chorus") {
			t.Error("view should not show hash prefix in header")
		}
	})

	t.Run("typing state shows hint when set", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Hello world"})
		m.state = stateTyping
		m.hint = "world"

		view := m.View()

		if !strings.Contains(view, "Hint: world") {
			t.Error("view should show hint")
		}
	})

	t.Run("typing state hides hint when empty", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Hello world"})
		m.state = stateTyping
		m.hint = ""

		view := m.View()

		if strings.Contains(view, "Hint:") {
			t.Error("view should not show hint when empty")
		}
	})
}

func TestQuitCommands(t *testing.T) {
	t.Run("ctrl+c quits in section select state", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Test"})
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

		if cmd == nil {
			t.Error("expected quit command")
		}
	})

	t.Run("escape quits in section select state", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Test"})
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

		if cmd == nil {
			t.Error("expected quit command")
		}
	})

	t.Run("ctrl+c quits in typing state", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Test"})
		m.state = stateTyping
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

		if cmd == nil {
			t.Error("expected quit command")
		}
	})

	t.Run("escape quits in typing state", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Test"})
		m.state = stateTyping
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

		if cmd == nil {
			t.Error("expected quit command")
		}
	})

	t.Run("ctrl+c quits in result state", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Test"})
		m.state = stateResult
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

		if cmd == nil {
			t.Error("expected quit command")
		}
	})

	t.Run("escape quits in result state", func(t *testing.T) {
		m := initialModel(metadata{}, []string{"Test"})
		m.state = stateResult
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

		if cmd == nil {
			t.Error("expected quit command")
		}
	})
}

func TestReadFile(t *testing.T) {
	t.Run("parses YAML front matter", func(t *testing.T) {
		// Create temp file with front matter
		content := `---
title: Amazing Grace
artist: John Newton
---
How sweet the sound
That saved a wretch like me
`
		f, err := os.CreateTemp("", "lyrics-*.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())
		if _, err := f.WriteString(content); err != nil {
			t.Fatal(err)
		}
		f.Close()

		meta, lines, err := readFile(f.Name())
		if err != nil {
			t.Fatalf("readFile error: %v", err)
		}

		if meta.Title != "Amazing Grace" {
			t.Errorf("Title = %q, want %q", meta.Title, "Amazing Grace")
		}
		if meta.Artist != "John Newton" {
			t.Errorf("Artist = %q, want %q", meta.Artist, "John Newton")
		}
		if len(lines) != 2 {
			t.Errorf("len(lines) = %d, want 2", len(lines))
		}
		if lines[0] != "How sweet the sound" {
			t.Errorf("lines[0] = %q, want %q", lines[0], "How sweet the sound")
		}
	})

	t.Run("handles file without front matter", func(t *testing.T) {
		content := `Line one
Line two
`
		f, err := os.CreateTemp("", "lyrics-*.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())
		if _, err := f.WriteString(content); err != nil {
			t.Fatal(err)
		}
		f.Close()

		meta, lines, err := readFile(f.Name())
		if err != nil {
			t.Fatalf("readFile error: %v", err)
		}

		if meta.Title != "" || meta.Artist != "" {
			t.Error("metadata should be empty for file without front matter")
		}
		if len(lines) != 2 {
			t.Errorf("len(lines) = %d, want 2", len(lines))
		}
	})

	t.Run("skips empty lines", func(t *testing.T) {
		content := `---
title: Test
---
Line one

Line two
`
		f, err := os.CreateTemp("", "lyrics-*.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())
		if _, err := f.WriteString(content); err != nil {
			t.Fatal(err)
		}
		f.Close()

		_, lines, err := readFile(f.Name())
		if err != nil {
			t.Fatalf("readFile error: %v", err)
		}

		if len(lines) != 2 {
			t.Errorf("len(lines) = %d, want 2 (should skip empty lines)", len(lines))
		}
	})
}
