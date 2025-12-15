package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

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

func TestInitialModel(t *testing.T) {
	t.Run("basic initialization", func(t *testing.T) {
		lines := []string{"Line one", "Line two"}
		m := initialModel(lines)

		if m.currentLine != 0 {
			t.Errorf("currentLine = %d, want 0", m.currentLine)
		}
		if m.state != stateTyping {
			t.Errorf("state = %v, want stateTyping", m.state)
		}
		if len(m.results) != 2 {
			t.Errorf("len(results) = %d, want 2", len(m.results))
		}
	})

	t.Run("skips leading comments", func(t *testing.T) {
		lines := []string{"# Comment", "# Another comment", "First real line"}
		m := initialModel(lines)

		if m.currentLine != 2 {
			t.Errorf("currentLine = %d, want 2", m.currentLine)
		}
		// Comments should be marked as correct
		if !m.results[0] || !m.results[1] {
			t.Error("comment lines should be marked as correct")
		}
	})

	t.Run("all comments transitions to result state", func(t *testing.T) {
		lines := []string{"# Only", "# Comments"}
		m := initialModel(lines)

		if m.state != stateResult {
			t.Errorf("state = %v, want stateResult", m.state)
		}
	})
}

func TestHandleTypingInput(t *testing.T) {
	t.Run("correct input", func(t *testing.T) {
		m := initialModel([]string{"Hello world", "Second line"})
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
		m.input = "hello world"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if !m.results[0] {
			t.Error("case insensitive match should be correct")
		}
	})

	t.Run("incorrect input", func(t *testing.T) {
		m := initialModel([]string{"Hello world"})
		m.input = "Wrong input"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if m.results[0] {
			t.Error("incorrect input should be marked wrong")
		}
	})

	t.Run("whitespace trimming", func(t *testing.T) {
		m := initialModel([]string{"Hello world"})
		m.input = "  Hello world  "

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if !m.results[0] {
			t.Error("whitespace-trimmed match should be correct")
		}
	})

	t.Run("skips comments after enter", func(t *testing.T) {
		m := initialModel([]string{"First line", "# Comment", "Third line"})
		m.input = "First line"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if m.currentLine != 2 {
			t.Errorf("currentLine = %d, want 2 (should skip comment)", m.currentLine)
		}
	})

	t.Run("transitions to result after last line", func(t *testing.T) {
		m := initialModel([]string{"Only line"})
		m.input = "Only line"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(model)

		if m.state != stateResult {
			t.Errorf("state = %v, want stateResult", m.state)
		}
	})

	t.Run("backspace removes character", func(t *testing.T) {
		m := initialModel([]string{"Test"})
		m.input = "Hello"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		m = newModel.(model)

		if m.input != "Hell" {
			t.Errorf("input = %q, want %q", m.input, "Hell")
		}
	})

	t.Run("typing adds characters", func(t *testing.T) {
		m := initialModel([]string{"Test"})
		m.input = "He"

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l', 'l'}})
		m = newModel.(model)

		if m.input != "Hell" {
			t.Errorf("input = %q, want %q", m.input, "Hell")
		}
	})

	t.Run("space adds space", func(t *testing.T) {
		m := initialModel([]string{"Test"})
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
	t.Run("typing state shows current line bold", func(t *testing.T) {
		m := initialModel([]string{"Test line"})
		view := m.View()

		if !strings.Contains(view, "Test line") {
			t.Error("view should contain current line")
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
		m.currentLine = 1
		m.results[0] = true

		view := m.View()

		if !strings.Contains(view, "✓") {
			t.Error("view should contain checkmark for correct line")
		}
	})

	t.Run("shows X for incorrect lines", func(t *testing.T) {
		m := initialModel([]string{"Line one", "Line two"})
		m.currentLine = 1
		m.results[0] = false

		view := m.View()

		if !strings.Contains(view, "✗") {
			t.Error("view should contain X for incorrect line")
		}
	})
}

func TestQuitCommands(t *testing.T) {
	t.Run("ctrl+c quits in typing state", func(t *testing.T) {
		m := initialModel([]string{"Test"})
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

		if cmd == nil {
			t.Error("expected quit command")
		}
	})

	t.Run("escape quits in typing state", func(t *testing.T) {
		m := initialModel([]string{"Test"})
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
