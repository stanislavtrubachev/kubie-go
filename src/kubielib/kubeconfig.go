package kubie

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	yaml "gopkg.in/yaml.v3"
)

// KubeConfig represents the contents of the kubeconfig file.
// To support additional fields that may be present in the file, custom serialization is used: all unknown keys
// are stored in the Others field and embedded in the root object when writing.
type KubeConfig struct {
	Clusters       []NamedCluster         `yaml:"clusters"`
	Users          []NamedUser            `yaml:"users"`
	Contexts       []NamedContext         `yaml:"contexts"`
	CurrentContext *string                `yaml:"current-context,omitempty"`
	Others         map[string]interface{} `yaml:",inline"`
}

type Mapping map[string]interface{}

type NamedCluster struct {
	Name    string  `yaml:"name"`
	Cluster Mapping `yaml:"cluster"`
}

type NamedUser struct {
	Name string  `yaml:"name"`
	User Mapping `yaml:"user"`
}

type NamedContext struct {
	Name    string  `yaml:"name"`
	Context Context `yaml:"context"`
}

type Context struct {
	Cluster   string  `yaml:"cluster"`
	Namespace *string `yaml:"namespace,omitempty"`
	User      string  `yaml:"user"`
}

type Sourced[T any] struct {
	Source string // todo: maybe *string ?
	Item   T
}

func NewSourced[T any](source string, item T) Sourced[T] {
	return Sourced[T]{Source: source, Item: item}
}

// String (for Debug)
func (s Sourced[T]) String() string {
	return fmt.Sprintf("Sourced{Source: %s, Item: %v}", s.Source, s.Item)
}

type Installed struct {
	Clusters []Sourced[NamedCluster]
	Users    []Sourced[NamedUser]
	Contexts []Sourced[NamedContext]
}

