package flags

import (
	"flag"
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func ResetForTesting(usage func()) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.Usage = usage
}

func TestNoMandatoryFlag(t *testing.T) {
	g := NewWithT(t)

	_, err := ParseFlags()
	g.Expect(err).To(BeNil())
}

func TestParseFlagsDefaults(t *testing.T) {
	g := NewWithT(t)

	ResetForTesting(func() { t.Fatal("Parsing failed") })

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{
		"cmd",
		"--watch-namespace", "default",
		"--ingress-class", "nginx",
		"--sync-period", "30s",
	}

	conf, err := ParseFlags()
	g.Expect(err).To(BeNil())
	g.Expect(conf).NotTo(BeNil())
	g.Expect(conf.Namespace).To(Equal("default"))
	g.Expect(conf.IngressClass).To(Equal("nginx"))
	g.Expect(conf.ResyncPeriod).To(Equal(30 * time.Second))
}
