package analytics

import (
	"net/http"
	"sync"

	"github.com/getlantern/flashlight/config"
	"github.com/mitchellh/mapstructure"

	"github.com/getlantern/analytics"
	"github.com/getlantern/flashlight/ui"
	"github.com/getlantern/golog"
)

const (
	messageType = `Analytics`
)

var (
	log        = golog.LoggerFor("flashlight.analytics")
	service    *ui.Service
	httpClient *http.Client
	hostName   *string
	cfgMutex   sync.Mutex
)

func Configure(cfg *config.Config, serverSession bool, newClient *http.Client) {

	cfgMutex.Lock()
	defer cfgMutex.Unlock()

	httpClient = newClient

	sessionPayload := &analytics.Payload{
		ClientId:      cfg.InstanceId,
		ClientVersion: string(cfg.Version),
		HitType:       analytics.EventType,
		Event: &analytics.Event{
			Category: "Session",
			Action:   "Start",
		},
	}

	if serverSession {
		sessionPayload.Hostname = cfg.Server.AdvertisedHost
	} else {
		sessionPayload.Hostname = "localhost"
	}

	analytics.SessionEvent(httpClient, sessionPayload)

	if !serverSession && cfg.AutoReport != nil && *cfg.AutoReport {
		err := startService()
		if err != nil {
			log.Errorf("Error starting analytics service: %q", err)
		}
	}
}

// Used with clients to track user interaction with the UI
func startService() error {

	var err error

	if service != nil {
		return nil
	}

	newMessage := func() interface{} {
		return &analytics.Payload{}
	}

	if service, err = ui.Register(messageType, newMessage, nil); err != nil {
		log.Errorf("Unable to register analytics service: %q", err)
		return err
	}

	// process analytics messages
	go read()

	return err

}

func read() {

	for msg := range service.In {
		log.Debugf("New UI analytics message: %q", msg)
		var payload analytics.Payload
		if err := mapstructure.Decode(msg, &payload); err != nil {
			log.Errorf("Could not decode payload: %q", err)
		} else {
			// set to localhost on clients
			payload.Hostname = "localhost"
			payload.HitType = analytics.PageViewType
			// for now, the only analytics messages we are
			// currently receiving from the UI are initial page
			// views which indicate new UI sessions
			analytics.UIEvent(httpClient, &payload)
		}
	}
}