// WriteToFile writes the configuration to a file at the specified path with access rights 0600
// (only the owner can read/write)
func (k *KubeConfig) WriteToFile(path string) error {

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("could not write file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file) // BufWriter

	encoder := yaml.NewEncoder(writer)
	defer encoder.Close()

	if err := encoder.Encode(k); err != nil {
		return err
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	return nil
}

// FindContextByName searches for a context by name in the Contexts list
func (inst *Installed) FindContextByName(name string) *Sourced[NamedContext] {
	for i := range inst.Contexts {
		if inst.Contexts[i].Item.Name == name {
			return &inst.Contexts[i]
		}
	}
	return nil
}

// FindClusterByName searches for a cluster by name and source path,
// First it searches for an exact match by name and source, then only by name.
func (inst *Installed) FindClusterByName(name string, source string) *Sourced[NamedCluster] {
	for i := range inst.Clusters {
		if inst.Clusters[i].Item.Name == name && inst.Clusters[i].Source == source {
			return &inst.Clusters[i]
		}
	}

	for i := range inst.Clusters {
		if inst.Clusters[i].Item.Name == name {
			return &inst.Clusters[i]
		}
	}
	return nil
}

// FindUserByName searches for a user by name and source path,
// First it searches for an exact match by name and source, then only by name.
func (inst *Installed) FindUserByName(name string, source string) *Sourced[NamedUser] {
	// Первый проход: точное совпадение имени и источника
	for i := range inst.Users {
		if inst.Users[i].Item.Name == name && inst.Users[i].Source == source {
			return &inst.Users[i]
		}
	}
	// Второй проход: совпадение только по имени
	for i := range inst.Users {
		if inst.Users[i].Item.Name == name {
			return &inst.Users[i]
		}
	}
	return nil
}

// FindContextsByCluster searches for all contexts belonging to the specified cluster from the specified source
func (inst *Installed) FindContextsByCluster(name string, source string) []*Sourced[NamedContext] {
	var result []*Sourced[NamedContext]
	for i := range inst.Contexts {
		ctx := &inst.Contexts[i]

		if ctx.Item.Context.Cluster == name && ctx.Source == source {
			result = append(result, ctx)
		}
	}
	return result
}

// FindContextsByUser searches for all contexts belonging to the specified user from the specified source.
func (inst *Installed) FindContextsByUser(name string, source string) []*Sourced[NamedContext] {
	var result []*Sourced[NamedContext]
	for i := range inst.Contexts {
		ctx := &inst.Contexts[i]

		if ctx.Item.Context.User == name && ctx.Source == source {
			result = append(result, ctx)
		}
	}
	return result
}

// GetContextsMatching Accepts a template string (glob, for example dev-*) and the 'allowMultipleContextPatterns` flag.
// If the flag is enabled, it breaks the template into spaces and processes each part separately. This allows you to
// write `kubie exec 'dev-* staging-1' ...` to select multiple groups of contexts at once. For each template,
// it iterates through all loaded contexts via the `filepath'.Match`, collects matches and warns if the pattern does
// not find anything, then combines the results, sorts them by name and eliminates repetitions. It is used in exec and
// export to define a list of contexts to perform an operation on.
func (inst *Installed) GetContextsMatching(pattern string, allowMultipleContextPatterns bool) []*Sourced[NamedContext] {
	var result []*Sourced[NamedContext]

	var patterns []string
	if allowMultipleContextPatterns {
		patterns = strings.Fields(pattern)
	} else {
		patterns = []string{pattern}
	}

	for _, p := range patterns {
		var matches []*Sourced[NamedContext]
		for i := range inst.Contexts {
			ctx := &inst.Contexts[i]
			matched, err := filepath.Match(p, ctx.Item.Name)
			if err == nil && matched {
				matches = append(matches, ctx)
			}
		}

		if len(patterns) > 1 && len(matches) == 0 {
			fmt.Printf("WARNING: No context matching %s\n", p)
		}
		result = append(result, matches...)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Item.Name < result[j].Item.Name
	})

	if len(result) == 0 {
		return result
	}
	unique := []*Sourced[NamedContext]{result[0]}
	for _, v := range result[1:] {
		if v.Item.Name != unique[len(unique)-1].Item.Name {
			unique = append(unique, v)
		}
	}
	return unique
}

// DeleteContext deletes the context by name from all involved kubeconfig files.
// If the file becomes empty after deletion, it is deleted completely.
func (inst *Installed) DeleteContext(name string) error {
	// 1. Ищем контекст по имени
	ctx := inst.FindContextByName(name)
	if ctx == nil {
		return fmt.Errorf("context not found")
	}

	filePath := ctx.Source

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("could not read kubeconfig file %s: %w", filePath, err)
	}

	var kubeconfig KubeConfig
	if err := yaml.Unmarshal(data, &kubeconfig); err != nil {
		return fmt.Errorf("could not parse YAML: %w", err)
	}

	newContexts := make([]NamedContext, 0, len(kubeconfig.Contexts))
	for _, c := range kubeconfig.Contexts {
		if c.Name != ctx.Item.Name {
			newContexts = append(newContexts, c)
		}
	}
	kubeconfig.Contexts = newContexts

	newClusters := make([]NamedCluster, 0, len(kubeconfig.Clusters))
	for _, cl := range kubeconfig.Clusters {
		if cl.Name != ctx.Item.Context.Cluster {
			newClusters = append(newClusters, cl)
		}
	}
	kubeconfig.Clusters = newClusters

	newUsers := make([]NamedUser, 0, len(kubeconfig.Users))
	for _, u := range kubeconfig.Users {
		if u.Name != ctx.Item.Context.User {
			newUsers = append(newUsers, u)
		}
	}
	kubeconfig.Users = newUsers

	if len(kubeconfig.Contexts) == 0 && len(kubeconfig.Clusters) == 0 && len(kubeconfig.Users) == 0 {
		fmt.Printf("Deleting kubeconfig %s because it is now empty.\n", filePath)
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("could not delete empty kubeconfig file: %w", err)
		}
	} else {
		fmt.Printf("Updating kubeconfig %s.\n", filePath)
		newData, err := yaml.Marshal(&kubeconfig)
		if err != nil {
			return fmt.Errorf("could not marshal YAML: %w", err)
		}
		if err := os.WriteFile(filePath, newData, 0600); err != nil {
			return fmt.Errorf("could not write kubeconfig file: %w", err)
		}
	}

	return nil
}

// makePathAbsolute checks and combines the key value in mapping
func (inst *Installed) makePathAbsolute(mapping map[string]interface{}, key string, parent string) {
	if _, ok := mapping[key]; !ok {
		return
	}
	str, ok := mapping[key].(string)
	if !ok {
		panic("value should be a string")
	}
	if !filepath.IsAbs(str) {
		newPath := filepath.Join(parent, str)
		if !utf8.ValidString(newPath) {
			panic("path should be a valid unicode string")
		}
		mapping[key] = newPath
	}
}

