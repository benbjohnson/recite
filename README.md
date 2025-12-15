# recite

A terminal-based program for memorizing song lyrics, poems, speeches, or any text you want to commit to memory.

Recite helps you practice by typing out lines either while seeing the text (practice mode) or purely from memory (memory mode). It tracks your accuracy and lets you retry until you've got it down.

## Install

### Homebrew (macOS/Linux)

```bash
brew install benbjohnson/tap/recite
```

### Go

```bash
go install github.com/benbjohnson/recite@latest
```

### Binary releases

Download pre-built binaries from the [releases page](https://github.com/benbjohnson/recite/releases).

## Usage

```bash
recite <lyrics-file>
```

When you start, you'll be prompted to select a mode:

1. **Practice** - The line is displayed and you type it back
2. **Memory** - Type each line from memory without seeing it

After typing each line and pressing Enter, you'll see whether you got it right (green checkmark) or wrong (red X). At the end, you'll see your score and can choose to try again.

### File format

Create a text file with one line per line of lyrics:

```
# Verse 1
Twinkle twinkle little star
How I wonder what you are

# Verse 2
Up above the world so high
Like a diamond in the sky
```

- Empty lines are skipped
- Lines starting with `#` are comments (displayed in gray, not typed by user)

### Controls

- **Enter** - Submit your answer and move to the next line
- **Backspace** - Delete the last character
- **Esc** or **Ctrl+C** - Quit
