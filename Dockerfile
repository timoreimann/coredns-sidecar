FROM scratch

ADD coredns-sidecar /
ENTRYPOINT ["/coredns-sidecar"]
