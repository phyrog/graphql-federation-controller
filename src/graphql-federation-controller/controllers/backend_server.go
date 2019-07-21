package controllers

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
)

var memoryStore = map[types.NamespacedName]*GraphQLBackendConfig{}

func UpdateMessageListener(updateChannel chan UpdateMessage, log logr.Logger) {
	for msg := range updateChannel {
		if msg.Config == nil {
			log.Info("Removing " + msg.NamespacedName.String() + " from internal store.")
			delete(memoryStore, msg.NamespacedName)
		} else {
			log.Info("Updating " + msg.NamespacedName.String() + " in internal store.")
			memoryStore[msg.NamespacedName] = msg.Config
		}
	}
}

func StartWebserver(updateChannel chan UpdateMessage, log logr.Logger) {
	go UpdateMessageListener(updateChannel, log)
}
