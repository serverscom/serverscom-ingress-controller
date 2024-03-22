package annotations

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestGetRegionIDByCode(t *testing.T) {
	t.Run("Region not found", func(t *testing.T) {
		g := NewWithT(t)
		id, found := GetStorageRegionIDByCode("notexist")
		g.Expect(found).To(BeFalse())
		g.Expect(id).To(BeEquivalentTo(0))
	})

	t.Run("Region found", func(t *testing.T) {
		g := NewWithT(t)
		id, found := GetStorageRegionIDByCode("US01")
		g.Expect(found).To(BeTrue())
		g.Expect(id).To(BeEquivalentTo(1))
	})
}
