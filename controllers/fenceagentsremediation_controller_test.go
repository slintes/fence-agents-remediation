/*
Copyright 2023.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/medik8s/fence-agents-remediation/api/v1alpha1"
	"github.com/medik8s/fence-agents-remediation/pkg/cli"
)

const (
	defaultNamespace = "default"
)

var (
	executedCommand []string
)

var _ = Describe("FAR Controller", func() {
	var (
		underTestFAR *v1alpha1.FenceAgentsRemediation
	)

	Context("Defaults", func() {
		BeforeEach(func() {
			underTestFAR = &v1alpha1.FenceAgentsRemediation{
				ObjectMeta: metav1.ObjectMeta{Name: "test-far", Namespace: defaultNamespace},
			}
			err := k8sClient.Create(context.Background(), underTestFAR)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err := k8sClient.Delete(context.Background(), underTestFAR)
			Expect(err).NotTo(HaveOccurred())
		})

		When("creating a resource", func() {
			It("should have a namespace scoped CR", func() {
				Expect(underTestFAR.Namespace).To(Not(BeEmpty()))
			})
		})
	})

	Context("Reconcilie", func() {
		//Scenarios
		nodeName := "master-0"
		testFields := []string{"--username", "--password", "--action", "--ip", "--lanplus", "--ipport"}
		testValues := []string{"adminn", "password", "reboot", "192.168.111.1", ""}
		nodeFeilds := []string{"master-0", "master-1", "master-2", "worker-0", "worker-1", "worker-2"}
		nodeValues := []string{"6230", "6231", "6232", "6233", "6234", "6235"}
		testShareParam, testNodeParam := buildFARTemplate(testFields, testValues, nodeFeilds, nodeValues, nodeName)
		templateTest := newFenceAgentsRemediationTemplate(" ", testShareParam, testNodeParam)
		test := &v1alpha1.FenceAgentsRemediation{
			ObjectMeta: metav1.ObjectMeta{Name: nodeName, Namespace: defaultNamespace},
		}
		fenceAgentsPod := buildFarPod()

		JustBeforeEach(func() {
			err := k8sClient.Create(context.Background(), test)
			Expect(err).NotTo(HaveOccurred())
			Expect(test.Namespace).To(Not(BeEmpty()))
		})
		BeforeEach(func() {
			// Create fenceAgentsPod and FAR Template
			Expect(k8sClient.Create(context.Background(), fenceAgentsPod)).NotTo(HaveOccurred())
			Expect(k8sClient.Create(context.Background(), templateTest)).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(context.Background(), fenceAgentsPod)).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(context.Background(), templateTest)).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(context.Background(), test)).NotTo(HaveOccurred())
		})

		When("creating FAR CR", func() {
			It("should build the exec command based on Template and Remediation CRs", func() {
				Eventually(func() bool {
					return cliCommandsEquality(templateTest, test)
				}, 1*time.Second, 500*time.Millisecond).Should(BeTrue())
			})
		})
	})
})

// buildFARTemplate from string to arrays to two string maps (key-value manner)
func buildFARTemplate(fields []string, values []string, nodeFields []string, nodeValues []string, node string) (map[v1alpha1.ParameterName]string, map[v1alpha1.ParameterName]map[v1alpha1.NodeName]string) {
	testShareParam := make(map[v1alpha1.ParameterName]string)
	testNodeParam := make(map[v1alpha1.ParameterName]map[v1alpha1.NodeName]string)
	i := 0
	for i = 0; i < len(values); i++ {
		field := v1alpha1.ParameterName(fields[i])
		testShareParam[field] = values[i]
	}

	nodeName := v1alpha1.NodeName(node)
	numNodeParam := len(fields) - len(values)
	for j := 0; j < numNodeParam; j++ {
		if indexOf(node, nodeFields) > -1 {
			field := v1alpha1.ParameterName(fields[i+j])
			testNodeParam[field] = make(map[v1alpha1.NodeName]string)
			testNodeParam[field][nodeName] = nodeValues[indexOf(node, nodeFields)]
		}
	}
	return testShareParam, testNodeParam
}

// indexOf return the index of element in data array. If it is not found, return -1
func indexOf(element string, data []string) int {
	for k, v := range data {
		if element == v {
			return k
		}
	}
	return -1
}

// newFenceAgentsRemediationTemplate assign the input to the FenceAgentsRemediationTemplate's Spec
func newFenceAgentsRemediationTemplate(agent string, sharedparameters map[v1alpha1.ParameterName]string, nodeparameters map[v1alpha1.ParameterName]map[v1alpha1.NodeName]string) *v1alpha1.FenceAgentsRemediationTemplate {
	return &v1alpha1.FenceAgentsRemediationTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: fenceAgentsTemplateName, Namespace: defaultNamespace},
		Spec: v1alpha1.FenceAgentsRemediationTemplateSpec{
			Agent:            agent,
			SharedParameters: sharedparameters,
			NodeParameters:   nodeparameters,
		},
	}
}

// buildFarPod builds a dummy pod with FAR label and namespace
func buildFarPod() *corev1.Pod {
	fenceAgentsPod := &corev1.Pod{}
	fenceAgentsPod.Labels = faPodLabels
	fenceAgentsPod.Name = "mock-fence-agents"
	fenceAgentsPod.Namespace = defaultNamespace
	container := corev1.Container{
		Name:  "foo",
		Image: "foo",
	}
	fenceAgentsPod.Spec.Containers = []corev1.Container{container}
	return fenceAgentsPod
}

// cliCommandsEquality creates the command for CLI and compares it with the production command
func cliCommandsEquality(farTemplate *v1alpha1.FenceAgentsRemediationTemplate, far *v1alpha1.FenceAgentsRemediation) bool {
	//fence_ipmilan --ip=192.168.111.1 --ipport=6233 --username=admin --password=password --action=status --lanplus
	command, err := buildFenceAgentParams(farTemplate, far)
	Expect(err).NotTo(HaveOccurred())
	command = append([]string{farTemplate.Spec.Agent}, command...)
	Expect(executedCommand).ToNot(Equal(nil))
	fmt.Printf("%s is the executedCommand in prod, and %s is the expected command in test.\n", executedCommand, command)
	return contains(executedCommand, command) && contains(command, executedCommand)
}

// contains check if all the elements from s1 are in s2
func contains(s1 []string, s2 []string) bool {
	hits := 0
	for _, element1 := range s1 {
		for _, element2 := range s2 {
			if element1 == element2 {
				hits++
				continue
			}
		}
	}
	// Check that hits equal to the number of elements in s1 and the number of elements in s2, and thus they have the same size
	if hits == len(s2) && hits == len(s1) {
		return true
	} else {
		return false
	}
}

// Implements Execute function to mock/test Execute of FenceAgentsRemediationReconciler
type mockExecuter struct {
	expected []string
}

// mockNewExecuter is a dummy function for testing
func newMockExecuter() cli.Executer {
	mockExpected := []string{"mockExecuter"}
	mockE := mockExecuter{expected: mockExpected}
	return &mockE
}

// Execute is a dummy function for testing which stores the production command in the global variable
func (e *mockExecuter) Execute(_ *corev1.Pod, command []string) (stdout string, stderr string, err error) {
	//store the executed command for testing its validaty
	executedCommand = command
	// e.expected = command
	return "", "", nil
}
