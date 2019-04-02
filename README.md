# coredns-sidecar

_coredns-sidecar_ provides additional functionality to CoreDNS when run inside Kubernetes.

Currently, it supports watching over nodes and generating an `/etc/hosts` file that can be consumed by CoreDNS through the [hosts plugin](https://coredns.io/plugins/hosts/).

## sidecar vs. plugin

CoreDNS supports extending functionality through a plugin mechanism. While this makes the implementation easier, it also requires building the plugin binary from CoreDNS source (presumably matching the version of the CoreDNS version intended to run on your Kubernetes cluster).

The sidecar solution depends on the hosts plugin only (at the price of added complexity).

## Usage

```bash
./coredns-sidecar -hosts <path to hostsfile>
```

To increase the amount of logging, add `-v=3` as parameter.

## Makefile targets

- `make test`: run tests
- `make build`: compile a native binary into `bin/$OS_$ARCH`.
- `make container`: create a Docker image (set the `VERSION` environment variable to choose a custom image tag)
- `make push`: push the Docker image

## Releasing

1. Update the `VERSION` variable in the Makefile.
1. Create and push a version tag like `vX.Y.Z`.
1. Build and push a Docker with `make push`.
1. Create a [new Github release](https://github.com/timoreimann/coredns-sidecar/releases/new).
