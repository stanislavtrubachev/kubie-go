package ui

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// ANSI color constants.
const (
	colorReset     = "\033[0m"
	colorRed       = "\033[31m"
	colorGreen     = "\033[32m"
	colorYellow    = "\033[33m"
	colorWhite     = "\033[37m"
	colorDarkGray  = "\033[90m"
	colorLightGray = "\033[97m"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

const (
	frameInterval  = 150 * time.Millisecond
	blinkEvery     = 2  // toggle brightness every N frames (~300ms)
	pollTimeoutMs  = 20 // ms; governs how quickly the input goroutine reacts to shutdown
)

// SpinnerStatus is the outcome of the work function passed to Run.
type SpinnerStatus int

const (
	StatusOK      SpinnerStatus = iota
	StatusWarning               // cluster reachable but nodes have warnings
	StatusError                 // cluster unreachable or critical error
)

// IsInteractive reports whether stdout is attached to a terminal.
func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// Run shows an animated Braille spinner at the prompt position while workFn executes:
//
//	[⠼ ctx|ns] $ <keystrokes typed during animation>
//
// stdin is put into raw mode so that keystrokes typed during the animation are
// captured and rendered as part of each frame (manual echo). Terminal echo is off
// during animation; cleanup() restores the original terminal state and is guaranteed
// to run before term.Restore so the parent shell's TTY is never corrupted.
//
// After at least one full animation cycle the final status is shown (▮ → ✓/!/✕ → ▮).
// In non-interactive mode workFn runs silently. SIGINT and Ctrl-C (raw byte 3) both
// cause a clean exit with an error symbol.
func Run(ctxName, ns string, workFn func() SpinnerStatus) {
	if !IsInteractive() {
		workFn()
		return
	}

	stdinFd := int(os.Stdin.Fd())
	rawState, rawErr := term.MakeRaw(stdinFd)

	// done closes to signal the input goroutine to stop polling.
	done := make(chan struct{})
	var goroutineWg sync.WaitGroup

	var (
		capMu    sync.Mutex
		captured []rune
	)
	ctrlCCh    := make(chan struct{}, 1)
	inputEvent := make(chan struct{}, 32) // buffered: immediate redraw on keystroke

	if rawErr == nil {
		goroutineWg.Add(1)
		go func() {
			defer goroutineWg.Done()
			buf := make([]byte, 1)
			fds := []unix.PollFd{{Fd: int32(stdinFd), Events: unix.POLLIN}}
			for {
				// Check shutdown signal first.
				select {
				case <-done:
					return
				default:
				}
				// Non-blocking poll so we revisit <-done every pollTimeoutMs.
				n, err := unix.Poll(fds, pollTimeoutMs)
				if err == unix.EINTR {
					continue // system call interrupted (e.g. SIGWINCH) — just retry
				}
				if err != nil || n <= 0 {
					continue // timeout or other transient error
				}
				n2, err := syscall.Read(stdinFd, buf)
				if err != nil || n2 == 0 {
					continue
				}
				switch buf[0] {
				case 3: // Ctrl-C: ISIG is disabled in raw mode
					select {
					case ctrlCCh <- struct{}{}:
					default:
					}
					return
				case 127, 8: // DEL / Backspace
					capMu.Lock()
					if len(captured) > 0 {
						captured = captured[:len(captured)-1]
					}
					capMu.Unlock()
					select {
					case inputEvent <- struct{}{}:
					default:
					}
				default:
					if buf[0] >= 32 && buf[0] < 127 { // ASCII printable only
						capMu.Lock()
						captured = append(captured, rune(buf[0]))
						capMu.Unlock()
						select {
						case inputEvent <- struct{}{}:
						default:
						}
					}
				}
			}
		}()
	}

	// cleanup stops the input goroutine, then restores the terminal.
	// sync.Once makes it safe to call from multiple paths (defer + explicit).
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			if rawErr == nil {
				close(done)
				goroutineWg.Wait() // at most ~pollTimeoutMs ms
				term.Restore(stdinFd, rawState) //nolint:errcheck
			}
		})
	}
	defer cleanup() // catches any return path not already covered

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	defer signal.Stop(sigCh)

	resultCh := make(chan SpinnerStatus, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultCh <- StatusError
			}
		}()
		resultCh <- workFn()
	}()

	getInput := func() string {
		capMu.Lock()
		defer capMu.Unlock()
		return string(captured)
	}

	minDuration := time.Duration(len(spinnerFrames)) * frameInterval
	start := time.Now()
	ticker := time.NewTicker(frameInterval)
	defer ticker.Stop()

	frame := 0
	var result SpinnerStatus
	resultReady := false

	inRawMode := rawErr == nil

	// Draw the initial frame.
	printSpinnerFrame(ctxName, ns, getInput(), frame, brightAt(frame), inRawMode)

	for {
		select {
		case <-ctrlCCh: // Ctrl-C in raw mode
			cleanup()
			eraseLine()
			printFinalLine(ctxName, ns, colorRed, "✕")
			fmt.Println()
			os.Exit(1)

		case <-sigCh: // SIGINT from kill(1) or when raw mode unavailable
			cleanup()
			eraseLine()
			printFinalLine(ctxName, ns, colorRed, "✕")
			fmt.Println()
			os.Exit(1)

		case r := <-resultCh:
			result = r
			resultReady = true

		case <-inputEvent:
			// Immediate redraw on keystroke so the user sees their typing without
			// waiting for the next ticker tick.
			printSpinnerFrame(ctxName, ns, getInput(), frame, brightAt(frame), inRawMode)

		case <-ticker.C:
			if resultReady && time.Since(start) >= minDuration {
				cleanup() // restore terminal BEFORE drawing final line
				eraseLine()
				showFinalAnimation(ctxName, ns, result)
				return
			}
			frame++
			printSpinnerFrame(ctxName, ns, getInput(), frame, brightAt(frame), inRawMode)
		}
	}
}

