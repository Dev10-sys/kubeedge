/*
Copyright 2025 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package get

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/kubeedge/api/apis/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/metaclient"
)

var edgeHubStatusShortDescription = `Get the health status of EdgeHub component`

type EdgeHubStatusOptions struct {
	NodeName string
}

func NewEdgeHubStatusGet() *cobra.Command {
	opts := &EdgeHubStatusOptions{}
	cmd := &cobra.Command{
		Use:   "edgehub-status",
		Short: edgeHubStatusShortDescription,
		Long:  edgeHubStatusShortDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdutil.CheckErr(opts.getEdgeHubStatus())
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.NodeName, "node", "", "Specify the node name to check EdgeHub status")

	return cmd
}

func (o *EdgeHubStatusOptions) getEdgeHubStatus() error {
	nodeName := o.NodeName
	if nodeName == "" {
		config, err := util.ParseEdgecoreConfig(constants.EdgecoreConfigPath)
		if err != nil {
			return fmt.Errorf("failed to get edge core config: %v", err)
		}
		nodeName = config.Modules.Edged.HostnameOverride
	}

	clientset, err := metaclient.KubeClient()
	if err != nil {
		return fmt.Errorf("failed to create kube client: %v", err)
	}

	node, err := clientset.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node %s: %v", nodeName, err)
	}

	status := "Not Ready (approx)"
	connection := "Unknown"

	// Using NodeReady as approximation since EdgeHub status is not directly exposed
	for _, condition := range node.Status.Conditions {
		if condition.Type == v1.NodeReady {
			if condition.Status == v1.ConditionTrue {
				status = "Ready (approx)"
			}
			break
		}
	}

	fmt.Fprintf(os.Stdout, "Node: %s\n", nodeName)
	fmt.Fprintf(os.Stdout, "Status: %s\n", status)
	fmt.Fprintf(os.Stdout, "Connection: %s\n", connection)

	return nil
}
