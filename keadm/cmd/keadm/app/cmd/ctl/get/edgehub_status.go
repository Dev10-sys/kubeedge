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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/kubeedge/api/apis/common/constants"
	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var edgeHubStatusShortDescription = `Get the health status of EdgeHub component`

// EdgeHubStatusOptions holds the configuration for the edgehub-status command
type EdgeHubStatusOptions struct {
	Output string
}

// EdgeHubStatus represents the health status of EdgeHub
type EdgeHubStatus struct {
	IsEnabled     bool               `json:"isEnabled"`
	Protocol      string             `json:"protocol"`
	Server        string             `json:"server"`
	Heartbeat     int32              `json:"heartbeatSeconds"`
	CertStatus    CertificateStatus  `json:"certificateStatus"`
}

// CertificateStatus holds the validity info for TLS certificates
type CertificateStatus struct {
	CAPresent      bool   `json:"caPresent"`
	CertPresent    bool   `json:"certPresent"`
	KeyPresent     bool   `json:"keyPresent"`
	CertExpiry     string `json:"certExpiry,omitempty"`
	CertValid      bool   `json:"certValid"`
}

// NewEdgeHubStatusGet returns a cobra command for getting edgehub status.
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

	cmd.Flags().StringVarP(&opts.Output, common.FlagNameOutput, "o", opts.Output,
		"Output format. Supports: json|yaml")

	return cmd
}

func (o *EdgeHubStatusOptions) getEdgeHubStatus() error {
	config, err := util.ParseEdgecoreConfig(constants.EdgecoreConfigPath)
	if err != nil {
		return fmt.Errorf("get edge config failed with err:%v", err)
	}

	status := buildEdgeHubStatus(config)

	if o.Output == "json" || o.Output == "yaml" {
		return printStatusJSON(status)
	}

	printStatusTable(status)
	return nil
}

// buildEdgeHubStatus constructs an EdgeHubStatus from the edgecore config.
func buildEdgeHubStatus(config *v1alpha2.EdgeCoreConfig) *EdgeHubStatus {
	hub := config.Modules.EdgeHub

	status := &EdgeHubStatus{
		IsEnabled: hub.Enable,
		Heartbeat: hub.Heartbeat,
	}

	// Determine active protocol and server address
	if hub.WebSocket != nil && hub.WebSocket.Enable {
		status.Protocol = "WebSocket"
		status.Server = hub.WebSocket.Server
	} else if hub.Quic != nil && hub.Quic.Enable {
		status.Protocol = "QUIC"
		status.Server = hub.Quic.Server
	} else {
		status.Protocol = "Unknown"
		status.Server = ""
	}

	// Check certificate files
	status.CertStatus = checkCertificates(hub)

	return status
}

// checkCertificates validates the presence and expiry of TLS certificates.
func checkCertificates(hub *v1alpha2.EdgeHub) CertificateStatus {
	cs := CertificateStatus{}

	cs.CAPresent = fileExists(hub.TLSCAFile)
	cs.CertPresent = fileExists(hub.TLSCertFile)
	cs.KeyPresent = fileExists(hub.TLSPrivateKeyFile)

	if cs.CertPresent && cs.KeyPresent {
		expiry, valid, err := getCertExpiry(hub.TLSCertFile, hub.TLSPrivateKeyFile)
		if err == nil {
			cs.CertExpiry = expiry.Format(time.RFC3339)
			cs.CertValid = valid
		}
	}

	return cs
}

// fileExists checks whether a file exists at the given path.
func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// getCertExpiry loads the TLS cert/key pair and returns the NotAfter time and
// whether the certificate is currently valid.
func getCertExpiry(certFile, keyFile string) (time.Time, bool, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return time.Time{}, false, err
	}
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return time.Time{}, false, err
	}
	now := time.Now()
	valid := now.After(x509Cert.NotBefore) && now.Before(x509Cert.NotAfter)
	return x509Cert.NotAfter, valid, nil
}

// printStatusJSON prints the EdgeHubStatus in JSON format
func printStatusJSON(status *EdgeHubStatus) error {
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// printStatusTable prints the EdgeHubStatus in a human-readable table
func printStatusTable(status *EdgeHubStatus) {
	fmt.Println("EDGEHUB STATUS")
	fmt.Println("──────────────────────────────────────")
	fmt.Printf("  Enabled:              %v\n", status.IsEnabled)
	fmt.Printf("  Protocol:             %s\n", status.Protocol)
	fmt.Printf("  Server:               %s\n", status.Server)
	fmt.Printf("  Heartbeat Interval:   %ds\n", status.Heartbeat)
	fmt.Println()
	fmt.Println("CERTIFICATE STATUS")
	fmt.Println("──────────────────────────────────────")
	fmt.Printf("  CA File Present:      %v\n", status.CertStatus.CAPresent)
	fmt.Printf("  Cert File Present:    %v\n", status.CertStatus.CertPresent)
	fmt.Printf("  Key File Present:     %v\n", status.CertStatus.KeyPresent)
	if status.CertStatus.CertExpiry != "" {
		fmt.Printf("  Cert Expiry:          %s\n", status.CertStatus.CertExpiry)
		fmt.Printf("  Cert Valid:           %v\n", status.CertStatus.CertValid)
	}
}
