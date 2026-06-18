package kubie

import (
	"fmt"
	"os"
	"strconv"
)

// GetDepth returns the current nesting depth of shells from the KUBIE_DEPTH environment variable
func GetDepth() uint32 {
	val, ok := os.LookupEnv("KUBIE_DEPTH")
	if !ok {
		return 0
	}
	depth, err := strconv.ParseUint(val, 10, 32)
	if err != nil {
		return 0
	}
	return uint32(depth)
}

// IsKubieActive checks whether the kubie environment is active using the KUBIE_ACTIVE environment variable
func IsKubieActive() bool {
	val, ok := os.LookupEnv("KUBIE_ACTIVE")
	if !ok {
		return false
	}
	return val == "1"
}

func EnsureKubieActive() error {
	if !IsKubieActive() {
		return fmt.Errorf("Not in a kubie-go shell!")
	}
	return nil
}

func GetSessionPath() string {
	val, ok := os.LookupEnv("KUBIE_SESSION")
	if !ok {
		return ""
	}
	return val
}
