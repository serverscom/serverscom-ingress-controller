package config

import (
	"fmt"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	k8sconfig "k8s.io/component-base/config"
)

// NewCubeClient creates a new k8s clientset
func NewCubeClient(configFile string) (*kubernetes.Clientset, error) {
	conf, err := getKubernetesConfig(configFile)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(conf)
}

// getKubernetesConfig prepares k8s config
func getKubernetesConfig(configFile string) (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		if configFile == "" {
			configFile = filepath.Join(homeDir(), ".kube", "config")
		}
		config, err = clientcmd.BuildConfigFromFlags("", configFile)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes configuration")
	}
	return config, nil
}

// DefaultLeaderElectionConfiguration returns default config for leader election
func DefaultLeaderElectionConfiguration() *k8sconfig.LeaderElectionConfiguration {
	return &k8sconfig.LeaderElectionConfiguration{
		LeaderElect:   true,
		LeaseDuration: metav1.Duration{Duration: LeDefaultLeaseDuration},
		RenewDeadline: metav1.Duration{Duration: LeDefaultRenewDeadline},
		RetryPeriod:   metav1.Duration{Duration: LeDefaultRetryPeriod},
		ResourceLock:  resourcelock.LeasesResourceLock,
	}
}

// homeDir returns home dir
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
