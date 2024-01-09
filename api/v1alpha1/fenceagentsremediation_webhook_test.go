package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("FenceAgentsRemediation Validation", func() {

	Context("creating FenceAgentsRemediation", func() {

		When("agent name match format and binary", func() {
			It("should be accepted", func() {
				far := getTestFAR(validAgentName)
				_, err := far.ValidateCreate()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		When("agent name was not found ", func() {
			It("should be rejected", func() {
				far := getTestFAR(invalidAgentName)
				_, err := far.ValidateCreate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Unsupported fence agent: %s", invalidAgentName))
			})
		})
	})

	Context("updating FenceAgentsRemediation", func() {

		oldFAR := getTestFAR(invalidAgentName)
		When("agent name match format and binary", func() {
			It("should be accepted", func() {
				far := getTestFAR(validAgentName)
				_, err := far.ValidateUpdate(oldFAR)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		When("agent name was not found ", func() {
			It("should be rejected", func() {
				far := getTestFAR(invalidAgentName)
				_, err := far.ValidateUpdate(oldFAR)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Unsupported fence agent: %s", invalidAgentName))
			})
		})
	})
})

func getTestFAR(agentName string) *FenceAgentsRemediation {
	return &FenceAgentsRemediation{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-" + agentName,
		},
		Spec: FenceAgentsRemediationSpec{
			Agent: agentName,
		},
	}
}
