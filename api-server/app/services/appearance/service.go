package appearance

import (
	"api-server/v2/modelMethods/appearance"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

type Service struct{ Store appearance.Store }

type Settings struct {
	Version    int64  `json:"version"`
	Theme      Theme  `json:"theme"`
	CanRestore bool   `json:"canRestore"`
	Action     string `json:"action,omitempty"`
}

var (
	ErrInvalidAction = errors.New("Choose a publish action")
	ErrNoPrevious    = errors.New("There is no previous theme")
	ErrReadPrevious  = errors.New("Previous appearance could not be loaded")
)

type ValidationError struct{ error }

func (s Service) Read(ctx context.Context, previous bool) (Settings, error) {
	revision, err := s.Store.Read(ctx, previous)
	if err != nil {
		return Settings{}, err
	}
	settings := Settings{Version: revision.Version, CanRestore: revision.Version > 1}
	err = json.Unmarshal(revision.Settings, &settings.Theme)
	return settings, err
}

func (s Service) Publish(ctx context.Context, actor int, input Settings) (Settings, error) {
	switch input.Action {
	case "restore":
		previous, err := s.Read(ctx, true)
		if errors.Is(err, sql.ErrNoRows) {
			return Settings{}, ErrNoPrevious
		}
		if err != nil {
			return Settings{}, fmt.Errorf("%w: %w", ErrReadPrevious, err)
		}
		input.Theme = previous.Theme
	case "defaults":
		input.Theme = Theme{}
	case "publish":
		input.Theme = input.Theme.WithDefaults()
	default:
		return Settings{}, ErrInvalidAction
	}
	if err := input.Theme.Validate(); err != nil {
		return Settings{}, &ValidationError{err}
	}
	data, err := json.Marshal(input.Theme)
	if err != nil {
		return Settings{}, err
	}
	version, err := s.Store.Publish(ctx, input.Version, data, actor)
	if err != nil {
		return Settings{}, err
	}
	return Settings{Version: version, Theme: input.Theme.WithDefaults(), CanRestore: true}, nil
}
