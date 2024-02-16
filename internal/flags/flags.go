package flags

import (
	"flag"
	"os"

	"github.com/serverscom/serverscom-ingress-controller/internal/ingress/controller"

	"github.com/serverscom/serverscom-ingress-controller/internal/config"

	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	k8sopts "k8s.io/component-base/config/options"
)

// ParseFlags parses os args and map them to controller configuration
func ParseFlags() (*controller.Configuration, error) {
	var (
		flags = pflag.NewFlagSet("", pflag.ExitOnError)

		showVersion = flags.Bool("version", false,
			`Show controller version and exit.`)

		watchNamespace = flags.String("watch-namespace", v1.NamespaceAll,
			`Namespace to watch for Ingress/Services/Endpoints.`)

		ingressClass = flags.String("ingress-class", "",
			`If set, overrides what ingress classes are managed by the controller.`)

		resyncPeriod = flags.Duration("sync-period", 0,
			`Period at which the controller forces the repopulation of its local object stores. Disabled by default.`)
	)

	flags.AddGoFlagSet(flag.CommandLine)
	if err := flags.Parse(os.Args); err != nil {
		return nil, err
	}

	conf := &controller.Configuration{
		ShowVersion:       *showVersion,
		Namespace:         *watchNamespace,
		LeaderElectionCfg: config.DefaultLeaderElectionConfiguration(),
		ResyncPeriod:      *resyncPeriod,
		IngressClass:      *ingressClass,
	}

	k8sopts.BindLeaderElectionFlags(conf.LeaderElectionCfg, flags)

	return conf, nil
}
