package annotations

import (
	"strings"
)

var (
	regionsIDs = map[string]int{
		"NL01":  0,
		"US01":  1,
		"LU01":  2,
		"MO01":  3,
		"SIN01": 4,
		"MOW2":  5,
	}
)

// GetStorageRegionIDByCode returns ID of cloud storage region by region code
func GetStorageRegionIDByCode(code string) (int, bool) {
	id, ok := regionsIDs[strings.ToUpper(code)]
	return id, ok
}
