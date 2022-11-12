package accounts

import "github.com/mattn/go-mastodon"

type NamedApplication struct {
	ServerName string                `json:"server_name"`
	App        *mastodon.Application `json:"app"`
}
