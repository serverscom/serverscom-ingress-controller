package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

const (
	// namespace which owns the lock object.
	DefaultLockObjectNamespace = "kube-system"

	// object name of the lock object.
	DefaultLockObjectName = "ingress-serverscom-lock"

	// leader election default lease duration
	LeDefaultLeaseDuration = 15 * time.Second

	// leader election default renew deadline
	LeDefaultRenewDeadline = 10 * time.Second

	// leader election default retry period
	LeDefaultRetryPeriod = 2 * time.Second
)

func init() {
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			log.Fatalf("Error loading .env file: %v", err)
		}
	}
}
