package domain

import (
	"fmt"
	"sort"
	"strings"
)

type PackageItem struct {
	AssetID     ID   `json:"asset_id"`
	Position    int  `json:"position"`
	Pinned      bool `json:"pinned"`
	Replacement ID   `json:"replacement_asset_id,omitempty"`
}

type AssetPackage struct {
	ID          ID            `json:"id"`
	Name        string        `json:"name"`
	Items       []PackageItem `json:"items"`
	Fallback    bool          `json:"fallback"`
	Environment string        `json:"environment"`
	Version     int64         `json:"version"`
}

func NewAssetPackage(name, environment string, fallback bool) (AssetPackage, error) {
	p := AssetPackage{ID: NewID("pkg"), Name: strings.TrimSpace(name), Environment: environment, Fallback: fallback, Version: 1}
	if p.Name == "" || !validEnvironment(environment) {
		return AssetPackage{}, fmt.Errorf("package name and valid environment are required")
	}
	return p, nil
}

func (p *AssetPackage) Add(assetID, replacement ID, pinned bool) error {
	if assetID.Empty() || assetID == replacement {
		return ErrInvalidReference
	}
	for _, item := range p.Items {
		if item.AssetID == assetID {
			return fmt.Errorf("%w: asset already in package", ErrConflict)
		}
	}
	p.Items = append(p.Items, PackageItem{AssetID: assetID, Replacement: replacement, Pinned: pinned, Position: len(p.Items) + 1})
	p.normalize()
	p.Version++
	return nil
}

func (p *AssetPackage) Reorder(assetIDs []ID) error {
	if len(assetIDs) != len(p.Items) {
		return ErrInvalidReference
	}
	byID := make(map[ID]PackageItem, len(p.Items))
	for _, item := range p.Items {
		byID[item.AssetID] = item
	}
	next := make([]PackageItem, 0, len(p.Items))
	for index, assetID := range assetIDs {
		item, ok := byID[assetID]
		if !ok {
			return ErrInvalidReference
		}
		delete(byID, assetID)
		item.Position = index + 1
		next = append(next, item)
	}
	if len(byID) != 0 {
		return ErrInvalidReference
	}
	p.Items = next
	p.normalize()
	p.Version++
	return nil
}

func (p *AssetPackage) normalize() {
	sort.SliceStable(p.Items, func(i, j int) bool {
		if p.Items[i].Pinned != p.Items[j].Pinned {
			return p.Items[i].Pinned
		}
		return p.Items[i].Position < p.Items[j].Position
	})
	for i := range p.Items {
		p.Items[i].Position = i + 1
	}
}

func validEnvironment(value string) bool {
	return value == "development" || value == "staging" || value == "production"
}
