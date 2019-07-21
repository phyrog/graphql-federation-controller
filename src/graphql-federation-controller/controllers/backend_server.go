package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
)

var memoryStore = map[types.NamespacedName]*GraphQLBackendConfig{}

func UpdateMessageListener(updateChannel chan UpdateMessage, log logr.Logger) {
	for msg := range updateChannel {
		if _, ok := memoryStore[msg.NamespacedName]; ok && msg.Config == nil {
			log.Info("Removing " + msg.NamespacedName.String() + " from internal store.")
			delete(memoryStore, msg.NamespacedName)
		} else {
			log.Info("Updating " + msg.NamespacedName.String() + " in internal store.")
			memoryStore[msg.NamespacedName] = msg.Config
		}
	}
}

func secretHandler(w http.ResponseWriter, r *http.Request) {
	msg, _ := json.Marshal(memoryStore)
	w.WriteHeader(200)
	fmt.Fprintf(w, string(msg))
}

func StartWebserver(updateChannel chan UpdateMessage, log logr.Logger) {
	go UpdateMessageListener(updateChannel, log)

	http.HandleFunc("/", secretHandler)
	log.Error(http.ListenAndServe(":8000", nil), "")
}
