package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	kubie "github.com/stanislavtrubacev/kubie-go/kubielib"
	"github.com/stanislavtrubacev/kubie-go/kubielib/health"
)

// HealthOptions holds parsed flags for the health command.
type HealthOptions struct {
	Watch     bool
	Interval  time.Duration
	Output    health.OutputFormat
	Namespace string
}

// Health runs the kubie health command.
func Health(settings *kubie.Settings, opts HealthOptions) error {
	if err := kubie.EnsureKubieActive(); err != nil {
		return err
	}

	k8s, mc, err := health.BuildClients()
	if err != nil {
		return fmt.Errorf("could not connect to cluster: %w", err)
	}

	run := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		h, err := health.Collect(ctx, k8s, mc, opts.Namespace)
		if err != nil {
			// print partial data even on collection errors
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}

		health.RunChecks(ctx, h, k8s, mc)

		return health.Render(os.Stdout, h, opts.Output)
	}

	if !opts.Watch {
		return run()
	}

	// watch mode: clear screen and refresh on interval
	for {
		fmt.Print("\033[H\033[2J") // clear screen
		if err := run(); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "\nRefreshing every %s — press Ctrl+C to exit\n", opts.Interval)
		time.Sleep(opts.Interval)
	}
}
