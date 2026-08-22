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
	current, err := p.indexItems()
	if err != nil {
		return err
	}
	if err := validateRequestedOrder(assetIDs, current); err != nil {
		return err
	}

	ordered := make([]PackageItem, 0, len(assetIDs))
	for position, assetID := range assetIDs {
		original := current[assetID]
		original.Position = position + 1
		ordered = append(ordered, original)
	}
	p.Items = normalizePackageOrder(ordered)
	p.Version++
	return nil
}

func (p AssetPackage) indexItems() (map[ID]PackageItem, error) {
	indexed := make(map[ID]PackageItem, len(p.Items))
	for _, item := range p.Items {
		if item.AssetID.Empty() {
			return nil, ErrInvalidReference
		}
		if _, exists := indexed[item.AssetID]; exists {
			return nil, fmt.Errorf("%w: duplicate package asset", ErrConflict)
		}
		indexed[item.AssetID] = item
	}
	return indexed, nil
}

func validateRequestedOrder(assetIDs []ID, current map[ID]PackageItem) error {
	if len(assetIDs) != len(current) {
		return ErrInvalidReference
	}
	seen := make(map[ID]struct{}, len(assetIDs))
	for _, assetID := range assetIDs {
		if assetID.Empty() {
			return ErrInvalidReference
		}
		if _, duplicate := seen[assetID]; duplicate {
			return fmt.Errorf("%w: duplicate requested asset", ErrConflict)
		}
		if _, exists := current[assetID]; !exists {
			return ErrInvalidReference
		}
		seen[assetID] = struct{}{}
	}
	return nil
}

func normalizePackageOrder(items []PackageItem) []PackageItem {
	result := append([]PackageItem(nil), items...)
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Pinned != result[j].Pinned {
			return result[i].Pinned
		}
		return result[i].Position < result[j].Position
	})
	for index := range result {
		result[index].Position = index + 1
	}
	return result
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
