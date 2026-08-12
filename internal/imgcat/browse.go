package imgcat

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/term"
)

// NavAction is a decoded navigation keypress.
type NavAction int

const (
	// NavNone represents an unrecognized or ignored key.
	NavNone NavAction = iota
	// NavNext advances to the next image (space, right, down, enter).
	NavNext
	// NavPrev goes back to the previous image (left, up).
	NavPrev
	// NavQuit exits the browser (q, Ctrl-C).
	NavQuit
)

// Browse displays each image in paths one at a time with keyboard
// navigation: space/right/down advance, left/up go back, q quits.
// It switches the terminal into raw mode and reads keys from /dev/tty
// so that piping an image through stdin still leaves the keyboard
// available.
func Browse(w io.Writer, paths []string, opts Options) error {
	if len(paths) == 0 {
		return fmt.Errorf("no images to browse")
	}

	tty, err := openTTY()
	if err != nil {
		return err
	}
	defer tty.Close()

	oldState, err := term.MakeRaw(int(tty.Fd()))
	if err != nil {
		return fmt.Errorf("set raw terminal mode: %w", err)
	}
	defer term.Restore(int(tty.Fd()), oldState)

	return browseLoop(w, tty, paths, opts)
}

// openTTY returns a handle to the controlling terminal for keyboard input.
// /dev/tty is preferred so that piping an image through stdin still leaves
// the keyboard available. Falls back to stdin when it is a terminal
// (e.g. Windows, where /dev/tty does not exist).
func openTTY() (*os.File, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err == nil {
		return tty, nil
	}
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return os.Stdin, nil
	}
	return nil, fmt.Errorf("open /dev/tty for keyboard input: %w", err)
}

// browseLoop renders paths one at a time, reading navigation keys from r.
// The reader parameter is accepted as an io.Reader (not *os.File) so tests
// can drive it with a scripted reader.
func browseLoop(w io.Writer, r io.Reader, paths []string, opts Options) error {
	idx := 0
	for {
		clearScreen(w)

		data, err := os.ReadFile(paths[idx])
		if err != nil {
			return fmt.Errorf("read %s: %w", paths[idx], err)
		}
		if err := Display(w, data, opts); err != nil {
			return fmt.Errorf("display %s: %w", paths[idx], err)
		}
		statusLine(w, paths[idx], idx, len(paths))

		action, err := readKey(r)
		if err != nil {
			return err
		}
		switch action {
		case NavNext:
			idx = (idx + 1) % len(paths)
		case NavPrev:
			idx = (idx - 1 + len(paths)) % len(paths)
		case NavQuit:
			return nil
		}
	}
}

// readKey reads one keypress from a raw-mode terminal. It maps single
// bytes (q, space, Ctrl-C) and ANSI arrow-key sequences to NavActions.
// Unrecognized input is ignored (NavNone).
func readKey(r io.Reader) (NavAction, error) {
	var b [1]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return NavNone, err
	}

	switch b[0] {
	case 'q', 'Q', '\x03': // \x03 is Ctrl-C in raw mode (ISIG is off)
		return NavQuit, nil
	case ' ', '\r', '\n':
		return NavNext, nil
	case '\x1b':
		// Arrow keys arrive as ESC [ A/B/C/D.
		var rest [2]byte
		if _, err := io.ReadFull(r, rest[:]); err != nil {
			return NavNone, nil // bare ESC or truncated — ignore
		}
		if rest[0] == '[' {
			switch rest[1] {
			case 'C', 'B': // right, down
				return NavNext, nil
			case 'A', 'D': // left, up
				return NavPrev, nil
			}
		}
		return NavNone, nil
	default:
		return NavNone, nil
	}
}

// clearScreen clears the terminal and homes the cursor (ANSI).
func clearScreen(w io.Writer) {
	fmt.Fprint(w, "\x1b[2J\x1b[H")
}

// statusLine prints the position indicator and navigation hints below the
// current image.
func statusLine(w io.Writer, path string, index, total int) {
	fmt.Fprintf(w, "\nImage %d/%d — %s  (← → arrows, q to quit)\n",
		index+1, total, filepath.Base(path))
}
