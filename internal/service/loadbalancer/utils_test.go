package loadbalancer

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestGetLoadBalancerName(t *testing.T) {
	g := NewWithT(t)

	shortUID := types.UID("1234567890abcdef")
	ing := &v1.Ingress{ObjectMeta: metav1.ObjectMeta{UID: shortUID}}
	expectedNameShort := fmt.Sprintf("ingress-a%s", shortUID)

	nameShort := GetLoadBalancerName(ing)
	g.Expect(nameShort).To(Equal(expectedNameShort))

	longUID := types.UID("1234567890abcdef1234567890abcdef1234567890abcdef")
	ing = &v1.Ingress{ObjectMeta: metav1.ObjectMeta{UID: longUID}}
	expectedNameLong := fmt.Sprintf("ingress-a%s", longUID[:31])

	nameLong := GetLoadBalancerName(ing)
	g.Expect(nameLong).To(Equal(expectedNameLong))

	uidWithDashes := types.UID("1234-5678-90ab-cdef")
	ing = &v1.Ingress{ObjectMeta: metav1.ObjectMeta{UID: uidWithDashes}}
	expectedNameDashes := "ingress-a1234567890abcdef"

	nameDashes := GetLoadBalancerName(ing)
	g.Expect(nameDashes).To(Equal(expectedNameDashes))
}
