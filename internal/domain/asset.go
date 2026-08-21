package domain

import (
	"fmt"
	"strings"
	"time"
)

type AssetState string

const (
	AssetActive  AssetState = "active"
	AssetInvalid AssetState = "invalid"
	AssetExpired AssetState = "expired"
)

type Asset struct {
	ID            ID         `json:"id"`
	Name          string     `json:"name"`
	MediaType     string     `json:"media_type"`
	StorageKey    string     `json:"storage_key"`
	RightsEndsAt  *time.Time `json:"rights_ends_at,omitempty"`
	State         AssetState `json:"state"`
	InvalidReason string     `json:"invalid_reason,omitempty"`
	Version       int64      `json:"version"`
}

func NewAsset(name, mediaType, storageKey string, rightsEndsAt *time.Time) (Asset, error) {
	asset := Asset{ID: NewID("ast"), Name: strings.TrimSpace(name), MediaType: mediaType, StorageKey: storageKey, RightsEndsAt: rightsEndsAt, State: AssetActive, Version: 1}
	if err := asset.Validate(); err != nil {
		return Asset{}, err
	}
	return asset, nil
}

func (a Asset) Validate() error {
	if a.ID.Empty() || a.Name == "" || a.StorageKey == "" {
		return fmt.Errorf("asset identity, name and storage key are required")
	}
	switch a.MediaType {
	case "image/png", "image/jpeg", "video/mp4":
		return nil
	default:
		return fmt.Errorf("unsupported media type %q", a.MediaType)
	}
}

func (a Asset) Eligible(at time.Time) bool {
	if a.State != AssetActive {
		return false
	}
	return a.RightsEndsAt == nil || a.RightsEndsAt.After(at)
}

func (a *Asset) Invalidate(reason string, at time.Time) {
	a.State = AssetInvalid
	a.InvalidReason = strings.TrimSpace(reason)
	if a.RightsEndsAt != nil && !a.RightsEndsAt.After(at) {
		a.State = AssetExpired
	}
	a.Version++
}
