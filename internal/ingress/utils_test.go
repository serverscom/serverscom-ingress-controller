package ingress

import (
	"testing"

	. "github.com/onsi/gomega"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsScIngress(t *testing.T) {
	g := NewWithT(t)

	defaultClass := DefaultScIngressClass
	otherClass := "other-class"

	g.Expect(IsScIngress(&v1.Ingress{}, "")).To(BeFalse())
	g.Expect(IsScIngress(&v1.Ingress{}, defaultClass)).To(BeFalse())

	ingress := &v1.Ingress{
		Spec: v1.IngressSpec{
			IngressClassName: &defaultClass,
		},
	}
	g.Expect(IsScIngress(ingress, defaultClass)).To(BeTrue())
	g.Expect(IsScIngress(ingress, otherClass)).To(BeFalse())

	ingress.Spec.IngressClassName = &otherClass
	g.Expect(IsScIngress(ingress, otherClass)).To(BeTrue())

	ingressClassAnnotation := &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				IngressClassKey: defaultClass,
			},
		},
	}
	g.Expect(IsScIngress(ingressClassAnnotation, defaultClass)).To(BeTrue())
	g.Expect(IsScIngress(ingressClassAnnotation, otherClass)).To(BeFalse())

	ingressClassAnnotation.Annotations[IngressClassKey] = otherClass
	g.Expect(IsScIngress(ingressClassAnnotation, defaultClass)).To(BeFalse())
}
