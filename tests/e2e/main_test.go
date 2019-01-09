package e2e

import (
	"flag"
	"fmt"
	"os"
	"time"

	"testing"

	"github.com/kinvolk/karydia/tests/e2e/framework"
)

var f *framework.Framework

func TestMain(m *testing.M) {
	os.Exit(main(m))
}

func main(m *testing.M) int {
	var err error

	kubeconfig := flag.String("kubeconfig", "", "Path to the kubeconfig file")
	server := flag.String("server", "", "The address and port of the Kubernetes API server")
	namespace := flag.String("namespace", "", "Namespace to deploy karydia into")

	flag.Parse()

	f, err = framework.Setup(*server, *kubeconfig, *namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup framework: %v\n", err)
		return 1
	}

	if err := f.CreateNamespace(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create namespace: %v", err)
		return 1
	}

	// We configure the webhook first to make sure
	// the certificate, key and configuration are
	// in place once the pod starts.
	if err := f.ConfigureWebhook(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register karydia webhook: %v", err)
		return 1
	}
	defer func() {
		if err := f.DeleteWebhook(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to remove webhook configuration: %v", err)
		}
	}()

	if err := f.SetupKarydia(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup karydia: %v\n", err)
		return 1
	}
	defer func() {
		if err := f.DeleteAll(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete karydia and all test resources: %v\n", err)
			fmt.Fprintf(os.Stderr, "You have to cleanup yourself, sorry\n")
		}
	}()

	timeout := 2 * time.Minute
	if err := f.WaitRunning(timeout); err != nil {
		fmt.Fprintf(os.Stderr, "karydia hasn't fully started within 2 minutes\n")
		return 1
	}

	return m.Run()
}
