package main

import (
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const hostsFileHeader = "## THIS IS AN AUTO-GENERATED HOSTS FILE -- DO NOT EDIT.\n#"

// hostsFile maps from host names to IP addresses.
type hostsFile map[string]string

func (hf hostsFile) String() string {
	// Sort keys for stable output.
	var hosts []string
	for host := range hf {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)

	lines := []string{hostsFileHeader}
	for _, host := range hosts {
		line := fmt.Sprintf("%s\t\t%s", hf[host], host)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n") + "\n"
}

// hostRecord describes an entry in the /etc/hosts file.
type hostRecord struct {
	ipAddr   string
	hostname string
}

func (hr *hostRecord) String() string {
	addr := hr.ipAddr
	if addr == "" {
		addr = "<removed>"
	}

	return fmt.Sprintf("%s -> %s", hr.hostname, addr)
}

// toHostRecord creates a host record from a node.
func toHostRecord(node *corev1.Node) hostRecord {
	rec := hostRecord{hostname: node.ObjectMeta.Name}
	for _, nodeAddr := range node.Status.Addresses {
		switch nodeAddr.Type {
		case corev1.NodeInternalIP:
			rec.ipAddr = nodeAddr.Address
		case corev1.NodeExternalIP:
			// Use external IP address only if internal one cannot be used.
			if rec.ipAddr == "" {
				rec.ipAddr = nodeAddr.Address
			}
		}
	}
	return rec
}
