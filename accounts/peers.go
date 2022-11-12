package accounts

import (
	"context"
	"github.com/mattn/go-mastodon"
	"sync"
	"time"
)

type PeersOf struct {
	Server string    `json:"server"`
	Peers  []string  `json:"peers"`
	Err    error     `json:"err"`
	At     time.Time `json:"at"`
}

func GetAllPeers(
	ctx context.Context,
	apps map[string]*mastodon.Application,
	maxConcurrent int,
	ignoring map[string]bool,
	perServerTimeout time.Duration,
) <-chan PeersOf {
	outputCh := make(chan PeersOf)

	// Get a list of every server name that isn't in ignoring.
	var servers []string
	for server := range apps {
		if !ignoring[server] {
			servers = append(servers, server)
		}
	}

	if len(servers) == 0 {
		close(outputCh)
		return outputCh
	}

	// Create a channel as a queue of servers to process.
	queue := make(chan string, len(servers))
	for _, server := range servers {
		queue <- server
	}
	close(queue)

	// Create up to maxConcurrent goroutines to process the queue.
	var wg sync.WaitGroup
	goRoutines := maxConcurrent
	if len(servers) < maxConcurrent {
		goRoutines = len(servers)
	}
	wg.Add(goRoutines)

	for i := 0; i < goRoutines; i++ {
		go func() {
			defer wg.Done()
			for server := range queue {
				peers, err := GetPeers(ctx, server, apps[server], perServerTimeout)
				outputCh <- PeersOf{
					Server: server, Peers: peers, Err: err, At: time.Now(),
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(outputCh)
	}()

	return outputCh
}

func GetPeers(
	ctx context.Context,
	server string,
	app *mastodon.Application,
	timeout time.Duration,
) ([]string, error) {
	client := mastodon.NewClient(&mastodon.Config{
		Server:       server,
		ClientID:     app.ClientID,
		ClientSecret: app.ClientSecret,
	})

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	peers, err := client.GetInstancePeers(ctx)
	if err != nil {
		return nil, err
	}

	return peers, nil
}

type Registration struct {
	Server string
	App    *mastodon.Application
	Err    error
}

func RegisterAll(
	ctx context.Context,
	frontier map[string]bool,
	maxConcurrent int,
	perServerTimeout time.Duration,
	clientName string,
	requiredAppScopes string,
	appWebsite string,
) <-chan Registration {
	// Register all the peers we found.
	queue := make(chan string, len(frontier))
	for server := range frontier {
		queue <- server
	}
	close(queue)

	if len(frontier) < maxConcurrent {
		maxConcurrent = len(frontier)
	}

	ch := make(chan Registration)

	var wg sync.WaitGroup
	wg.Add(maxConcurrent)

	for i := 0; i < maxConcurrent; i++ {
		go func() {
			defer wg.Done()

			for server := range queue {
				func(server string) {
					ctx, timeout := context.WithTimeout(ctx, perServerTimeout)
					defer timeout()

					app, err := mastodon.RegisterApp(ctx, &mastodon.AppConfig{
						Server:     server,
						ClientName: clientName,
						Scopes:     requiredAppScopes,
						Website:    appWebsite,
					})

					ch <- Registration{Server: server, App: app, Err: err}
				}(server)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	return ch
}