// brightAt returns true if the given frame index should use bright (light) gray.
func brightAt(frame int) bool {
	return (frame/blinkEvery)%2 == 0
}

// eraseLine clears the current terminal line.
func eraseLine() {
	fmt.Print("\r\033[K")
}

// printSpinnerFrame renders one animation frame:
//
//	[⠼ ctx|ns] $ userInput
//
// In raw mode (rawMode=true): \033[K clears the line before drawing; userInput is our
// manually-captured buffer (terminal echo is off). In cooked mode: only \r is used so
// the terminal's own echo of the user's typing is not erased.
func printSpinnerFrame(ctxName, ns, userInput string, frame int, bright, rawMode bool) {
	grayColor := colorDarkGray
	if bright {
		grayColor = colorLightGray
	}
	spinChar := spinnerFrames[frame%len(spinnerFrames)]
	content := ctxName
	if ns != "" {
		content += "|" + ns
	}
	if rawMode {
		fmt.Printf("\r\033[K%s[%s %s]%s $ %s", grayColor, spinChar, content, colorReset, userInput)
	} else {
		fmt.Printf("\r%s[%s %s]%s $ ", grayColor, spinChar, content, colorReset)
	}
}

// printFinalLine renders the steady final status line with proper colors:
// symColor for the status symbol, red for ctx, white for |, green for ns.
func printFinalLine(ctxName, ns, symColor, sym string) {
	if ns != "" {
		fmt.Printf("[%s%s%s %s%s%s%s|%s%s%s%s]",
			symColor, sym, colorReset,
			colorRed, ctxName, colorReset,
			colorWhite, colorReset,
			colorGreen, ns, colorReset)
	} else {
		fmt.Printf("[%s%s%s %s%s%s]",
			symColor, sym, colorReset,
			colorRed, ctxName, colorReset)
	}
}

// showFinalAnimation plays the three-phase finale: ▮ → check/warn/err char (300ms) → ▮.
// Must be called after cleanup() so the terminal is in cooked mode (fmt.Println works).
func showFinalAnimation(ctxName, ns string, status SpinnerStatus) {
	block := "▮"
	symColor := statusColor(status)

	var altSym string
	switch status {
	case StatusOK:
		altSym = "✓"
	case StatusWarning:
		altSym = "!"
	default:
		altSym = "✕"
	}

	// Phase 1: block symbol
	printFinalLine(ctxName, ns, symColor, block)
	time.Sleep(300 * time.Millisecond)

	// Phase 2: alternate symbol
	eraseLine()
	printFinalLine(ctxName, ns, symColor, altSym)
	time.Sleep(300 * time.Millisecond)

	// Phase 3: block symbol (permanent)
	eraseLine()
	printFinalLine(ctxName, ns, symColor, block)
	fmt.Println()
}

// statusColor returns the ANSI color for a given SpinnerStatus.
func statusColor(status SpinnerStatus) string {
	switch status {
	case StatusOK:
		return colorGreen
	case StatusWarning:
		return colorYellow
	default:
		return colorRed
	}
}
