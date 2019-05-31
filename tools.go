// +build tools

// Copyright (C) 2019 SAP SE or an SAP affiliate company. All rights reserved.
// This file is licensed under the Apache Software License, v. 2 except as
// noted otherwise in the LICENSE file.
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

package tools

import (
	_ "k8s.io/code-generator"
)

// This project uses go-based tools, e.g. to help with code generation. To
// ensure that everyone is using the same version these tools should be
// tracked in module's 'go.mod' file. Thus, this 'tools.go' file implements
// the recommended approach via import statements along with a '// +build
// tools' build constraint. The imports allow go commands to precisely
// record the tools' version information in 'go.mod' while the build
// constraint prevents builds from importing these tools.
//
// Sources:
// - https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
// - https://github.com/go-modules-by-example/index/blob/master/010_tools/README.md

