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

package v1alpha1

import (
	"fmt"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/medik8s/fence-agents-remediation/pkg/validation"
)

var (
	// webhookFARLog is for logging in this package.
	webhookFARLog = logf.Log.WithName("fenceagentsremediation-resource")
	// verify agent existence with os.Stat function
	agentValidator = validation.NewAgentValidator()
)

func (r *FenceAgentsRemediation) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:path=/validate-fence-agents-remediation-medik8s-io-v1alpha1-fenceagentsremediation,mutating=false,failurePolicy=fail,sideEffects=None,groups=fence-agents-remediation.medik8s.io,resources=fenceagentsremediations,verbs=create;update,versions=v1alpha1,name=vfenceagentsremediation.kb.io,admissionReviewVersions=v1
var _ webhook.Validator = &FenceAgentsRemediation{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (far *FenceAgentsRemediation) ValidateCreate() (admission.Warnings, error) {
	webhookFARLog.Info("validate create", "name", far.Name)
	return validateAgentName(far.Spec.Agent)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (far *FenceAgentsRemediation) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	webhookFARLog.Info("validate update", "name", far.Name)
	return validateAgentName(far.Spec.Agent)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (far *FenceAgentsRemediation) ValidateDelete() (admission.Warnings, error) {
	webhookFARLog.Info("validate delete", "name", far.Name)
	return nil, nil
}

func validateAgentName(agent string) (admission.Warnings, error) {
	exists, err := agentValidator.ValidateAgentName(agent)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to validate fence agent: %s. You might want to try again.", agent)
	}
	if !exists {
		return nil, fmt.Errorf("Unsupported fence agent: %s", agent)
	}
	return nil, nil
}
