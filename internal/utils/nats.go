package utils

import (
	natsserver "github.com/nats-io/nats-server/v2/server"
	natsservertest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
)

func RunServer() *natsserver.Server {
	opts := natsservertest.DefaultTestOptions
	opts.Port = natsserver.RANDOM_PORT

	return RunServerWithOptions(&opts)
}

func RunServerWithOptions(opts *natsserver.Options) *natsserver.Server {
	return natsservertest.RunServer(opts)
}

// GetJetstreamConnection returns a JetStreamContext for use within tests
func GetJetstreamConnection(ns *natsserver.Server) nats.JetStreamContext {
	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		panic(err)
	}

	js, err := nc.JetStream()
	if err != nil {
		panic(err)
	}

	return js
}
