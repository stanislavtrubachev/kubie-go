package kubie

import (
	"bufio"
	"fmt"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// HOME_DIR user home directory
var HOME_DIR string

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic("could not get home directory path")
	}
	HOME_DIR = home
}

func homeDir() string {
	return HOME_DIR
}

// expandUser replaces the prefix "~/" in the path with the home directory
func expandUser(path string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir(), path[2:])
	}
	return path
}

// Fzf (if use)
type Fzf struct {
	Mouse      bool    `yaml:"mouse"`
	Reverse    bool    `yaml:"reverse"`
	IgnoreCase bool    `yaml:"ignore_case"`
	InfoHidden bool    `yaml:"info_hidden"`
	Height     *string `yaml:"height"`
	Prompt     *string `yaml:"prompt"`
	Color      *string `yaml:"color"`
}

func (f *Fzf) UnmarshalYAML(value *yaml.Node) error {
	*f = DefaultFzf()
	type plain Fzf
	return value.Decode((*plain)(f))
}

func DefaultFzf() Fzf {
	return Fzf{
		Mouse: true,
	}
}

type Configs struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}

func defaultIncludePath() []string {
	home := homeDir()
	return []string{
		filepath.Join(home, ".kube", "config"),
		filepath.Join(home, ".kube", "*.yml"),
		filepath.Join(home, ".kube", "*.yaml"),
		filepath.Join(home, ".kube", "configs", "*.yml"),
		filepath.Join(home, ".kube", "configs", "*.yaml"),
		filepath.Join(home, ".kube", "kubie", "*.yml"),
		filepath.Join(home, ".kube", "kubie", "*.yaml"),
	}
}

func DefaultConfigs() Configs {
	return Configs{
		Include: defaultIncludePath(),
		Exclude: []string{},
	}
}

type Prompt struct {
	Disable             bool `yaml:"disable"`
	ShowDepth           bool `yaml:"show_depth"`
	ZshUseRps1          bool `yaml:"zsh_use_rps1"`
	FishUseRprompt      bool `yaml:"fish_use_rprompt"`
	XonshUseRightPrompt bool `yaml:"xonsh_use_right_prompt"`
}

func (p *Prompt) UnmarshalYAML(value *yaml.Node) error {
	*p = DefaultPrompt()
	type plain Prompt
	return value.Decode((*plain)(p))
}

func DefaultPrompt() Prompt {
	return Prompt{
		ShowDepth: true,
	}
}

type ContextHeaderBehavior string

const (
	ContextHeaderBehaviorAuto   ContextHeaderBehavior = "auto"
	ContextHeaderBehaviorAlways ContextHeaderBehavior = "always"
	ContextHeaderBehaviorNever  ContextHeaderBehavior = "never"
)

func (b ContextHeaderBehavior) ShouldPrintHeaders() bool {
	switch b {
	case ContextHeaderBehaviorAuto:
		return term.IsTerminal(int(os.Stdout.Fd()))
	case ContextHeaderBehaviorAlways:
		return true
	case ContextHeaderBehaviorNever:
		return false
	default:
		return false
	}
}

func DefaultContextHeaderBehavior() ContextHeaderBehavior {
	return ContextHeaderBehaviorAuto
}

type ValidateNamespacesBehavior string

const (
	ValidateNamespacesBehaviorTrue    ValidateNamespacesBehavior = "true"
	ValidateNamespacesBehaviorFalse   ValidateNamespacesBehavior = "false"
	ValidateNamespacesBehaviorPartial ValidateNamespacesBehavior = "partial"
)

func (v ValidateNamespacesBehavior) CanListNamespaces() bool {
	switch v {
	case ValidateNamespacesBehaviorTrue, ValidateNamespacesBehaviorPartial:
		return true
	case ValidateNamespacesBehaviorFalse:
		return false
	default:
		return false
	}
}

func DefaultValidateNamespacesBehavior() ValidateNamespacesBehavior {
	return ValidateNamespacesBehaviorTrue
}

