# Deployment

Deploying coredns-sidecar requires the following modifications to an existing CoreDNS installation:

1. add the sidecar to the CoreDNS deployment (see [this patch](coredns-deploy-patch.yml) that can be applied running `kubectl patch --namespace kube-system deploy coredns --patch "$(cat coredns-deploy-patch.yml)"`)
1. expand existing RBAC rules for CoreDNS to permit listing and watching nodes
1. amend the CoreDNS Corefile by a piece of configuration for the hosts plugin:

```
hosts /shared/hosts {
    ttl 5
    fallthrough
}
```

`fallthrough` is needed so that lookups which cannot be fulfilled by the hosts file get relayed to other resolution mechanisms provided by CoreDNS. `tls` can be adjusted per your own discretion.
