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
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/metaclient"
)

func TestGetEdgeHubStatus(t *testing.T) {
	nodeName := "edge-node-1"
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: nodeName},
		Status: v1.NodeStatus{
			Conditions: []v1.NodeCondition{
				{
					Type:               v1.NodeReady,
					Status:             v1.ConditionTrue,
				},
			},
		},
	}

	fakeClient := fake.NewSimpleClientset(node)

	patches := gomonkey.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) {
		return fakeClient, nil
	})
	defer patches.Reset()

	// Intercept stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	opts := &EdgeHubStatusOptions{NodeName: nodeName}
	err := opts.getEdgeHubStatus()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Node: edge-node-1")
	assert.Contains(t, output, "Status: Ready (approx)")
	assert.Contains(t, output, "Connection: Unknown")
}
