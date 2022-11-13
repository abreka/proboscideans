package streaming

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/abreka/proboscideans/accounts"

	"github.com/mattn/go-mastodon"
)

type Mux struct {
	apps    map[string]*mastodon.Application
	clients map[string]*mastodon.Client
}

func NewMuxFromCredentialsDir(accountStore accounts.Store) (*Mux, error) {
	apps, err := accountStore.GetAll()
	if err != nil {
		return nil, err
	}

	clients := make(map[string]*mastodon.Client)
	for server, app := range apps {
		clients[server] = mastodon.NewClient(&mastodon.Config{
			Server:       server,
			ClientID:     app.ClientID,
			ClientSecret: app.ClientSecret,
		})
	}

	return &Mux{
		apps:    apps,
		clients: clients,
	}, nil
}

type StreamError struct {
	Server string `json:"server"`
	Err    error  `json:"error"`
}

func (m *Mux) StreamPublic(ctx context.Context, isLocal bool) (<-chan mastodon.Event, <-chan *StreamError) {
	ch := make(chan mastodon.Event)
	errCh := make(chan *StreamError)

	// For each client, start a goroutine that streams public events
	// and sends them to the channel.
	for serverName, client := range m.clients {
		// TODO: client has a Config.Server field
		// TODO: better error handling and retries. right now this just dies which
		// is good for not hammering but bad for working
		go streamPublicSafely(ctx, serverName, client, isLocal, ch, errCh)
	}

	return ch, errCh
}

func streamPublicSafely(ctx context.Context, serverName string, client *mastodon.Client, isLocal bool, ch chan<- mastodon.Event, errCh chan<- *StreamError) {
	// TODO: this is a kludge to work around the fact that the client
	// will keep hammering on an error in a tight loop.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := client.StreamingPublic(ctx, isLocal)
	if err != nil {
		errCh <- &StreamError{Server: serverName, Err: err}
		return
	}

	for event := range stream {
		switch event := event.(type) {
		case *mastodon.ErrorEvent:
			errCh <- &StreamError{
				Server: serverName,
				Err:    errors.New(event.Error()),
			}
			return
		default:
			ch <- event
		}
	}
}

func ServerURIFromAppAuthURI(app *mastodon.Application) (string, error) {
	// Find the "/oauth/" in the auth uri and use that as the server.
	// This is a KLUDGE to work around the fact that the app doesn't
	// store the server but I think they all have the same url structure
	// so it should be fine for now...until it isn.t
	i := strings.Index(app.AuthURI, "/oauth/")
	if i == -1 {
		return "", fmt.Errorf("unable to find /oauth/ in auth uri: %s", app.AuthURI)
	}
	return app.AuthURI[:i], nil
}

// LoadAppFromJSON loads an app from a JSON file
func LoadAppFromJSON(filePath string) (*mastodon.Application, error) {
	fp, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = fp.Close() }()

	var app mastodon.Application
	if err := json.NewDecoder(fp).Decode(&app); err != nil {
		return nil, err
	}

	return &app, nil
}
