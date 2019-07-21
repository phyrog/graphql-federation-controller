package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
)

var memoryStore = map[types.NamespacedName]*GraphQLBackendConfig{}

func UpdateMessageListener(updateChannel chan UpdateMessage, log logr.Logger) {
	for msg := range updateChannel {
		if msg.Config == nil {
			if _, ok := memoryStore[msg.NamespacedName]; ok {
				log.Info("Removing " + msg.NamespacedName.String() + " from internal store.")
				delete(memoryStore, msg.NamespacedName)
			}
		} else {
			log.Info("Updating " + msg.NamespacedName.String() + " in internal store.")
			memoryStore[msg.NamespacedName] = msg.Config
		}
	}
}

type ImplementingServiceLocation struct {
	Name string
	Path string
}

type BackendConfig struct {
	FormatVersion                int                           `json:"formatVersion"`
	Id                           string                        `json:"id"`
	SchemaHash                   string                        `json:"schemaHash"`
	ImplementingServiceLocations []ImplementingServiceLocation `json:"implementingServiceLocations"`
}

func memoryStoreToBackendConfig(memStore map[types.NamespacedName]*GraphQLBackendConfig) BackendConfig {
	config := BackendConfig{
		FormatVersion:                1,
		Id:                           "schema",
		SchemaHash:                   "schemaHash",
		ImplementingServiceLocations: make([]ImplementingServiceLocation, 0, len(memStore)),
	}

	for k, v := range memStore {
		config.ImplementingServiceLocations = append(config.ImplementingServiceLocations, ImplementingServiceLocation{
			Name: v.PartialName,
			Path: "service/" + k.String(),
		})
	}

	return config
}

// "/config"
func configHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	msg, err := json.Marshal(memoryStoreToBackendConfig(memoryStore))
	if err != nil {
		w.WriteHeader(500)
	} else {
		w.WriteHeader(200)
		fmt.Fprintf(w, string(msg))
	}
}

type BackendService struct {
	URL               string `json:"url"`
	PartialSchemaPath string `json:"partialSchemaPath"`
}

// "/service/:namespace/:service"
func serviceHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	service := types.NamespacedName{Namespace: params.ByName("namespace"), Name: params.ByName("service")}
	backend := memoryStore[service]
	msg, err := json.Marshal(BackendService{
		URL:               backend.Protocol + "://" + backend.Endpoint + ":" + strconv.Itoa(int(backend.Port)) + backend.Path,
		PartialSchemaPath: "schema/" + service.String(),
	})
	if err != nil {
		w.WriteHeader(500)
	} else {
		w.WriteHeader(200)
		fmt.Fprintf(w, string(msg))
	}
}

// "/schema/:namespace/:service"
func schemaHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	service := types.NamespacedName{Namespace: params.ByName("namespace"), Name: params.ByName("service")}
	backend, ok := memoryStore[service]
	if ok {
		w.WriteHeader(200)
		fmt.Fprintf(w, backend.Schema)
	} else {
		w.WriteHeader(500)
	}
}

// "/:graphid/storage-secret/:apikeyhash.json"
func secretHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.WriteHeader(200)
	fmt.Fprintf(w, `"secret"`)
}

type CompositionConfigLink struct {
	ConfigPath string `json:"configPath"`
}

// "/:secret/:graphvariant/v:federationversion/composition-config-link",
func configLinkHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.WriteHeader(200)

	msg, err := json.Marshal(CompositionConfigLink{
		ConfigPath: "config",
	})
	if err != nil {
		w.WriteHeader(500)
	} else {
		w.WriteHeader(200)
		fmt.Fprintf(w, string(msg))
	}
}

type Logger struct {
	handler http.Handler
	log     logr.Logger
}

func (l *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l.log.Info(r.Method + ": " + r.URL.Path)
	l.handler.ServeHTTP(w, r)
}

func StartWebserver(updateChannel chan UpdateMessage, log logr.Logger) {
	go UpdateMessageListener(updateChannel, log)

	router := httprouter.New()
	router.GET("/partial/config", configHandler)
	router.GET("/partial/schema/:namespace/:service", schemaHandler)
	router.GET("/partial/service/:namespace/:service", serviceHandler)
	router.GET("/partial/secret/:graphvariant/v:federationversion/composition-config-link", configLinkHandler)
	router.GET("/secret/:graphid/storage-secret/:apikeyhash.json", secretHandler)

	log.Error(http.ListenAndServe(":8000", &Logger{handler: router, log: log}), "")
}
