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
	"errors"
	"io"
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

func TestNewEdgeHubStatusGet(t *testing.T) {
	cmd := NewEdgeHubStatusGet()

	assert.NotNil(t, cmd)
	assert.Equal(t, "edgehub-status", cmd.Use)
	assert.Equal(t, edgeHubStatusShortDescription, cmd.Short)
	assert.Equal(t, edgeHubStatusShortDescription, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	outputFlag := cmd.Flags().Lookup("output")
	assert.NotNil(t, outputFlag)
	assert.Equal(t, "", outputFlag.DefValue)
}

func TestBuildEdgeHubStatus_WebSocket(t *testing.T) {
	config := v1alpha2.NewDefaultEdgeCoreConfig()
	config.Modules.EdgeHub.Enable = true
	config.Modules.EdgeHub.Heartbeat = 15
	config.Modules.EdgeHub.WebSocket = &v1alpha2.EdgeHubWebSocket{
		Enable: true,
		Server: "10.0.0.1:10000",
	}
	config.Modules.EdgeHub.Quic = &v1alpha2.EdgeHubQUIC{
		Enable: false,
		Server: "10.0.0.1:10001",
	}

	status := buildEdgeHubStatus(config)

	assert.True(t, status.IsEnabled)
	assert.Equal(t, "WebSocket", status.Protocol)
	assert.Equal(t, "10.0.0.1:10000", status.Server)
	assert.Equal(t, int32(15), status.Heartbeat)
}

func TestBuildEdgeHubStatus_QUIC(t *testing.T) {
	config := v1alpha2.NewDefaultEdgeCoreConfig()
	config.Modules.EdgeHub.Enable = true
	config.Modules.EdgeHub.Heartbeat = 30
	config.Modules.EdgeHub.WebSocket = &v1alpha2.EdgeHubWebSocket{
		Enable: false,
		Server: "10.0.0.1:10000",
	}
	config.Modules.EdgeHub.Quic = &v1alpha2.EdgeHubQUIC{
		Enable: true,
		Server: "10.0.0.1:10001",
	}

	status := buildEdgeHubStatus(config)

	assert.True(t, status.IsEnabled)
	assert.Equal(t, "QUIC", status.Protocol)
	assert.Equal(t, "10.0.0.1:10001", status.Server)
	assert.Equal(t, int32(30), status.Heartbeat)
}

func TestBuildEdgeHubStatus_Disabled(t *testing.T) {
	config := v1alpha2.NewDefaultEdgeCoreConfig()
	config.Modules.EdgeHub.Enable = false
	config.Modules.EdgeHub.WebSocket = &v1alpha2.EdgeHubWebSocket{Enable: false}
	config.Modules.EdgeHub.Quic = &v1alpha2.EdgeHubQUIC{Enable: false}

	status := buildEdgeHubStatus(config)

	assert.False(t, status.IsEnabled)
	assert.Equal(t, "Unknown", status.Protocol)
}

func TestBuildEdgeHubStatus_NoProtocolConfig(t *testing.T) {
	config := v1alpha2.NewDefaultEdgeCoreConfig()
	config.Modules.EdgeHub.Enable = true
	config.Modules.EdgeHub.WebSocket = nil
	config.Modules.EdgeHub.Quic = nil

	status := buildEdgeHubStatus(config)

	assert.True(t, status.IsEnabled)
	assert.Equal(t, "Unknown", status.Protocol)
	assert.Equal(t, "", status.Server)
}

func TestCheckCertificates_NoCerts(t *testing.T) {
	hub := &v1alpha2.EdgeHub{
		TLSCAFile:         "/nonexistent/ca.crt",
		TLSCertFile:       "/nonexistent/server.crt",
		TLSPrivateKeyFile: "/nonexistent/server.key",
	}

	cs := checkCertificates(hub)

	assert.False(t, cs.CAPresent)
	assert.False(t, cs.CertPresent)
	assert.False(t, cs.KeyPresent)
	assert.False(t, cs.CertValid)
	assert.Equal(t, "", cs.CertExpiry)
}

func TestCheckCertificates_EmptyPaths(t *testing.T) {
	hub := &v1alpha2.EdgeHub{
		TLSCAFile:         "",
		TLSCertFile:       "",
		TLSPrivateKeyFile: "",
	}

	cs := checkCertificates(hub)

	assert.False(t, cs.CAPresent)
	assert.False(t, cs.CertPresent)
	assert.False(t, cs.KeyPresent)
}

func TestFileExists(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		expect bool
	}{
		{
			name:   "empty path",
			path:   "",
			expect: false,
		},
		{
			name:   "nonexistent file",
			path:   "/does/not/exist/file.txt",
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fileExists(tt.path)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestPrintStatusJSON(t *testing.T) {
	status := &EdgeHubStatus{
		IsEnabled: true,
		Protocol:  "WebSocket",
		Server:    "10.0.0.1:10000",
		Heartbeat: 15,
		CertStatus: CertificateStatus{
			CAPresent:   true,
			CertPresent: true,
			KeyPresent:  true,
			CertExpiry:  "2026-12-31T00:00:00Z",
			CertValid:   true,
		},
	}

	oldStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	assert.NoError(t, pipeErr)
	os.Stdout = w

	err := printStatusJSON(status)

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, copyErr := io.Copy(&buf, r)
	assert.NoError(t, copyErr)

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "WebSocket")
	assert.Contains(t, buf.String(), "10.0.0.1:10000")
}

func TestPrintStatusTable(t *testing.T) {
	status := &EdgeHubStatus{
		IsEnabled: true,
		Protocol:  "QUIC",
		Server:    "10.0.0.1:10001",
		Heartbeat: 30,
		CertStatus: CertificateStatus{
			CAPresent:   false,
			CertPresent: false,
			KeyPresent:  false,
		},
	}

	oldStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	assert.NoError(t, pipeErr)
	os.Stdout = w

	printStatusTable(status)

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, copyErr := io.Copy(&buf, r)
	assert.NoError(t, copyErr)

	output := buf.String()
	assert.Contains(t, output, "EDGEHUB STATUS")
	assert.Contains(t, output, "QUIC")
	assert.Contains(t, output, "10.0.0.1:10001")
	assert.Contains(t, output, "30")
	assert.Contains(t, output, "CERTIFICATE STATUS")
}

func TestGetEdgeHubStatusErrorConfig(t *testing.T) {
	patches := gomonkey.ApplyFunc(util.ParseEdgecoreConfig,
		func(configPath string) (*v1alpha2.EdgeCoreConfig, error) {
			return nil, errors.New("config parsing failed")
		})
	defer patches.Reset()

	opts := &EdgeHubStatusOptions{}
	err := opts.getEdgeHubStatus()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get edge config failed")
}

func TestGetEdgeHubStatusTableOutput(t *testing.T) {
	config := v1alpha2.NewDefaultEdgeCoreConfig()
	config.Modules.EdgeHub.Enable = true
	config.Modules.EdgeHub.Heartbeat = 15
	config.Modules.EdgeHub.WebSocket = &v1alpha2.EdgeHubWebSocket{
		Enable: true,
		Server: "10.0.0.1:10000",
	}
	config.Modules.EdgeHub.TLSCAFile = ""
	config.Modules.EdgeHub.TLSCertFile = ""
	config.Modules.EdgeHub.TLSPrivateKeyFile = ""

	patches := gomonkey.ApplyFunc(util.ParseEdgecoreConfig,
		func(configPath string) (*v1alpha2.EdgeCoreConfig, error) {
			return config, nil
		})
	defer patches.Reset()

	oldStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	assert.NoError(t, pipeErr)
	os.Stdout = w

	opts := &EdgeHubStatusOptions{}
	err := opts.getEdgeHubStatus()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, copyErr := io.Copy(&buf, r)
	assert.NoError(t, copyErr)

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "EDGEHUB STATUS")
}

func TestGetEdgeHubStatusJSONOutput(t *testing.T) {
	config := v1alpha2.NewDefaultEdgeCoreConfig()
	config.Modules.EdgeHub.Enable = true
	config.Modules.EdgeHub.Heartbeat = 15
	config.Modules.EdgeHub.WebSocket = &v1alpha2.EdgeHubWebSocket{
		Enable: true,
		Server: "10.0.0.1:10000",
	}
	config.Modules.EdgeHub.TLSCAFile = ""
	config.Modules.EdgeHub.TLSCertFile = ""
	config.Modules.EdgeHub.TLSPrivateKeyFile = ""

	patches := gomonkey.ApplyFunc(util.ParseEdgecoreConfig,
		func(configPath string) (*v1alpha2.EdgeCoreConfig, error) {
			return config, nil
		})
	defer patches.Reset()

	oldStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	assert.NoError(t, pipeErr)
	os.Stdout = w

	opts := &EdgeHubStatusOptions{Output: "json"}
	err := opts.getEdgeHubStatus()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, copyErr := io.Copy(&buf, r)
	assert.NoError(t, copyErr)

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "\"isEnabled\": true")
	assert.Contains(t, buf.String(), "WebSocket")
}
