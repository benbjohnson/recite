package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModeSelect(t *testing.T) {
	t.Run("initial model starts in mode select state", func(t *testing.T) {
		m := initialModel([]string{"Line one"})
		if m.state != stateModeSelect {
			t.Errorf("state = %v, want stateModeSelect", m.state)
		}
	})

	t.Run("pressing 1 selects practice mode", func(t *testing.T) {
		m := initialModel([]string{"Line one"})

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
		m = newModel.(model)

		if m.mode != modePractice {
			t.Errorf("mode = %v, want modePractice", m.mode)
		}
		if m.state != stateTyping {
			t.Errorf("state = %v, want stateTyping", m.state)
		}
	})

	t.Run("pressing 2 selects memory mode", func(t *testing.T) {
		m := initialModel([]string{"Line one"})

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
		m = newModel.(model)

		if m.mode != modeMemory {
			t.Errorf("mode = %v, want modeMemory", m.mode)
		}
		if m.state != stateTyping {
			t.Errorf("state = %v, want stateTyping", m.state)
		}
	})

	t.Run("mode select skips leading comments", func(t *testing.T) {
		m := initialModel([]string{"# Comment", "Real line"})

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
		m = newModel.(model)

		if m.currentLine != 1 {
			t.Errorf("currentLine = %d, want 1 (should skip comment)", m.currentLine)
		}
	})

	t.Run("ctrl+c quits from mode select", func(t *testing.T) {
		m := initialModel([]string{"Line one"})
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

		if cmd == nil {
			t.Error("expected quit command")
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
		m := initialModel(lines)

		if m.currentLine != 0 {
			t.Errorf("currentLine = %d, want 0", m.currentLine)
		}
		if m.state != stateModeSelect {
			t.Errorf("state = %v, want stateModeSelect", m.state)
		}
		if len(m.results) != 2 {
			t.Errorf("len(results) = %d, want 2", len(m.results))
		}
	})
}

func TestHandleTypingInput(t *testing.T) {
	t.Run("correct input", func(t *testing.T) {
		m := initialModel([]string{"Hello world", "Second line"})
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
		m := initialModel([]string{"Hello World"})
		m.state = stateTyping
		m.input = "hello world"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if !m.results[0] {
			t.Error("case insensitive match should be correct")
		}
	})

	t.Run("incorrect input", func(t *testing.T) {
		m := initialModel([]string{"Hello world"})
		m.state = stateTyping
		m.input = "Wrong input"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if m.results[0] {
			t.Error("incorrect input should be marked wrong")
		}
	})

	t.Run("whitespace trimming", func(t *testing.T) {
		m := initialModel([]string{"Hello world"})
		m.state = stateTyping
		m.input = "  Hello world  "

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if !m.results[0] {
			t.Error("whitespace-trimmed match should be correct")
		}
	})

	t.Run("ignores punctuation", func(t *testing.T) {
		m := initialModel([]string{"Don't stop believin'"})
		m.state = stateTyping
		m.input = "dont stop believin"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if !m.results[0] {
			t.Error("punctuation-free input should match")
		}
	})

	t.Run("ignores extra spaces", func(t *testing.T) {
		m := initialModel([]string{"Hello world"})
		m.state = stateTyping
		m.input = "Hello    world"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if !m.results[0] {
			t.Error("input with extra spaces should match")
		}
	})

	t.Run("skips comments after enter", func(t *testing.T) {
		m := initialModel([]string{"First line", "# Comment", "Third line"})
		m.state = stateTyping
		m.input = "First line"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if m.currentLine != 2 {
			t.Errorf("currentLine = %d, want 2 (should skip comment)", m.currentLine)
		}
	})

	t.Run("transitions to result after last line", func(t *testing.T) {
		m := initialModel([]string{"Only line"})
		m.state = stateTyping
		m.input = "Only line"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if m.state != stateResult {
			t.Errorf("state = %v, want stateResult", m.state)
		}
	})

	t.Run("backspace removes character", func(t *testing.T) {
		m := initialModel([]string{"Test"})
		m.state = stateTyping
		m.input = "Hello"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		m = newModel.(model)

		if m.input != "Hell" {
			t.Errorf("input = %q, want %q", m.input, "Hell")
		}
	})

	t.Run("typing adds characters", func(t *testing.T) {
		m := initialModel([]string{"Test"})
		m.state = stateTyping
		m.input = "He"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l', 'l'}})
		m = newModel.(model)

		if m.input != "Hell" {
			t.Errorf("input = %q, want %q", m.input, "Hell")
		}
	})

	t.Run("space adds space", func(t *testing.T) {
		m := initialModel([]string{"Test"})
		m.state = stateTyping
		m.input = "Hello"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
		m = newModel.(model)

		if m.input != "Hello " {
			t.Errorf("input = %q, want %q", m.input, "Hello ")
		}
	})
}

func TestHandleResultInput(t *testing.T) {
	t.Run("y restarts", func(t *testing.T) {
		m := initialModel([]string{"Line one", "Line two"})
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
		m := initialModel([]string{"Line one"})
		m.state = stateResult
		m.currentLine = 1

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
		m = newModel.(model)

		if m.state != stateTyping {
			t.Errorf("state = %v, want stateTyping", m.state)
		}
	})

	t.Run("restart skips leading comments", func(t *testing.T) {
		m := initialModel([]string{"# Comment", "Real line"})
		m.state = stateResult
		m.currentLine = 2

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		m = newModel.(model)

		if m.currentLine != 1 {
			t.Errorf("currentLine = %d, want 1 (should skip comment)", m.currentLine)
		}
	})

	t.Run("restart preserves mode", func(t *testing.T) {
		m := initialModel([]string{"Line one"})
		m.mode = modeMemory
		m.state = stateResult
		m.currentLine = 1

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		m = newModel.(model)

		if m.mode != modeMemory {
			t.Errorf("mode = %v, want modeMemory (should preserve mode on restart)", m.mode)
		}
	})

	t.Run("n quits", func(t *testing.T) {
		m := initialModel([]string{"Line one"})
		m.state = stateResult

		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

		if cmd == nil {
			t.Error("expected quit command")
		}
	})

	t.Run("N quits (uppercase)", func(t *testing.T) {
		m := initialModel([]string{"Line one"})
		m.state = stateResult

		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})

		if cmd == nil {
			t.Error("expected quit command")
		}
	})
}

