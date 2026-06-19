package cmd

import (
	"fmt"

	"github.com/stanislavtrubacev/kubie-go/kubielib"
)

// KubieInfoKind type of information requested
type KubieInfoKind string

const (
	KubieInfoKindContext   KubieInfoKind = "context"
	KubieInfoKindNamespace KubieInfoKind = "namespace"
	KubieInfoKindDepth     KubieInfoKind = "depth"
)

type KubieInfo struct {
	Kind     KubieInfoKind
	Settings *kubie.Settings
}

// Info outputs information depending on the type
func Info(info KubieInfo) error {
	switch info.Kind {
	case KubieInfoKindContext:
		if err := kubie.EnsureKubieActive(); err != nil {
			return err
		}
		conf, err := kubie.GetCurrentConfig()
		if err != nil {
			return err
		}
		if conf.CurrentContext != nil {
			fmt.Println(kubie.ResolveAlias(info.Settings, *conf.CurrentContext))
		} else {
			fmt.Println("")
		}

	case KubieInfoKindNamespace:
		if err := kubie.EnsureKubieActive(); err != nil {
			return err
		}
		conf, err := kubie.GetCurrentConfig()
		if err != nil {
			return err
		}
		if len(conf.Contexts) > 0 && conf.Contexts[0].Context.Namespace != nil {
			fmt.Println(*conf.Contexts[0].Context.Namespace)
		} else {
			fmt.Println("default")
		}

	case KubieInfoKindDepth:
		if err := kubie.EnsureKubieActive(); err != nil {
			return err
		}
		fmt.Println(kubie.GetDepth())

	default:
		return fmt.Errorf("unknown info kind: %s", info.Kind)
	}
	return nil
}
