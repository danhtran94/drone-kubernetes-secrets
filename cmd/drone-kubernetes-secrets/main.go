// Copyright 2018 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package main

import (
	"net/http"
	"time"

	"github.com/drone/drone-go/plugin/secret"
	"github.com/drone/drone-kubernetes-secrets/plugin"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"

	_ "github.com/joho/godotenv/autoload"
)

type config struct {
	Debug     bool   `envconfig:"DEBUG"`
	Address   string `envconfig:"SERVER_ADDRESS"`
	Secret    string `envconfig:"SECRET_KEY"`
	Config    string `envconfig:"KUBERNETES_CONFIG"`
	Namespace string `envconfig:"KUBERNETES_NAMESPACE"`
}

func main() {
	spec := new(config)
	err := envconfig.Process("", spec)
	if err != nil {
		logrus.Fatal(err)
	}

	if spec.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if spec.Secret == "" {
		logrus.Fatalln("missing secret key")
	}
	if spec.Address == "" {
		spec.Address = ":3000"
	}
	if spec.Namespace == "" {
		spec.Namespace = "default"
	}

	client, err := createClient(spec.Config)
	if err != nil {
		logrus.Fatal(err)
	}

	handler := secret.Handler(
		spec.Secret,
		plugin.New(client, spec.Namespace),
		logrus.StandardLogger(),
	)

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			client, err := createClient(spec.Config)
			if err != nil {
				logrus.Fatal(err)
			}

			retry.OnError(retry.DefaultRetry, func(err error) bool {
				return true
			}, func() error {
				handler = secret.Handler(
					spec.Secret,
					plugin.New(client, spec.Namespace),
					logrus.StandardLogger(),
				)
				return nil
			})
		}
	}()

	logrus.Infof("server listening on address %s", spec.Address)

	http.Handle("/", handler)
	logrus.Fatal(http.ListenAndServe(spec.Address, nil))
}

func createClient(path string) (*kubernetes.Clientset, error) {
	if path == "" {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		return kubernetes.NewForConfig(config)
	}

	config, err := clientcmd.BuildConfigFromFlags("", path)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}
