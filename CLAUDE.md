# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Recite is a terminal-based program for memorizing song lyrics, written in Go using the Bubble Tea TUI framework.

## Build Commands

```bash
go install ./...        # Build and install binary
go test ./...           # Run all tests
go test -v ./...        # Run tests with verbose output
go test -run TestName   # Run a specific test
```

## CI

GitHub Actions runs on push/PR to main:
- `test`: Builds and runs all tests
- `lint`: Runs golangci-lint

## Usage

```bash
./recite <lyrics-file>
```

The lyrics file should contain one line per line of lyrics. Empty lines are skipped. Lines starting with `#` are comments (displayed in gray, not typed by user).

## Architecture

Single-file Bubble Tea application with a state machine:
- `stateTyping`: User types each line, Enter submits and advances
- `stateResult`: Shows final score, y/n to restart or quit

The model tracks current line index, user input, and per-line results (correct/incorrect).
