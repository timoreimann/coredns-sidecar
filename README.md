# coredns-sidecar

_coredns-sidecar_ provides additional functionality to CoreDNS when run inside Kubernetes.

Currently, it supports watching over nodes and generating an `/etc/hosts` file that can be consumed by CoreDNS through the [hosts plugin](https://coredns.io/plugins/hosts/).
