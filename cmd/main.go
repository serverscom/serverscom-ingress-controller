package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/serverscom/serverscom-ingress-controller/internal/config"
	"github.com/serverscom/serverscom-ingress-controller/internal/flags"
	"github.com/serverscom/serverscom-ingress-controller/internal/ingress/controller"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
)

var (
	version   string
	gitCommit string
)

func main() {
	klog.InitFlags(nil)

	conf, err := flags.ParseFlags()
	if err != nil {
		klog.Fatal(err)
	}
	if conf.ShowVersion {
		fmt.Printf("Version=%v GitCommit=%v\n", version, gitCommit)
		os.Exit(0)
	}

	kubeClient, err := config.NewCubeClient("")
	if err != nil {
		klog.Fatalf(err.Error())
	}

	if conf.Namespace != "" {
		_, err = kubeClient.CoreV1().Namespaces().Get(context.TODO(), conf.Namespace, metav1.GetOptions{})
		if err != nil {
			klog.Fatalf("No namespace with name %v found: %v", conf.Namespace, err)
		}
	}

	conf.KubeClient = kubeClient

	serverscomClient, err := config.NewServerscomClient()
	if err != nil {
		klog.Fatal(err.Error())
	}
	serverscomClient.SetupUserAgent(fmt.Sprintf("serverscom-ingress-controller/%s %s", version, gitCommit))

	ic := controller.NewIngressController(conf, serverscomClient)

	hostname, err := os.Hostname()
	if err != nil {
		klog.Fatalf("unable to get hostname: %v", err)
	}

	id := fmt.Sprintf("%s_%d", hostname, rand.Intn(1e6))

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      config.DefaultLockObjectName,
			Namespace: config.DefaultLockObjectNamespace,
		},
		Client: kubeClient.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id,
		},
	}

	leConfig := leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: conf.LeaderElectionCfg.LeaseDuration.Duration,
		RenewDeadline: conf.LeaderElectionCfg.RenewDeadline.Duration,
		RetryPeriod:   conf.LeaderElectionCfg.RetryPeriod.Duration,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				klog.Infof("%s starts leading", id)
				stopCh := make(chan struct{})
				defer close(stopCh)
				go handleSigterm(ic, 5)
				ic.Run(stopCh)
			},
			OnStoppedLeading: func() {
				klog.Fatal("lost master")
			},
		},
	}

	leaderelection.RunOrDie(context.Background(), leConfig)
}

func handleSigterm(ic *controller.IngressController, delay int) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)
	<-signalChan
	klog.InfoS("Received SIGTERM, shutting down")

	ic.Stop()

	klog.Infof("Handled quit, delaying controller exit for %d seconds", delay)
	time.Sleep(time.Duration(delay) * time.Second)

	exitCode := 0
	klog.Infof("Exiting with %v", exitCode)
	os.Exit(exitCode)
}
