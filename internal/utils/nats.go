package utils

import (
	natsserver "github.com/nats-io/nats-server/v2/server"
	natsservertest "github.com/nats-io/nats-server/v2/test"
)

func RunServer() *natsserver.Server {
	opts := natsservertest.DefaultTestOptions
	opts.Port = natsserver.RANDOM_PORT

	return RunServerWithOptions(&opts)
}

func RunServerWithOptions(opts *natsserver.Options) *natsserver.Server {
	return natsservertest.RunServer(opts)
}
