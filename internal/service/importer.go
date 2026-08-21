package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/wyw14/cry-075/internal/application"
	"github.com/wyw14/cry-075/internal/domain"
)

type AssetImporter struct{ Catalog application.CatalogService }
type ImportResult struct {
	Created  []domain.ID `json:"created"`
	Rejected []string    `json:"rejected"`
}

func (i AssetImporter) Import(ctx context.Context, reader io.Reader, actor domain.Actor, requestID string) (ImportResult, error) {
	rows, err := csv.NewReader(reader).ReadAll()
	if err != nil {
		return ImportResult{}, err
	}
	if len(rows) == 0 {
		return ImportResult{}, fmt.Errorf("empty import")
	}
	header := strings.Join(rows[0], ",")
	if header != "name,media_type,storage_key,rights_ends_at" {
		return ImportResult{}, fmt.Errorf("unexpected import columns")
	}
	result := ImportResult{Created: []domain.ID{}, Rejected: []string{}}
	for line, row := range rows[1:] {
		if len(row) != 4 {
			result.Rejected = append(result.Rejected, fmt.Sprintf("line %d: field count", line+2))
			continue
		}
		var rights *time.Time
		if row[3] != "" {
			value, err := time.Parse(time.RFC3339, row[3])
			if err != nil {
				result.Rejected = append(result.Rejected, fmt.Sprintf("line %d: rights time", line+2))
				continue
			}
			rights = &value
		}
		asset, err := i.Catalog.CreateAsset(ctx, row[0], row[1], row[2], rights, actor, requestID)
		if err != nil {
			result.Rejected = append(result.Rejected, fmt.Sprintf("line %d: %v", line+2, err))
			continue
		}
		result.Created = append(result.Created, asset.ID)
	}
	return result, nil
}
