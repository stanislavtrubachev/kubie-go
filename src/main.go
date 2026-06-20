package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	cmd "github.com/stanislavtrubacev/kubie-go/cmd"
	kubie "github.com/stanislavtrubacev/kubie-go/kubielib"
	"github.com/stanislavtrubacev/kubie-go/kubielib/health"
	"github.com/stanislavtrubacev/kubie-go/shell"
)

func main() {
	settings, err := kubie.LoadSettings()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading settings: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcommand := os.Args[1]
	args := os.Args[2:]

	switch subcommand {
	case "ctx":
		fs := flag.NewFlagSet("ctx", flag.ExitOnError)
		namespace := fs.String("n", "", "namespace to set")
		recursive := fs.Bool("r", false, "push context onto existing kubie shell")
		fs.Parse(args)
		var contextName *string
		if fs.NArg() > 0 {
			s := fs.Arg(0)
			contextName = &s
		}
		var ns *string
		if *namespace != "" {
			ns = namespace
		}
		err = cmd.Context(&settings, contextName, ns, nil, *recursive)

	case "ns":
		fs := flag.NewFlagSet("ns", flag.ExitOnError)
		recursive := fs.Bool("r", false, "push namespace onto existing kubie shell")
		unset := fs.Bool("unset", false, "unset namespace back to default")
		fs.Parse(args)
		var ns *string
		if fs.NArg() > 0 {
			s := fs.Arg(0)
			ns = &s
		}
		err = cmd.Namespace(&settings, ns, *recursive, *unset)

	case "info":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "info: required argument: ctx|ns|depth")
			os.Exit(1)
		}
		var kind cmd.KubieInfoKind
		switch args[0] {
		case "ctx", "context":
			kind = cmd.KubieInfoKindContext
		case "ns", "namespace":
			kind = cmd.KubieInfoKindNamespace
		case "depth":
			kind = cmd.KubieInfoKindDepth
		default:
			fmt.Fprintf(os.Stderr, "info: unknown kind: %s\n", args[0])
			os.Exit(1)
		}
		err = cmd.Info(cmd.KubieInfo{Kind: kind, Settings: &settings})

	case "exec":
		fs := flag.NewFlagSet("exec", flag.ExitOnError)
		exitEarly := fs.Bool("exit-early", false, "exit if kubectl returns non-zero")
		contextHeaders := fs.String("context-headers", "", "context-headers behaviour: always|never|auto")
		fs.Parse(args)
		remaining := fs.Args()
		if len(remaining) < 2 {
			fmt.Fprintln(os.Stderr, "exec: required: CONTEXT NAMESPACE [-- ARGS...]")
			os.Exit(1)
		}
		contextName := remaining[0]
		namespaceName := remaining[1]
		execArgs := remaining[2:]
		if len(execArgs) > 0 && execArgs[0] == "--" {
			execArgs = execArgs[1:]
		}
		var chb *kubie.ContextHeaderBehavior
		if *contextHeaders != "" {
			v := kubie.ContextHeaderBehavior(*contextHeaders)
			chb = &v
		}
		err = cmd.Exec(&settings, contextName, namespaceName, *exitEarly, chb, execArgs)

	case "lint":
		err = cmd.Lint(&settings)

	case "edit":
		var contextName *string
		if len(args) > 0 {
			contextName = &args[0]
		}
		err = cmd.EditContext(&settings, contextName)

	case "edit-config":
		err = cmd.EditConfig(&settings)

	case "update":
		err = cmd.Update()

	case "delete":
		var contextName *string
		if len(args) > 0 {
			contextName = &args[0]
		}
		err = cmd.DeleteContext(&settings, contextName)

	case "export":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "export: required: CONTEXT NAMESPACE")
			os.Exit(1)
		}
		err = cmd.Export(&settings, args[0], args[1])

	case "health":
		fs := flag.NewFlagSet("health", flag.ExitOnError)
		watch := fs.Bool("watch", false, "continuously refresh output")
		watchShort := fs.Bool("w", false, "continuously refresh output (shorthand)")
		interval := fs.Duration("interval", 2*time.Second, "refresh interval for --watch")
		output := fs.String("output", "human", "output format: human|json|yaml")
		outputShort := fs.String("o", "", "output format shorthand")
		namespace := fs.String("namespace", "", "filter by namespace")
		fs.Parse(args)

		outFmt := health.OutputFormat(*output)
		if *outputShort != "" {
			outFmt = health.OutputFormat(*outputShort)
		}
		err = cmd.Health(&settings, cmd.HealthOptions{
			Watch:     *watch || *watchShort,
			Interval:  *interval,
			Output:    outFmt,
			Namespace: *namespace,
		})

	case "generate-completion":
		fs := flag.NewFlagSet("generate-completion", flag.ExitOnError)
		shellFlag := fs.String("shell", "", "shell kind: bash|zsh|fish|xonsh|nu")
		fs.Parse(args)
		gc := cmd.GenerateCompletionCommand{}
		if *shellFlag != "" {
			if kind, ok := shell.ShellKindFromStr(*shellFlag); ok {
				gc.Shell = &kind
			}
		}
		cmd.GenerateCompletion(gc)
		return

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", subcommand)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: kubie <command> [args]")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  ctx [NAME] [-n NAMESPACE] [-r]")
	fmt.Fprintln(os.Stderr, "  ns  [NAME] [-r] [--unset]")
	fmt.Fprintln(os.Stderr, "  info ctx|ns|depth")
	fmt.Fprintln(os.Stderr, "  exec CTX NS [--exit-early] [--context-headers ...] [-- ARGS...]")
	fmt.Fprintln(os.Stderr, "  lint")
	fmt.Fprintln(os.Stderr, "  edit [NAME]")
	fmt.Fprintln(os.Stderr, "  edit-config")
	fmt.Fprintln(os.Stderr, "  update")
	fmt.Fprintln(os.Stderr, "  delete [NAME]")
	fmt.Fprintln(os.Stderr, "  export CTX NS")
	fmt.Fprintln(os.Stderr, "  generate-completion [--shell SHELL]")
	fmt.Fprintln(os.Stderr, "  health [-w] [-o human|json|yaml] [--namespace NS] [--interval 2s]")
}
