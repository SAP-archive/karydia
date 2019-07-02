// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"flag"
	"fmt"
	"os"

	"testing"

	"github.com/karydia/karydia/tests/e2e/framework"
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

	defer func() {
		if err := f.DeleteAll(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete karydia and all test resources: %v\n", err)
			fmt.Fprintf(os.Stderr, "You have to cleanup yourself, sorry\n")
		}
	}()

	return m.Run()
}
