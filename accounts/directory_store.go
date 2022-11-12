package accounts

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/mattn/go-mastodon"
)

type DirectoryStore struct {
	dirPath string
	apps    map[string]*mastodon.Application

	sync.Mutex
}

func NewDirectoryStorage(dirPath string) (*DirectoryStore, error) {
	err := ensureDirectory(dirPath)
	if err != nil {
		return nil, err
	}

	return &DirectoryStore{
		dirPath: dirPath,
		apps:    make(map[string]*mastodon.Application),
	}, nil
}

func (ds *DirectoryStore) LoadByClientID(clientID string) (*NamedApplication, error) {
	return ds.LoadFromPath(path.Join(ds.dirPath, clientID+".json"))
}

func (ds *DirectoryStore) LoadFromPath(filePath string) (*NamedApplication, error) {
	fp, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = fp.Close() }()

	var app NamedApplication
	if err := json.NewDecoder(fp).Decode(&app); err != nil {
		return nil, err
	}

	return &app, nil
}

func (ds *DirectoryStore) GetByServerName(serverName string) (*mastodon.Application, error) {
	ds.Lock()
	defer ds.Unlock()

	// Try to load from existing apps
	app, ok := ds.apps[serverName]
	if ok {
		return app, nil
	}

	// Try to reload
	apps, err := ds.loadAll()
	if err != nil {
		return nil, err
	}
	ds.apps = apps

	// Try from existing apps again
	app, ok = ds.apps[serverName]
	if !ok {
		return nil, fmt.Errorf("no app for server %s", serverName)
	}

	return app, nil
}

func (ds *DirectoryStore) GetAll() (map[string]*mastodon.Application, error) {
	// Reload/Ensure loaded
	err := ds.LoadAll()
	if err != nil {
		return nil, err
	}

	ds.Lock()
	defer ds.Unlock()

	// Copy the map
	apps := make(map[string]*mastodon.Application)
	for k, v := range ds.apps {
		apps[k] = v
	}

	return apps, nil
}

func (ds *DirectoryStore) LoadAll() error {
	apps, err := ds.loadAll()
	if err != nil {
		return err
	}

	ds.Lock()
	defer ds.Unlock()
	ds.apps = apps

	return nil
}

func (ds *DirectoryStore) loadAll() (map[string]*mastodon.Application, error) {
	credPaths, err := filepath.Glob(filepath.Join(ds.dirPath, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("unable to find credentials file paths: %v", err)
	}

	// Technically this is kinda wrong if you are going to store multiple credentials
	// for each server but i am not so for now too bad.
	apps := make(map[string]*mastodon.Application)
	for _, credPath := range credPaths {
		pair, err := ds.LoadFromPath(credPath)
		if err != nil {
			return nil, fmt.Errorf("unable to load app from %s: %v", credPath, err)
		}
		apps[pair.ServerName] = pair.App
	}

	return apps, nil
}

func (ds *DirectoryStore) WriteApp(serverName string, app *mastodon.Application) (string, error) {
	namedApp := &NamedApplication{
		ServerName: serverName,
		App:        app,
	}

	asJson, err := json.MarshalIndent(namedApp, "", "  ")
	if err != nil {
		return "", err
	}

	filePath := filepath.Join(ds.dirPath, app.ClientID+".json")
	err = os.WriteFile(filePath, asJson, 0644)
	if err != nil {
		return "", err
	}

	return filePath, err
}

func ensureDirectory(dirPath string) error {
	// If the dirPath does not exist make the directory.
	if fileInfo, err := os.Stat(dirPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		err := os.MkdirAll(dirPath, 0700)
		if err != nil {
			return err
		}
	} else {
		// Verify this is a directory
		if !fileInfo.IsDir() {
			return fmt.Errorf("%s is not a directory", dirPath)
		}
	}
	return nil
}
