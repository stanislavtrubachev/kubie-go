package shell

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	kubie "github.com/stanislavtrubacev/kubie-go/kubielib"
	kubepath "github.com/stanislavtrubacev/kubie-go/path"
)

// MergeEnv takes the parent environment and applies overrides on top of it.
// With duplicate keys, the values from overrides win
func MergeEnv(overrides map[string]string) []string {
	env := make(map[string]string, len(os.Environ())+len(overrides))
	for _, e := range os.Environ() {
		if idx := strings.Index(e, "="); idx >= 0 {
			env[e[:idx]] = e[idx+1:]
		}
	}
	for k, v := range overrides {
		env[k] = v
	}
	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, k+"="+v)
	}
	return result
}

type EnvVars struct {
	Vars map[string]string
}

func NewEnvVars() EnvVars {
	return EnvVars{Vars: make(map[string]string)}
}

func (e *EnvVars) Insert(name, value string) {
	e.Vars[name] = value
}

func (e *EnvVars) Apply(cmd *exec.Cmd) {
	for k, v := range e.Vars {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
}

// SpawnShell launches a new shell with the specified settings, configuration and session.
// spinnerFile is the path to the animation state file; pass "" to disable animation.
func SpawnShell(settings *kubie.Settings, config kubie.KubeConfig, session *kubie.Session, spinnerFile string) error {

	var kind ShellKind
	if settings.Shell != nil {
		var ok bool
		kind, ok = ShellKindFromStr(*settings.Shell)
		if !ok {
			return fmt.Errorf("invalid shell setting: %s", *settings.Shell)
		}
	} else {
		var err error
		kind, err = Detect()
		if err != nil {
			return err
		}
	}

	configFile, err := os.CreateTemp("", "kubie-config-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp config file: %w", err)
	}
	defer os.Remove(configFile.Name())
	if err := configFile.Close(); err != nil {
		return err
	}
	if err := config.WriteToFile(configFile.Name()); err != nil {
		return err
	}

	sessionFile, err := os.CreateTemp("", "kubie-session-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp session file: %w", err)
	}
	defer os.Remove(sessionFile.Name())
	if err := sessionFile.Close(); err != nil {
		return err
	}
	if err := session.Save(sessionFile.Name()); err != nil {
		return err
	}

	depth := kubie.GetDepth()
	nextDepth := depth + 1

	envVars := NewEnvVars()
	envVars.Insert("KUBECONFIG", configFile.Name())
	envVars.Insert("KUBIE_ACTIVE", "1")
	envVars.Insert("KUBIE_DEPTH", strconv.Itoa(int(nextDepth)))
	envVars.Insert("KUBIE_KUBECONFIG", configFile.Name())
	envVars.Insert("KUBIE_SESSION", sessionFile.Name())
	envVars.Insert("KUBIE_STATE", kubepath.State())

	if settings.Prompt.Disable {
		envVars.Insert("KUBIE_PROMPT_DISABLE", "1")
	} else {
		envVars.Insert("KUBIE_PROMPT_DISABLE", "0")
	}
	if settings.Prompt.ZshUseRps1 {
		envVars.Insert("KUBIE_ZSH_USE_RPS1", "1")
	} else {
		envVars.Insert("KUBIE_ZSH_USE_RPS1", "0")
	}
	if settings.Prompt.FishUseRprompt {
		envVars.Insert("KUBIE_FISH_USE_RPROMPT", "1")
	} else {
		envVars.Insert("KUBIE_FISH_USE_RPROMPT", "0")
	}
	if settings.Prompt.XonshUseRightPrompt {
		envVars.Insert("KUBIE_XONSH_USE_RIGHT_PROMPT", "1")
	} else {
		envVars.Insert("KUBIE_XONSH_USE_RIGHT_PROMPT", "0")
	}

	switch kind {
	case ShellKindBash:
		envVars.Insert("KUBIE_SHELL", "bash")
	case ShellKindFish:
		envVars.Insert("KUBIE_SHELL", "fish")
	case ShellKindXonsh:
		envVars.Insert("KUBIE_SHELL", "xonsh")
	case ShellKindZsh:
		envVars.Insert("KUBIE_SHELL", "zsh")
	case ShellKindNu:
		envVars.Insert("KUBIE_SHELL", "nu")
	}

	prompt := GeneratePS1(settings, nextDepth, kind)

	// Extract ctx / ns for animation header.
	ctxName := ""
	ns := ""
	if len(config.Contexts) > 0 {
		ctxName = config.Contexts[0].Name
		if config.Contexts[0].Context.Namespace != nil {
			ns = *config.Contexts[0].Context.Namespace
		}
	}

	info := &ShellSpawnInfo{
		Settings:    settings,
		EnvVars:     envVars,
		Prompt:      prompt,
		SpinnerFile: spinnerFile,
		CtxName:     ctxName,
		NS:          ns,
	}

	switch kind {
	case ShellKindBash:
		return SpawnShellBash(info)
	case ShellKindFish:
		return SpawnShellFish(info)
	case ShellKindXonsh:
		return SpawnShellXonsh(info)
	case ShellKindZsh:
		return SpawnShellZsh(info)
	case ShellKindNu:
		return SpawnShellNu(info)
	default:
		return fmt.Errorf("unsupported shell kind: %s", kind)
	}
}
