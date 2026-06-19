package kubie

import "strings"

// ResolveAlias returns the alias for contextName if one is configured, otherwise returns contextName unchanged.
func ResolveAlias(settings *Settings, contextName string) string {
	if settings == nil || len(settings.Aliases) == 0 {
		return contextName
	}
	if alias, ok := settings.Aliases[contextName]; ok && alias != "" {
		return alias
	}
	return contextName
}

// FormatContextForSelector formats a context name for display in an interactive list.
// If an alias exists, returns "original(alias)"; otherwise returns "original".
func FormatContextForSelector(settings *Settings, contextName string) string {
	if settings == nil || len(settings.Aliases) == 0 {
		return contextName
	}
	if alias, ok := settings.Aliases[contextName]; ok && alias != "" {
		return contextName + "(" + alias + ")"
	}
	return contextName
}

// ParseContextFromSelector extracts the original context name from a formatted selector string.
// Handles "original(alias)" is "original" and plain "original" is "original".
// Uses the last '(' to be safe if the original name itself contains parentheses.
func ParseContextFromSelector(formatted string) string {
	idx := strings.LastIndex(formatted, "(")
	if idx > 0 && strings.HasSuffix(formatted, ")") {
		return formatted[:idx]
	}
	return formatted
}
