package cmd

import (
	"fmt"

	"github.com/stanislavtrubacev/kubie-go/kubielib"
)

// LintClusters checks clusters in established contexts for presence of:
// - clusters that are not referenced by any context
// - duplicate cluster names in the same file
func LintClusters(installed *kubie.Installed) {
	seen := make(map[string]struct{})

	for _, clusterSrc := range installed.Clusters {
		named := clusterSrc.Item

		contexts := installed.FindContextsByCluster(named.Name, clusterSrc.Source)
		if len(contexts) == 0 {
			fmt.Printf("Cluster '%s' has no context referencing it in file %s\n",
				named.Name, clusterSrc.Source)
		}

		key := named.Name + "|" + clusterSrc.Source
		if _, exists := seen[key]; exists {
			fmt.Printf("A cluster named '%s' appears more than once in file %s\n",
				named.Name, clusterSrc.Source)
		} else {
			seen[key] = struct{}{}
		}
	}
}

// LintUsers checks users in established contexts for presence of:
// - users who are not referenced by any context
// - duplicate usernames in the same file
func LintUsers(installed *kubie.Installed) {
	seen := make(map[string]struct{})

	for _, userSrc := range installed.Users {
		named := userSrc.Item

		contexts := installed.FindContextsByUser(named.Name, userSrc.Source)
		if len(contexts) == 0 {
			fmt.Printf("User '%s' has no context referencing it in file %s\n",
				named.Name, userSrc.Source)
		}

		key := named.Name + "|" + userSrc.Source
		if _, exists := seen[key]; exists {
			fmt.Printf("A user named '%s' appears more than once in file %s\n",
				named.Name, userSrc.Source)
		} else {
			seen[key] = struct{}{}
		}
	}
}

// LintContexts checks contexts in established contexts for presence of:
// - links to unknown clusters
// - links to unknown users
// - duplicate context names in the same file
func LintContexts(installed *kubie.Installed) {
	seen := make(map[string]struct{})

	for _, contextSrc := range installed.Contexts {
		named := contextSrc.Item

		cluster := installed.FindClusterByName(named.Context.Cluster, contextSrc.Source)
		if cluster == nil {
			fmt.Printf("Context '%s' references unknown cluster '%s' in file %s\n",
				named.Name, named.Context.Cluster, contextSrc.Source)
		}

		user := installed.FindUserByName(named.Context.User, contextSrc.Source)
		if user == nil {
			fmt.Printf("Context '%s' references unknown user '%s' in file %s\n",
				named.Name, named.Context.User, contextSrc.Source)
		}

		key := named.Name + "|" + contextSrc.Source
		if _, exists := seen[key]; exists {
			fmt.Printf("A context name '%s' appears more than once in file %s\n",
				named.Name, contextSrc.Source)
		} else {
			seen[key] = struct{}{}
		}
	}
}

// Lint checks loaded contexts for problems
func Lint(settings *kubie.Settings) error {
	installed, err := kubie.GetInstalledContexts(settings)
	if err != nil {
		return err
	}

	LintClusters(installed)
	LintUsers(installed)
	LintContexts(installed)

	return nil
}
