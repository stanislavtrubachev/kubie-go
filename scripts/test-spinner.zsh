#!/usr/bin/env zsh
# Test script for kubie spinner daemon.
# Simulates Go writing a k8s result after --k8s-delay seconds.
# Usage:
#   ./scripts/test-spinner.zsh [--k8s-delay 0.5] [--status ok|warn|err]
#
# Output: one line per daemon frame so you can verify the spinner
# runs at least 30 frames (3 rotations) before stopping.

set -e

# --- defaults ---
K8S_DELAY=0.5   # how fast k8s "responds" (should be < 4.5s to test min-rotation logic)
K8S_STATUS=ok

while [[ $# -gt 0 ]]; do
    case "$1" in
        --k8s-delay) K8S_DELAY=$2; shift 2 ;;
        --status)    K8S_STATUS=$2; shift 2 ;;
        *) echo "unknown arg: $1"; exit 1 ;;
    esac
done

SPIN_FILE=$(mktemp /tmp/kubie-test-XXXXXX.txt)
PIPE_FILE="${SPIN_FILE}.pipe"
FRAMES=(⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏)

echo "=== kubie spinner test ==="
echo "k8s delay : ${K8S_DELAY}s  (min animation = 4.5s = 30 frames × 150ms)"
echo "k8s status: ${K8S_STATUS}"
echo "spin file : ${SPIN_FILE}"
echo ""

# Write initial frame (mirrors what context.go does).
printf '%s:0' "${FRAMES[1]}" > "$SPIN_FILE"

# Cleanup on exit.
cleanup() {
    rm -f "$SPIN_FILE" "$PIPE_FILE"
    [[ -n "$DAEMON_PID" ]] && kill "$DAEMON_PID" 2>/dev/null
    [[ -n "$K8S_PID"    ]] && kill "$K8S_PID"    2>/dev/null
}
trap cleanup EXIT INT

# Log file to capture daemon output.
LOG_FILE=$(mktemp /tmp/kubie-daemon-log-XXXXXX.txt)

# Run the daemon inline (mirrors __kubie_daemon__ from gen.go).
(
    idx=0 total=0 final_status=''
    while [[ -f "$SPIN_FILE" ]]; do
        s=$(<"$SPIN_FILE") 2>/dev/null
        case "$s" in
            ok|warn|err) [[ -z "$final_status" ]] && final_status="$s" ;;
        esac
        if [[ -n "$final_status" ]] && (( total >= 30 )); then
            printf '%s' "$final_status" > "$SPIN_FILE"
            ts=$(date +%T.%3N)
            echo "[$ts] frame $total  DONE → wrote final status '$final_status' to file" >> "$LOG_FILE"
            exit 0
        fi
        char="${FRAMES[$((idx+1))]}"
        bright=$(( (idx / 2) % 2 ))
        printf '%s:%d' "$char" "$bright" > "$SPIN_FILE"
        ts=$(date +%T.%3N)
        status_note=""
        [[ -n "$final_status" ]] && status_note="  (holding: k8s said '$final_status', waiting for frame 30)"
        echo "[$ts] frame $total  char=$char bright=$bright${status_note}" >> "$LOG_FILE"
        idx=$(( (idx + 1) % 10 ))
        total=$(( total + 1 ))
        sleep 0.15
    done
) &
DAEMON_PID=$!

# Simulate k8s response after K8S_DELAY seconds.
(
    sleep "$K8S_DELAY"
    ts=$(date +%T.%3N)
    echo "[$ts] *** Go writes '$K8S_STATUS' to spin file (k8s response) ***" >> "$LOG_FILE"
    printf '%s' "$K8S_STATUS" > "$SPIN_FILE"
) &
K8S_PID=$!

# Wait for daemon to finish and stream the log.
echo "Frames (150ms each):"
echo "--------------------"
tail -f "$LOG_FILE" &
TAIL_PID=$!

wait "$DAEMON_PID"
sleep 0.2
kill "$TAIL_PID" 2>/dev/null

echo ""
echo "--------------------"
echo "Total frames logged: $(grep -c 'frame' "$LOG_FILE")"
echo "Spin file contents : $(cat "$SPIN_FILE" 2>/dev/null || echo '<deleted>')"
rm -f "$LOG_FILE"
