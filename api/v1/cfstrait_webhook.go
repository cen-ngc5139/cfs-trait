/*


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

package v1

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var cfstraitlog = logf.Log.WithName("cfstrait-resource")

func (r *CfsTrait) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-trait-ghostbaby-com-v1-cfstrait,mutating=true,failurePolicy=fail,groups=trait.ghostbaby.com,resources=cfstraits,verbs=create;update,versions=v1,name=mcfstrait.kb.io,sideEffects=none,admissionReviewVersions=["v1"]

var _ webhook.Defaulter = &CfsTrait{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *CfsTrait) Default() {
	cfstraitlog.Info("default", "name", r.Name)
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-trait-ghostbaby-com-v1-cfstrait,mutating=false,failurePolicy=fail,groups=trait.ghostbaby.com,resources=cfstraits,versions=v1,name=vcfstrait.kb.io,sideEffects=none,admissionReviewVersions=["v1"]

var _ webhook.Validator = &CfsTrait{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *CfsTrait) ValidateCreate() error {
	cfstraitlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return r.validateCfsTraitSpec()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *CfsTrait) ValidateUpdate(old runtime.Object) error {
	cfstraitlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return r.validateCfsTraitSpec()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *CfsTrait) ValidateDelete() error {
	cfstraitlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *CfsTrait) validateCfsTraitSpec() error {
	var allErrs field.ErrorList

	if (!r.Spec.IsAllPods && len(r.Spec.Pods) == 0) ||
		(r.Spec.IsAllPods && len(r.Spec.Pods) > 0) {
		err := field.Invalid(
			field.NewPath("spec").Child("isAllPods"),
			r.Name, "spec.isAllPods and spec.pods can only configure one of the parameters.")
		allErrs = append(allErrs, err)
	}

	if r.Spec.Period < 10000 || r.Spec.Quota < 10000 {
		err := field.Invalid(
			field.NewPath("spec").Child("Period"),
			r.Name, "spec.Period and spec.Quota value cannot < 10000 (10ms).")
		allErrs = append(allErrs, err)
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "trait.ghostbaby.com", Kind: "CfsTrait"},
		r.Name, allErrs)
}
