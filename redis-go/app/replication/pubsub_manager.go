package replication

import (
	"sync"

	"github.com/rs/zerolog"
)

type PubSubManager struct {
	subscribers          map[string]chan PubSubEvent
	SubscriptionsChannel chan SubscriberEvent
	EventsChannel        chan PubSubEvent
	Logger               zerolog.Logger
	mu                   sync.RWMutex
}

type PubSubEvent string

const (
	SubscribeAction   string = "subscribe"
	UnsubscribeAction string = "unsubscribe"
)

type SubscriberEvent struct {
	Action            string
	SubscriberId      string
	SubscriberChannel chan PubSubEvent
}

func NewPubSubManager(logger zerolog.Logger) PubSubManager {
	return PubSubManager{
		subscribers:          make(map[string]chan PubSubEvent),
		SubscriptionsChannel: make(chan SubscriberEvent),
		EventsChannel:        make(chan PubSubEvent),
		Logger:               logger,
	}
}

func (mgr *PubSubManager) Start() {
	// listen for subscriber events
	go func() {
		for event := range mgr.SubscriptionsChannel {
			switch event.Action {
			case SubscribeAction:
				mgr.mu.Lock()

				mgr.Logger.Info().Str("subscriber_id", event.SubscriberId).Msg("Subscribing")
				mgr.subscribers[event.SubscriberId] = event.SubscriberChannel
				mgr.mu.Unlock()
			case UnsubscribeAction:
				mgr.mu.Lock()
				mgr.Logger.Info().Str("subscriber_id", event.SubscriberId).Msg("Unsubscribing")
				delete(mgr.subscribers, event.SubscriberId)
				mgr.mu.Unlock()
			}
		}
	}()

	// fanout events to subscribers
	go func() {
		for event := range mgr.EventsChannel {
			mgr.mu.RLock()
			for _, channel := range mgr.subscribers {
				channel <- event
			}
			mgr.mu.RUnlock()
		}
	}()

	mgr.Logger.Info().Msg("Pubsub manager started...")
}
