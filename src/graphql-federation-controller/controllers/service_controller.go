/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Log    logr.Logger
	Config ServiceReconcilerConfig
}

// ServiceReconcilerConfig is used to configure the ServiceReconciler
type ServiceReconcilerConfig struct {
	SchemaName string
}

// GraphQLBackendConfig encapsulates the information about one backend service
type GraphQLBackendConfig struct {
	PartialName string
	Port        int32
	Path        string
	Endpoint    string
	Protocol    string
	Schema      string
}

func ignoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return nil
	}
	return err
}

func parseGraphQLBackendConfig(service corev1.Service) (GraphQLBackendConfig, error) {
	namespace := service.ObjectMeta.Namespace
	name := service.ObjectMeta.Name

	partialName, ok := service.ObjectMeta.Annotations["schema.graphql.org/partial"]
	if !ok {
		partialName = namespace + "/" + name
	}

	var portNumber int32
	portFound := false
	portName, ok := service.ObjectMeta.Annotations["schema.graphql.org/port"]

	if !ok {
		if len(service.Spec.Ports) == 1 {
			portNumber = service.Spec.Ports[0].Port
			portFound = true
		} else {
			portName = "graphql"
		}
	}

	if !portFound {
		for _, port := range service.Spec.Ports {
			if port.Name == portName {
				portNumber = port.Port
				portFound = true
			}
		}

		if !portFound {
			port, err := strconv.Atoi(portName)
			if err == nil {
				portNumber = int32(port)
				portFound = true
			}
		}

		if !portFound {
			return GraphQLBackendConfig{}, errors.New("Specified port not found")
		}
	}

	path, ok := service.ObjectMeta.Annotations["schema.graphql.org/path"]
	if !ok {
		path = "/graphql"
	}

	protocol, ok := service.ObjectMeta.Annotations["schema.graphql.org/protocol"]
	if !ok {
		protocol = "http"
	}

	ip := service.Spec.ClusterIP

	return GraphQLBackendConfig{
		PartialName: partialName,
		Endpoint:    ip,
		Port:        portNumber,
		Path:        path,
		Protocol:    protocol,
	}, nil
}

func buildGraphQLEndpointURL(config GraphQLBackendConfig) string {
	return config.Protocol + "://" + config.Endpoint + ":" + strconv.Itoa(int(config.Port)) + config.Path
}

type SchemaResult struct {
	Data struct {
		Service struct {
			Sdl string
		} `json:"_service"`
	}
}

var memoryStore = map[types.NamespacedName]*GraphQLBackendConfig{}

// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch

func (r *ServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("service", req.NamespacedName)

	var service corev1.Service
	if err := r.Client.Get(ctx, req.NamespacedName, &service); err != nil {
		log.Error(err, "unable to fetch Service")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, ignoreNotFound(err)
	}

	// Annotations:
	// schema.graphql.org/name: Schema name; used to identify which schema a partial schema belongs to
	// schema.graphql.org/partial: Name of the partial schema
	// schema.graphql.org/port: Port to be used on the service; either the name of
	// 	a port, or a port number; default is the only defined port, or in case of
	// 	multiple defined ports, the port with name "graphql"
	// scheme.graphql.org/path: graphql endpoint path; default: "/graphql"

	if val, ok := service.ObjectMeta.Annotations["schema.graphql.org/name"]; ok && val == r.Config.SchemaName {
		config, err := parseGraphQLBackendConfig(service)
		if err != nil {
			return ctrl.Result{}, err
		}

		endpointURL := buildGraphQLEndpointURL(config)
		log.Info(config.PartialName + ": " + endpointURL)

		payload := []byte(`{"query":"{_service{sdl}}"}`)
		log.Info(string(payload))

		httpReq, err := http.NewRequest("POST", endpointURL, bytes.NewBuffer(payload))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "application/json")

		client := &http.Client{}
		resp, err := client.Do(httpReq)
		if err != nil {
			return ctrl.Result{}, err
		}
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		var result SchemaResult
		json.Unmarshal(body, &result)
		config.Schema = result.Data.Service.Sdl

		memoryStore[req.NamespacedName] = &config

		log.Info(config.Schema)

		// your logic here

	} else {
		if _, ok := memoryStore[req.NamespacedName]; ok {
			delete(memoryStore, req.NamespacedName)
		}
	}

	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Complete(r)
}
