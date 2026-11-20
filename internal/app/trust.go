package app

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Claude Code asks whether to trust a folder it never ran in (a new task's
// worktree) before anything else, hooks included. The chat view shows that
// question so it can be answered without the terminal: derby reads it from
// the session's output and moves the dialog's cursor to "Yes".

const trustScanMax = 256 << 10 // the dialog comes first: a longer output is past it

// screenRows replays terminal output on a grid and returns its rows: enough
// of a terminal (cursor moves, erases) to read what Ink last drew.
func screenRows(b []byte) []string {
	grid := map[int]map[int]rune{}
	row, col, savedRow, savedCol, last := 0, 0, 0, 0, 0
	clamp := func() {
		row, col = max(row, 0), max(col, 0)
	}
	eraseLine := func(r, from, to int) {
		for c := range grid[r] {
			if c >= from && (to < 0 || c <= to) {
				delete(grid[r], c)
			}
		}
	}
	for i := 0; i < len(b); {
		c := b[i]
		switch {
		case c == 0x1b && i+1 < len(b):
			i = screenEscape(b, i, func(cmd byte, params []int, private bool) {
				n := 1
				if len(params) > 0 && params[0] > 0 {
					n = params[0]
				}
				p0 := 0
				if len(params) > 0 {
					p0 = params[0]
				}
				switch cmd {
				case 'A':
					row -= n
				case 'B', 'e':
					row += n
				case 'C', 'a':
					col += n
				case 'D':
					col -= n
				case 'E':
					row, col = row+n, 0
				case 'F':
					row, col = row-n, 0
				case 'G', '`':
					col = n - 1
				case 'd':
					row = n - 1
				case 'H', 'f':
					row, col = n-1, 0
					if len(params) > 1 && params[1] > 0 {
						col = params[1] - 1
					}
				case 'K':
					switch p0 {
					case 0:
						eraseLine(row, col, -1)
					case 1:
						eraseLine(row, 0, col)
					case 2:
						delete(grid, row)
					}
				case 'J':
					switch p0 {
					case 0:
						eraseLine(row, col, -1)
						for r := range grid {
							if r > row {
								delete(grid, r)
							}
						}
					case 1:
						eraseLine(row, 0, col)
						for r := range grid {
							if r < row {
								delete(grid, r)
							}
						}
					default:
						clear(grid)
					}
				case 's', '7':
					savedRow, savedCol = row, col
				case 'u', '8':
					row, col = savedRow, savedCol
				}
				clamp()
			})
			continue
		case c == '\r':
			col = 0
		case c == '\n':
			row++
		case c == '\b':
			col = max(col-1, 0)
		case c == '\t':
			col = (col/8 + 1) * 8
		case c < 0x20 || c == 0x7f:
		default:
			r, size := utf8.DecodeRune(b[i:])
			if grid[row] == nil {
				grid[row] = map[int]rune{}
			}
			grid[row][col] = r
			col++
			last = max(last, row)
			i += size
			continue
		}
		last = max(last, row)
		i++
	}
	rows := make([]string, last+1)
	for r, cells := range grid {
		if r > last || len(cells) == 0 {
			continue
		}
		width := 0
		for c := range cells {
			width = max(width, c+1)
		}
		line := make([]rune, width)
		for c := range line {
			line[c] = ' '
		}
		for c, ch := range cells {
			line[c] = ch
		}
		rows[r] = strings.TrimRight(string(line), " ")
	}
	return rows
}

// screenEscape reads the escape sequence at b[i] (an ESC), calls do for the
// cursor and erase ones (CSI, ESC 7 and ESC 8), and returns where the next
// byte starts.
func screenEscape(b []byte, i int, do func(cmd byte, params []int, private bool)) int {
	j := i + 1
	switch b[j] {
	case '[':
		j++
		start := j
		for j < len(b) && (b[j] < 0x40 || b[j] > 0x7e) {
			j++
		}
		if j >= len(b) {
			return j
		}
		raw := string(b[start:j])
		private := raw != "" && strings.ContainsAny(raw[:1], "?<>=")
		var params []int
		if !private {
			for _, p := range strings.Split(raw, ";") {
				n, _ := strconv.Atoi(p)
				params = append(params, n)
			}
		}
		if !private {
			do(b[j], params, private)
		}
		return j + 1
	case ']', 'P', '_', '^', 'X': // OSC, DCS, APC, PM, SOS: up to BEL or ST
		for j++; j < len(b); j++ {
			if b[j] == 0x07 && b[i+1] == ']' {
				return j + 1
			}
			if b[j] == 0x1b && j+1 < len(b) && b[j+1] == '\\' {
				return j + 2
			}
		}
		return j
	case '(', ')', '*', '+', '#', '%': // charset designations: one more byte
		return min(j+2, len(b))
	case '7', '8':
		do(b[j], nil, false)
		return j + 1
	default:
		return j + 1
	}
}

var optionNumber = regexp.MustCompile(`^\d+\.\s*`)

// readTrust finds Claude Code's folder trust dialog on the screen out draws:
// the folder it asks about, and how many Down presses move its cursor to
// "Yes". ok is false when the screen shows no such dialog.
func readTrust(out []byte) (folder string, downs int, ok bool) {
	rows := screenRows(out)
	yes, no := -1, -1
	for i, r := range rows {
		t := optionText(r)
		switch {
		case strings.HasPrefix(t, "No, exit"):
			no = i
		case strings.HasPrefix(t, "Yes") && (strings.Contains(t, "trust") || strings.Contains(t, "proceed")):
			yes = i
		}
	}
	if yes < 0 || no < 0 || yes-no > 3 || no-yes > 3 {
		return "", 0, false
	}
	lo, hi := min(yes, no), max(yes, no)
	var opts []int
	cur := 0
	for i := lo; i <= hi; i++ {
		if optionText(rows[i]) == "" {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(rows[i]), "❯") {
			cur = len(opts)
		}
		opts = append(opts, i)
	}
	yesAt := 0
	for k, i := range opts {
		if i == yes {
			yesAt = k
		}
	}
	downs = (yesAt - cur + len(opts)) % len(opts)
	for i := lo - 1; i >= 0 && i >= lo-12; i-- {
		t := strings.TrimSpace(rows[i])
		if strings.HasPrefix(t, "/") || strings.HasPrefix(t, "~") {
			folder = t
			break
		}
	}
	return folder, downs, true
}

// optionText is a dialog row without its cursor and numbering.
func optionText(row string) string {
	t := strings.TrimSpace(row)
	t = strings.TrimSpace(strings.TrimPrefix(t, "❯"))
	return optionNumber.ReplaceAllString(t, "")
}

// stepTrust is the folder a live step's Claude Code asks to trust, if it does.
func (a *App) stepTrust(stepID string) (string, int, bool) {
	sess, ok := a.run.Get(stepID)
	if !ok || sess.Exited() {
		return "", 0, false
	}
	out := sess.Output(trustScanMax)
	if out == nil {
		return "", 0, false
	}
	return readTrust(out)
}

// TrustFolder answers "Yes, I trust this folder" to the dialog a step's
// Claude Code shows.
func (a *App) TrustFolder(stepID string) error {
	folder, downs, ok := a.stepTrust(stepID)
	if !ok {
		return fmt.Errorf("Claude Code is not asking to trust a folder")
	}
	sess, _ := a.run.Get(stepID)
	for range downs {
		if _, err := sess.Write([]byte("\x1b[B")); err != nil {
			return err
		}
		time.Sleep(120 * time.Millisecond)
	}
	// Enter only once the cursor shows on "Yes": on "No, exit" it ends the session.
	for deadline := time.Now().Add(2 * time.Second); ; time.Sleep(100 * time.Millisecond) {
		_, d, ok := a.stepTrust(stepID)
		if ok && d == 0 {
			break
		}
		if !ok || time.Now().After(deadline) {
			return fmt.Errorf("could not select \"Yes\" for %s: answer in the terminal", folder)
		}
	}
	_, err := sess.Write([]byte("\r"))
	return err
}
