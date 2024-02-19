package config

import (
	"errors"
	"os"
	"testing"

	. "github.com/onsi/gomega"
)

func TestNewServerscomClient(t *testing.T) {
	g := NewWithT(t)
	os.Setenv("SC_ACCESS_TOKEN", "123")
	_, err := NewServerscomClient()
	g.Expect(err).To(BeNil())
}

func TestFetchEnv(t *testing.T) {
	g := NewWithT(t)
	os.Setenv("SC_TEST", "123")

	e := FetchEnv("SC_TEST")
	g.Expect(e).To(Equal("123"))

	e = FetchEnv("NOT_EXISTS", "default")
	g.Expect(e).To(Equal("default"))

}

func TestNewCubeClient(t *testing.T) {
	g := NewWithT(t)
	_, err := NewCubeClient("config")
	g.Expect(err).To(MatchError(errors.New("failed to get kubernetes configuration")))
}
