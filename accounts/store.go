package accounts

import "github.com/mattn/go-mastodon"

type Store interface {
	LoadByClientID(clientID string) (*NamedApplication, error)
	GetByServerName(serverName string) (*mastodon.Application, error)
	GetAll() (map[string]*mastodon.Application, error)
}
