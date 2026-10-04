// block-env-hook is a Claude Code PreToolUse hook that blocks any access to
// .env* files (except .env.example). Exit code 2 blocks the call and feeds
// stderr back to Claude.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/ecastro/dev-tools/internal/envguard"
)

func main() {
	payload, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(0)
	}
	targets, err := envguard.Targets(payload)
	if err != nil {
		os.Exit(0) // not a payload we understand: never block
	}
	if envguard.Blocked(targets) {
		fmt.Fprintln(os.Stderr, "Blocked: access to .env* files is prohibited.")
		os.Exit(2)
	}
}
