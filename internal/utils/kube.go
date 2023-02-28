package utils

import (
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func StartKube() (*envtest.Environment, *rest.Config) {
	env := envtest.Environment{}

	cfg, err := env.Start()
	if err != nil {
		panic(err)
	}

	return &env, cfg
}