func TestView(t *testing.T) {
	t.Run("mode select shows menu", func(t *testing.T) {
		m := initialModel([]string{"Test line"})
		view := m.View()

		if !strings.Contains(view, "Select Mode:") {
			t.Error("view should show mode selection header")
		}
		if !strings.Contains(view, "1. Practice") {
			t.Error("view should show practice option")
		}
		if !strings.Contains(view, "2. Memory") {
			t.Error("view should show memory option")
		}
	})

	t.Run("practice mode shows current line", func(t *testing.T) {
		m := initialModel([]string{"Test line"})
		m.state = stateTyping
		m.mode = modePractice
		view := m.View()

		if !strings.Contains(view, "Test line") {
			t.Error("practice mode should show current line")
		}
		if !strings.Contains(view, "_") {
			t.Error("view should contain cursor")
		}
	})

	t.Run("memory mode hides current line", func(t *testing.T) {
		m := initialModel([]string{"Test line"})
		m.state = stateTyping
		m.mode = modeMemory
		view := m.View()

		if strings.Contains(view, "Test line") {
			t.Error("memory mode should not show current line")
		}
		if !strings.Contains(view, "(type from memory)") {
			t.Error("memory mode should show memory prompt")
		}
		if !strings.Contains(view, "_") {
			t.Error("view should contain cursor")
		}
	})

	t.Run("result state shows score excluding comments", func(t *testing.T) {
		m := initialModel([]string{"# Comment", "Line one", "Line two"})
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
		m := initialModel([]string{"Line one"})
		m.state = stateResult
		m.currentLine = 1

		view := m.View()

		if !strings.Contains(view, "Try again? (y/n)") {
			t.Error("view should contain try again prompt")
		}
	})

	t.Run("shows checkmark for correct lines", func(t *testing.T) {
		m := initialModel([]string{"Line one", "Line two"})
		m.state = stateTyping
		m.currentLine = 1
		m.results[0] = true

		view := m.View()

		if !strings.Contains(view, "✓") {
			t.Error("view should contain checkmark for correct line")
		}
	})

	t.Run("shows X for incorrect lines", func(t *testing.T) {
		m := initialModel([]string{"Line one", "Line two"})
		m.state = stateTyping
		m.currentLine = 1
		m.results[0] = false

		view := m.View()

		if !strings.Contains(view, "✗") {
			t.Error("view should contain X for incorrect line")
		}
	})

	t.Run("typing state shows header text without hash prefix", func(t *testing.T) {
		m := initialModel([]string{"# Verse 1", "Line one"})
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
		m := initialModel([]string{"# Chorus", "Line one"})
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
}

func TestQuitCommands(t *testing.T) {
	t.Run("ctrl+c quits in mode select state", func(t *testing.T) {
		m := initialModel([]string{"Test"})
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

		if cmd == nil {
			t.Error("expected quit command")
		}
	})

	t.Run("escape quits in mode select state", func(t *testing.T) {
		m := initialModel([]string{"Test"})
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

		if cmd == nil {
			t.Error("expected quit command")
		}
	})

	t.Run("ctrl+c quits in typing state", func(t *testing.T) {
		m := initialModel([]string{"Test"})
		m.state = stateTyping
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

		if cmd == nil {
			t.Error("expected quit command")
		}
	})

	t.Run("escape quits in typing state", func(t *testing.T) {
		m := initialModel([]string{"Test"})
		m.state = stateTyping
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

		if cmd == nil {
			t.Error("expected quit command")
		}
	})

	t.Run("ctrl+c quits in result state", func(t *testing.T) {
		m := initialModel([]string{"Test"})
		m.state = stateResult
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

		if cmd == nil {
			t.Error("expected quit command")
		}
	})

	t.Run("escape quits in result state", func(t *testing.T) {
		m := initialModel([]string{"Test"})
		m.state = stateResult
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

		if cmd == nil {
			t.Error("expected quit command")
		}
	})
}
