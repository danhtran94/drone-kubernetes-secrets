// Copyright 2018 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package plugin

import (
	"context"
	"errors"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/secret"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// New returns a new secret plugin that sources secrets
// from the Kubernetes secrets manager.
func New(client *kubernetes.Clientset, namespace string) secret.Plugin {
	return &plugin{
		namespace: namespace,
		client:    client,
	}
}

type plugin struct {
	client    *kubernetes.Clientset
	namespace string
}

func (p *plugin) Find(ctx context.Context, req *secret.Request) (*drone.Secret, error) {
	if req.Path == "" {
		return nil, errors.New("invalid or missing secret path")
	}
	if req.Name == "" {
		return nil, errors.New("invalid or missing secret name")
	}

	path := req.Path
	name := req.Name

	// makes an api call to the kubernetes secrets manager and
	// attempts to retrieve the secret at the requested path.
	var secret *v1.Secret
	secret, err := p.client.CoreV1().Secrets(p.namespace).Get(ctx, path, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	data, ok := secret.Data[name]
	if !ok {
		return nil, errors.New("secret not found")
	}

	// the user can filter out requets based on event type
	// using the X-Drone-Events secret key. Check for this
	// user-defined filter logic.
	events := extractEvents(secret.ObjectMeta.Annotations)
	if !match(req.Build.Event, events) {
		return nil, errors.New("access denied: event does not match")
	}

	// the user can filter out requets based on repository
	// using the X-Drone-Repos secret key. Check for this
	// user-defined filter logic.
	repos := extractRepos(secret.ObjectMeta.Annotations)
	if !match(req.Repo.Slug, repos) {
		return nil, errors.New("access denied: repository does not match")
	}

	return &drone.Secret{
		Name: name,
		Data: string(data),
		Pull: true, // always true. use X-Drone-Events to prevent pull requests.
		Fork: true, // always true. use X-Drone-Events to prevent pull requests.
	}, nil
}