type Behavior struct {
	ValidateNamespaces           ValidateNamespacesBehavior `yaml:"validate_namespaces"`
	PrintContextInExec           ContextHeaderBehavior      `yaml:"print_context_in_exec"`
	AllowMultipleContextPatterns bool                       `yaml:"allow_multiple_context_patterns"`
}

func (b *Behavior) UnmarshalYAML(value *yaml.Node) error {
	*b = DefaultBehavior()
	type plain Behavior
	return value.Decode((*plain)(b))
}

func DefaultBehavior() Behavior {
	return Behavior{
		ValidateNamespaces: DefaultValidateNamespacesBehavior(),
		PrintContextInExec: DefaultContextHeaderBehavior(),
	}
}

type Hooks struct {
	StartCtx string `yaml:"start_ctx"`
	StopCtx  string `yaml:"stop_ctx"`
}

func DefaultHooks() Hooks {
	return Hooks{}
}

type Settings struct {
	Shell         *string           `yaml:"shell"`
	DefaultEditor *string           `yaml:"default_editor"`
	Configs       Configs           `yaml:"configs"`
	Prompt        Prompt            `yaml:"prompt"`
	Behavior      Behavior          `yaml:"behavior"`
	Hooks         Hooks             `yaml:"hooks"`
	Fzf           Fzf               `yaml:"fzf"`
	Aliases       map[string]string `yaml:"aliases"`
}

func DefaultSettings() Settings {
	return Settings{
		Configs:  DefaultConfigs(),
		Prompt:   DefaultPrompt(),
		Behavior: DefaultBehavior(),
		Hooks:    DefaultHooks(),
		Fzf:      DefaultFzf(),
		Aliases:  map[string]string{},
	}
}

// SettingsPath get to path (~/.kube/kubie.yaml).
func SettingsPath() string {
	return filepath.Join(homeDir(), ".kube", "kubie.yaml")
}

// LoadSettings loads the settings from the file or returns the default settings.
// After downloading, adds the path to the file itself in `exclude`.
func LoadSettings() (Settings, error) {
	settingsPath := SettingsPath()
	settings := DefaultSettings()

	if info, err := os.Stat(settingsPath); err == nil && !info.IsDir() {
		file, err := os.Open(settingsPath)
		if err != nil {
			return Settings{}, fmt.Errorf("could not open settings file: %w", err)
		}
		defer file.Close()
		reader := bufio.NewReader(file)
		dec := yaml.NewDecoder(reader)
		if err := dec.Decode(&settings); err != nil {
			return Settings{}, fmt.Errorf("could not parse kubie config: %w", err)
		}
	}

	settings.Configs.Exclude = append(settings.Configs.Exclude, settingsPath)
	return settings, nil
}

// GetKubeConfigsPaths collects all kubeconfig paths based on `include/exclude`.
func (s *Settings) GetKubeConfigsPaths() ([]string, error) {
	pathSet := make(map[string]struct{})
	processPattern := func(pattern string, add bool) error {
		expanded := expandUser(pattern)
		matches, err := filepath.Glob(expanded)
		if err != nil {
			return err
		}
		for _, m := range matches {
			info, err := os.Stat(m)
			if err != nil || info.IsDir() {
				continue
			}
			if add {
				pathSet[m] = struct{}{}
			} else {
				delete(pathSet, m)
			}
		}
		return nil
	}

	for _, inc := range s.Configs.Include {
		if err := processPattern(inc, true); err != nil {
			return nil, fmt.Errorf("failed to process include pattern: %w", err)
		}
	}
	for _, exc := range s.Configs.Exclude {
		if err := processPattern(exc, false); err != nil {
			return nil, fmt.Errorf("failed to process exclude pattern: %w", err)
		}
	}

	result := make([]string, 0, len(pathSet))
	for p := range pathSet {
		result = append(result, p)
	}
	sort.Strings(result)
	return result, nil
}
