// Copyright 2020-2025 Politecnico di Torino
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main contains the entrypoint for webSSH, a WebSocket SSH bridge for CrownLabs.
package main

import (
	"flag"
	"log"
	"strconv"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog/v2"

	crownlabsv1alpha1 "github.com/netgroup-polito/CrownLabs/operators/api/v1alpha1"
	crownlabsv1alpha2 "github.com/netgroup-polito/CrownLabs/operators/api/v1alpha2"
	"github.com/netgroup-polito/CrownLabs/operators/pkg/webssh"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = crownlabsv1alpha1.AddToScheme(scheme)
	_ = crownlabsv1alpha2.AddToScheme(scheme)
}

func main() {

	sshUserFlag := flag.String("websshuser", "crownlabs", "The user to use for SSH connections.")
	websshprivatekeypathFlag := flag.String("websshprivatekeypath", "", "The path to the private key file for SSH authentication.")
	websshtimeoutdurationFlag := flag.String("websshtimeoutduration", "30", "The timeout duration for SSH connections.")
	websshmaxconncountFlag := flag.String("websshmaxconncount", "1000", "The maximum number of concurrent SSH connections.")
	websshvmport := flag.String("websshvmport", "22", "The default SSH port for VMs.")
	websshwebsocketportFlag := flag.String("websshwebsocketport", "8085", "The port on which the WebSocket server listens.")

	timeout, err := strconv.Atoi(*websshtimeoutdurationFlag)
	if err != nil {
		timeout = 30
		log.Println("WEBSSH_TIMEOUT_DURATION is not a valid integer, using default value: ", timeout)
	}

	maxConn, err := strconv.Atoi(*websshmaxconncountFlag)
	if err != nil {
		maxConn = 1000
		log.Println("WEBSSH_MAX_CONN_COUNT is not a valid integer, using default value: ", maxConn)
	}

	klog.Info("Starting WebSocket SSH bridge")
	webssh.StartWebSSH(&webssh.Config{
		SSHUser:            *sshUserFlag,
		PrivateKeyPath:     *websshprivatekeypathFlag,
		TimeoutDuration:    timeout,
		MaxConnectionCount: maxConn,
		WebsocketPort:      *websshwebsocketportFlag,
		VMSSHPort:          *websshvmport,
	})
}
