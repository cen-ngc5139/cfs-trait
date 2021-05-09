package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/types"

	traitv1 "github.com/ghostbaby/cfs-trait/api/v1"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/rs/xid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// UnpackRevisionData will unpack revision.Data to Component
func UnpackRevisionData(rev *appsv1.ControllerRevision) (*traitv1.CfsTrait, error) {
	var err error
	if rev.Data.Object != nil {
		comp, ok := rev.Data.Object.(*traitv1.CfsTrait)
		if !ok {
			return nil, fmt.Errorf("invalid type of revision %s, type should not be %v", rev.Name, reflect.TypeOf(rev.Data.Object))
		}
		return comp, nil
	}
	var comp traitv1.CfsTrait
	err = json.Unmarshal(rev.Data.Raw, &comp)
	return &comp, err
}

//IsRevisionDiff check whether there's any different between two component revision
func (r *CfsTraitReconciler) IsRevisionDiff(mt metav1.Object, curCfsTrait *traitv1.CfsTrait) (bool, int64) {
	if curCfsTrait.Status.LatestRevision == nil {
		return true, 0
	}

	var oldRev appsv1.ControllerRevision

	namespaceName := types.NamespacedName{
		Name:      curCfsTrait.Status.LatestRevision.Name,
		Namespace: mt.GetNamespace(),
	}

	err := r.Client.Get(context.Background(), namespaceName, &oldRev)
	if err != nil {
		r.Log.Info(fmt.Sprintf("get old controllerRevision %s error %v, will create new revision", curCfsTrait.Status.LatestRevision.Name, err), "componentName", mt.GetName())
		return true, curCfsTrait.Status.LatestRevision.Revision
	}
	oldComp, err := UnpackRevisionData(&oldRev)
	if err != nil {
		r.Log.Info(fmt.Sprintf("Unmarshal old controllerRevision %s error %v, will create new revision", curCfsTrait.Status.LatestRevision.Name, err), "componentName", mt.GetName())
		return true, oldRev.Revision
	}

	if reflect.DeepEqual(curCfsTrait.Spec, oldComp.Spec) {
		return false, oldRev.Revision
	}
	return true, oldRev.Revision
}

func (r *CfsTraitReconciler) createControllerRevision(mt metav1.Object, obj runtime.Object) bool {
	curComp := obj.(*traitv1.CfsTrait)
	diff, curRevision := r.IsRevisionDiff(mt, curComp)
	if !diff {
		// No difference, no need to create new revision.
		return false
	}
	nextRevision := curRevision + 1
	revisionName := ConstructRevisionName(mt.GetName())

	curComp.Status.LatestRevision = &traitv1.Revision{
		Name:     revisionName,
		Revision: nextRevision,
	}
	// set annotation to component
	revision := appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name: revisionName,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: v1alpha2.SchemeGroupVersion.String(),
					Kind:       v1alpha2.ComponentKind,
					Name:       curComp.Name,
					UID:        curComp.UID,
					Controller: newTrue(),
				},
			},
		},
		Revision: nextRevision,
		Data:     runtime.RawExtension{Object: curComp},
	}

	err := r.Client.Create(context.Background(), &revision)
	if err != nil {
		r.Log.Info(fmt.Sprintf("error create controllerRevision %v", err), "componentName", mt.GetName())
		return false
	}
	err = r.Client.Status().Update(context.Background(), curComp)
	if err != nil {
		r.Log.Info(fmt.Sprintf("update component status latestRevision %s err %v", revisionName, err), "componentName", mt.GetName())
		return false
	}
	r.Log.Info(fmt.Sprintf("ControllerRevision %s created", revisionName))
	return true
}

// ConstructRevisionName will generate revisionName from componentName
// hash suffix char set added to componentName is (0-9, a-v)
func ConstructRevisionName(componentName string) string {
	return strings.Join([]string{componentName, xid.NewWithTime(time.Now()).String()}, "-")
}

func newTrue() *bool {
	b := true
	return &b
}
