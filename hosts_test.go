package main

import (
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createNode(name string, nodeAddresses []corev1.NodeAddress) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: corev1.NodeStatus{
			Addresses: nodeAddresses,
		},
	}
}

func hostsFileDiff(got, want string) string {
	return fmt.Sprintf(`got
BOM
%s
EOM
want
BOM
%s
EOM
`, got, want)
}

func TestHostsFileString(t *testing.T) {
	hf := hostsFile{
		"host4": "4.4.4.4",
		"host1": "1.1.1.1",
		"host3": "3.3.3.3",
		"host2": "2.2.2.2",
	}

	want := fmt.Sprintf("%s\n"+
		"1.1.1.1\t\thost1\n"+
		"2.2.2.2\t\thost2\n"+
		"3.3.3.3\t\thost3\n"+
		"4.4.4.4\t\thost4\n"+
		"", hostsFileHeader)

	got := hf.String()
	if got != want {
		t.Error(hostsFileDiff(got, want))
	}
}

func TestHostRecordString(t *testing.T) {
	tests := []struct {
		name string
		in   hostRecord
		want string
	}{
		{
			name: "adding host record",
			in: hostRecord{
				ipAddr:   "1.1.1.1",
				hostname: "host",
			},
			want: "host -> 1.1.1.1",
		},
		{
			name: "removing host record",
			in: hostRecord{
				ipAddr:   "",
				hostname: "host",
			},
			want: "host -> <removed>",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.in.String()
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestToHostRecord(t *testing.T) {
	const hostname = "host"

	tests := []struct {
		name          string
		nodeAddresses []corev1.NodeAddress
		want          hostRecord
	}{
		{
			name: "internal IP address only",
			nodeAddresses: []corev1.NodeAddress{
				corev1.NodeAddress{
					Type:    corev1.NodeInternalIP,
					Address: "1.1.1.1",
				},
			},
			want: hostRecord{
				ipAddr:   "1.1.1.1",
				hostname: hostname,
			},
		},
		{
			name: "external IP address only",
			nodeAddresses: []corev1.NodeAddress{
				corev1.NodeAddress{
					Type:    corev1.NodeExternalIP,
					Address: "1.1.1.1",
				},
			},
			want: hostRecord{
				ipAddr:   "1.1.1.1",
				hostname: hostname,
			},
		},
		{
			name: "internal and external IP addresses",
			nodeAddresses: []corev1.NodeAddress{
				corev1.NodeAddress{
					Type:    corev1.NodeInternalIP,
					Address: "1.1.1.1",
				},
				corev1.NodeAddress{
					Type:    corev1.NodeExternalIP,
					Address: "2.2.2.2",
				},
			},
			want: hostRecord{
				ipAddr:   "1.1.1.1",
				hostname: hostname,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			node := createNode(hostname, test.nodeAddresses)

			got := toHostRecord(node)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}
