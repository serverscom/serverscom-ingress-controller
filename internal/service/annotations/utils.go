package annotations

import (
	"context"
	"fmt"
	"strings"

	serverscom "github.com/serverscom/serverscom-go-client/pkg"
)

// GetRegionIDByCode returns ID of cloud region by region code
func GetRegionIDByCode(ctx context.Context, regionsService serverscom.CloudComputingRegionsService, code string) (int, error) {
	regions, err := regionsService.Collection().Collect(ctx)
	if err != nil {
		return 0, err
	}
	for _, r := range regions {
		if strings.EqualFold(r.Code, code) {
			return int(r.ID), nil
		}
	}
	return 0, fmt.Errorf("cloud region with code '%s' not found", code)
}
