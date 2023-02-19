/*
Copyright 2022.

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
	"errors"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/medik8s/fence-agents-remediation/api/v1alpha1"
	"github.com/medik8s/fence-agents-remediation/pkg/cli"

	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	//TODO mshitrit verify that template is created with this name
	fenceAgentsTemplateName = "fenceagentsremediationtemplate-default"
)

var (
	faPodLabels = map[string]string{"app": "fence-agents-remediation-operator"}
)

// FenceAgentsRemediationReconciler reconciles a FenceAgentsRemediation object
type FenceAgentsRemediationReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Executor cli.Executer
}

// SetupWithManager sets up the controller with the Manager.
func (r *FenceAgentsRemediationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.FenceAgentsRemediation{}).
		Complete(r)
}

//+kubebuilder:rbac:groups=core,resources=pods/exec,verbs=create
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;update;delete;deletecollection
//+kubebuilder:rbac:groups=fence-agents-remediation.medik8s.io,resources=fenceagentsremediationtemplates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=fence-agents-remediation.medik8s.io,resources=fenceagentsremediations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=fence-agents-remediation.medik8s.io,resources=fenceagentsremediations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=fence-agents-remediation.medik8s.io,resources=fenceagentsremediations/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FenceAgentsRemediation object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *FenceAgentsRemediationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("Begin FenceAgentsRemediation Reconcile")

	// Fetch the FenceAgentsRemediationTemplate instance
	emptyResult := ctrl.Result{}
	key := client.ObjectKey{Namespace: req.Namespace, Name: fenceAgentsTemplateName}
	farTemplate := &v1alpha1.FenceAgentsRemediationTemplate{}
	if err := r.Client.Get(ctx, key, farTemplate); err != nil {
		if apiErrors.IsNotFound(err) {
			// FAR Template is not found, stop reconciling
			r.Log.Info("FAR Template CR is not found - nothing to do", "CR Name", req.Name, "CR Namespace", req.Namespace)
			return emptyResult, nil
		}
		r.Log.Error(err, "failed to get FAR Template CR")
		return emptyResult, err
	}

	// Fetch the FenceAgentsRemediation instance
	r.Log.Info("Fetch FAR CR")
	far := &v1alpha1.FenceAgentsRemediation{}
	if err := r.Client.Get(ctx, req.NamespacedName, far); err != nil {
		if apiErrors.IsNotFound(err) {
			// FAR is deleted, stop reconciling
			r.Log.Info("FAR CR is deleted - nothing to do", "CR Name", req.Name, "CR Namespace", req.Namespace)
			return emptyResult, nil
		}
		r.Log.Error(err, "failed to get FAR CR")
		return emptyResult, err
	}
	// TODO: Validate FAR CR name to nodeName. Run isNodeNameValid
	// Fetch the FAR's pod
	r.Log.Info("Fetch FAR's pod")
	pod, err := r.getFenceAgentsPod(req.Namespace)
	if err != nil {
		return emptyResult, err
	}

	//TODO: Check that FA is excutable? run cli.IsExecuteable

	r.Log.Info("Create and execute the fence agent", "Fence Agent", farTemplate.Spec.Agent)
	faParams, err := buildFenceAgentParams(farTemplate, far)
	if err != nil {
		return emptyResult, err
	}
	cmd := append([]string{farTemplate.Spec.Agent}, faParams...)
	// The Fence Agent is excutable and the parameters are valid but we don't know about their values
	if _, _, err := r.Executor.Execute(pod, cmd); err != nil {
		//TODO: better seperation between errors from wrong shared parameters values and wrong node parameters values
		return emptyResult, err
	}
	r.Log.Info("Finish FenceAgentsRemediation Reconcile")
	return emptyResult, nil
}

// getFenceAgentsPod fetches the FAR pod based on FAR's label and namespace
func (r *FenceAgentsRemediationReconciler) getFenceAgentsPod(namespace string) (*corev1.Pod, error) {

	pods := new(corev1.PodList)

	podLabelsSelector, _ := metav1.LabelSelectorAsSelector(
		&metav1.LabelSelector{MatchLabels: faPodLabels})
	options := client.ListOptions{
		LabelSelector: podLabelsSelector,
		Namespace:     namespace,
	}
	if err := r.Client.List(context.Background(), pods, &options); err != nil {
		r.Log.Error(err, "failed fetching Fence Agent layer pod")
		return nil, err
	}
	if len(pods.Items) == 0 {
		r.Log.Info("No Fence Agent pods were found")
		podNotFoundErr := &apiErrors.StatusError{ErrStatus: metav1.Status{
			Status: metav1.StatusFailure,
			Code:   http.StatusNotFound,
			Reason: metav1.StatusReasonNotFound,
		}}
		return nil, podNotFoundErr
	}
	return &pods.Items[0], nil
}

// buildFenceAgentParams collects the FAR's parameters for the node based on the FARTemplate and FAR CRs
func buildFenceAgentParams(farTemplate *v1alpha1.FenceAgentsRemediationTemplate, far *v1alpha1.FenceAgentsRemediation) ([]string, error) {
	var fenceAgentParams []string
	for paramName, paramVal := range farTemplate.Spec.SharedParameters {
		fenceAgentParams = appendParamToSlice(fenceAgentParams, string(paramName), paramVal)
	}

	nodeName := v1alpha1.NodeName(far.Name)
	for paramName, nodeMap := range farTemplate.Spec.NodeParameters {
		if nodeMap[nodeName] != "" {
			fenceAgentParams = appendParamToSlice(fenceAgentParams, string(paramName), nodeMap[nodeName])
		} else {
			err := errors.New("node parameter is required, and cannot be empty")
			return nil, err
		}
	}
	return fenceAgentParams, nil
}

// appendParamToSlice appends parameters in a key-value manner, when value can be empty
func appendParamToSlice(fenceAgentParams []string, paramName string, paramVal string) []string {
	if paramVal != "" {
		fenceAgentParams = append(fenceAgentParams, fmt.Sprintf("%s=%s", paramName, paramVal))
	} else {
		fenceAgentParams = append(fenceAgentParams, paramName)
	}
	return fenceAgentParams
}

// TODO: Add isNodeNameValid function which call listNodeNames to validate the FAR's name with the cluster node names
