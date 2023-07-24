FROM golang:1.20 as build

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build -o /go/bin/load-balancer-operator

FROM gcr.io/distroless/static

COPY --from=build /go/bin/load-balancer-operator /load-balancer-operator

ENTRYPOINT ["/load-balancer-operator"]
CMD ["process"]
