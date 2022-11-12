package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-mastodon"
)

type Mux struct {
	apps    map[string]*mastodon.Application
	clients map[string]*mastodon.Client
}

func NewMuxFromCredentialsDir(dir string) (*Mux, error) {
	credPaths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("unable to find credentials file paths: %v", err)
	}

	apps := make(map[string]*mastodon.Application)
	for _, credPath := range credPaths {
		app, err := LoadAppFromJSON(credPath)
		if err != nil {
			return nil, fmt.Errorf("unable to load app from %s: %v", credPath, err)
		}
		apps[app.ClientID] = app
	}

	clients := make(map[string]*mastodon.Client)
	for _, app := range apps {
		server, err := ServerURIFromAppAuthURI(app)
		if err != nil {
			return nil, fmt.Errorf("unable to get server from app: %v", err)
		}

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

func (m *Mux) StreamPublic(ctx context.Context, isLocal bool) (<-chan mastodon.Event, <-chan StreamError) {
	ch := make(chan mastodon.Event)
	errCh := make(chan StreamError)

	// For each client, start a goroutine that streams public events
	// and sends them to the channel.
	for serverName, client := range m.clients {
		go func(serverName string, client *mastodon.Client) {
			// TODO add resuming and max retries
			stream, err := client.StreamingPublic(ctx, isLocal)
			if err != nil {
				errCh <- StreamError{Server: client.Config.Server, Err: err}
			}

			for event := range stream {
				ch <- event
			}
		}(serverName, client)
	}

	return ch, errCh
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