// MakeKubeconfigForContext creates a new KubeConfig containing only the specified context,
// (cluster, user, with absolute paths to the certificate files)
func (inst *Installed) MakeKubeconfigForContext(contextName string, namespaceName *string) (*KubeConfig, error) {
	// 1. Найти контекст
	var ctx Sourced[NamedContext]
	found := false
	for _, c := range inst.Contexts {
		if c.Item.Name == contextName {
			ctx = c
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("Could not find context %s", contextName)
	}

	if namespaceName != nil {
		ctx.Item.Context.Namespace = namespaceName
	}

	kubeconfigDir := filepath.Dir(ctx.Source)

	clusterSrcPtr := inst.FindClusterByName(ctx.Item.Context.Cluster, ctx.Source)
	if clusterSrcPtr == nil {
		return nil, fmt.Errorf("Could not find cluster %s referenced by context %s",
			ctx.Item.Context.Cluster, contextName)
	}
	clusterSrc := *clusterSrcPtr
	namedCluster := clusterSrc.Item

	inst.makePathAbsolute(namedCluster.Cluster, "certificate-authority", kubeconfigDir)

	userSrcPtr := inst.FindUserByName(ctx.Item.Context.User, ctx.Source)
	if userSrcPtr == nil {
		return nil, fmt.Errorf("Could not find user %s referenced by context %s",
			ctx.Item.Context.User, contextName)
	}
	userSrc := *userSrcPtr
	namedUser := userSrc.Item

	inst.makePathAbsolute(namedUser.User, "client-certificate", kubeconfigDir)
	inst.makePathAbsolute(namedUser.User, "client-key", kubeconfigDir)
	
	current := contextName
	kc := KubeConfig{
		Clusters:       []NamedCluster{namedCluster},
		Contexts:       []NamedContext{ctx.Item},
		Users:          []NamedUser{namedUser},
		CurrentContext: &current,
		Others: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Config",
		},
	}

	return &kc, nil
}

// LoadKubeconfigs loads all kubeconfig files from the passed path list, parses them and collects them into a single Installed struct.
// Each element (cluster, context, user) is saved along with the path to the file from which it was downloaded.
// Parsing errors are output to stderr, but they do not interrupt the execution.
func LoadKubeconfigs(paths []string) (*Installed, error) {
	installed := &Installed{
		Clusters: []Sourced[NamedCluster]{},
		Contexts: []Sourced[NamedContext]{},
		Users:    []Sourced[NamedUser]{},
	}

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil || info.IsDir() {
			continue
		}

		data, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading kubeconfig %s: %v\n", p, err)
			continue
		}

		var kubeconfig KubeConfig
		if err := yaml.Unmarshal(data, &kubeconfig); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading kubeconfig %s: %v\n", p, err)
			continue
		}

		for _, cluster := range kubeconfig.Clusters {
			installed.Clusters = append(installed.Clusters, NewSourced(p, cluster))
		}
		for _, ctx := range kubeconfig.Contexts {
			installed.Contexts = append(installed.Contexts, NewSourced(p, ctx))
		}
		for _, user := range kubeconfig.Users {
			installed.Users = append(installed.Users, NewSourced(p, user))
		}
	}

	return installed, nil
}

// GetInstalledContexts loads all contexts from kubeconfig files
func GetInstalledContexts(settings *Settings) (*Installed, error) {
	paths, err := settings.GetKubeConfigsPaths()
	if err != nil {
		return nil, err
	}

	installed, err := LoadKubeconfigs(paths)
	if err != nil {
		return nil, err
	}

	if len(installed.Contexts) == 0 {
		return nil, fmt.Errorf("Could not find any contexts in the Kubie kubeconfig directories!")
	}

	return installed, nil
}

// GetKubeconfigsContexts loads all kubeconfig files from the passed path list
func GetKubeconfigsContexts(kubeconfigs []string) (*Installed, error) {
	installed, err := LoadKubeconfigs(kubeconfigs)
	if err != nil {
		return nil, err
	}
	if len(installed.Contexts) == 0 {
		return nil, fmt.Errorf("Could not find any contexts in the given set of files!")
	}
	return installed, nil
}

// GetKubeconfigPath returns the path to the kubeconfig file from the KUBIE_KUBECONFIG environment variable
func GetKubeconfigPath() (string, error) {
	path, ok := os.LookupEnv("KUBIE_KUBECONFIG")
	if !ok {
		return "", fmt.Errorf("KUBIE_CONFIG not found")
	}
	return path, nil
}

// GetCurrentConfig loads the current kubeconfig from the file specified in the KUBIE_KUBECONFIG environment variable
func GetCurrentConfig() (*KubeConfig, error) {
	path, err := GetKubeconfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig file %s: %w", path, err)
	}

	var cfg KubeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML from %s: %w", path, err)
	}

	return &cfg, nil
}
