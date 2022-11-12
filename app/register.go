package app

import (
	"context"
	"fmt"
	"github.com/mattn/go-mastodon"
	"log"
)

func main() {
	app, err := mastodon.RegisterApp(context.Background(), &mastodon.AppConfig{
		Server:     "https://mstdn.jp",
		ClientName: "client-name",
		Scopes:     "read write follow",
		Website:    "https://github.com/mattn/go-mastodon",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("client-id    : %s\n", app.ClientID)
	fmt.Printf("client-secret: %s\n", app.ClientSecret)
}
