FROM gcr.io/distroless/static

# Copy the binary that goreleaser built
COPY load-balancer-operator /load-balancer-operator

ENTRYPOINT ["/load-balancer-operator"]
CMD ["process"]
