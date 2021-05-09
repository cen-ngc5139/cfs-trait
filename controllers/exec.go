package controllers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ghostbaby/cfs-trait/controllers/models"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ghostbaby/cfs-trait/controllers/base"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"

	"k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/controller-runtime/pkg/client"

	traitv1 "github.com/ghostbaby/cfs-trait/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

// CallBrokerToExec is exec to rewrite cfs period and quota
// - get pod list with filter
// - get node info which the filtered pod running
// - call cfs-broker to rewrite cfs config.
func (r *CfsTraitReconciler) CallBrokerToExec(cfs *traitv1.CfsTrait, req ctrl.Request) error {
	var (
		nodes             []string
		err               error
		count             int64
		nodeConditionList []traitv1.NodeCondition
	)
	// get pod list from deployment
	if cfs.Spec.IsAllPods {
		r.Log.Info("start to get node list with label selector",
			"namespace", req.Namespace, "name", req.Name)
		nodes, err = r.FilterAllPodList(cfs)
		if err != nil {
			r.Log.Error(err, "fail to get node list with label selector",
				"namespace", req.Namespace, "name", req.Name)
			return err
		}
		r.Log.Info("success to get node list with label selector",
			"namespace", req.Namespace, "name", req.Name)
	}

	// get pod list from spec.pods
	if len(cfs.Spec.Pods) > 0 {
		r.Log.Info("start to get node list with spec.pods",
			"namespace", req.Namespace, "name", req.Name)
		nodes, err = r.FilterPodList(cfs)
		if err != nil {
			r.Log.Error(err, "fail to get node list from spec.Pods",
				"namespace", req.Namespace, "name", req.Name)
			return err
		}
		r.Log.Info("success to get node list with spec.pods",
			"namespace", req.Namespace, "name", req.Name)
	}

	if len(nodes) == 0 {
		r.Log.Info("not found any node for pods",
			"namespace", req.Namespace, "name", req.Name)
		return nil
	}

	nodes = Unique(nodes)

	chJobs := make(chan *Jobs, len(nodes))
	chResults := make(chan *Results, len(nodes))
	cfs.Status.Nodes = int64(len(nodes))

	for w := 1; w <= DefaultThreadPoolSize; w++ {
		go ExecWork(cfs, chJobs, chResults, r.Log, cfs.Spec.Force)
	}

	for _, n := range nodes {
		r.Log.V(4).Info("start to post cfs config command",
			"namespace", req.Namespace, "name", req.Name, "node", n)
		broker := r.getCfsBrokerPodIP(n)

		cli := base.BaseClient{
			HTTP:      &http.Client{},
			Endpoint:  getCfsBrokerURL(broker),
			Transport: &http.Transport{},
		}

		job := &Jobs{
			Client:     cli,
			Node:       n,
			RetryTimes: DefaultRetryTimes,
		}
		chJobs <- job
	}
	close(chJobs)

	for i := 1; i <= len(nodes); i++ {
		res := <-chResults
		if res.Resp != nil {
			nodeCondition := traitv1.NodeCondition{
				NodeName:           res.Node,
				Status:             corev1.ConditionFalse,
				LastUpdateTime:     metav1.Now(),
				LastTransitionTime: metav1.Now(),
				Reason:             res.Resp.Error(),
			}
			nodeConditionList = append(nodeConditionList, nodeCondition)
			err = res.Resp
			//r.Log.Error(err, "fail to post cfs config command",
			//	"namespace", req.Namespace, "name", req.Name, "node", res.Node)
		}
		count++

		r.Log.V(4).Info("success to post cfs config command",
			"namespace", req.Namespace, "name", req.Name, "node", res.Node)
	}

	cfs.Status.UpdatedNodes = count
	cfs.Status.Conditions = nodeConditionList

	defer func() {

		if !isNeedUpdateStatus(cfs) {
			return
		}

		updateObservedGeneration(cfs)
		if err := r.writeStatus(cfs); err != nil {
			r.Log.Error(err, "fail to write cfs status", "namespace", req.Namespace, "name", req.Name)
			return
		}
		r.Log.Info(fmt.Sprintf("success to update CfsTrait status, ObservedGeneration: %d", cfs.Status.ObservedGeneration),
			"namespace", req.Namespace, "name", req.Name)
	}()

	return err

}

func getCfsBrokerURL(ip string) string {
	return fmt.Sprintf("http://%s", ip)
}

func (r *CfsTraitReconciler) getCfsBrokerPodIP(name string) (podIP string) {

	label := map[string]string{
		"app": "cfs-broker",
	}

	fieldSelector := fields.SelectorFromSet(fields.Set{"spec.nodeName": name})

	labelSelector := labels.SelectorFromSet(label)

	list := &corev1.PodList{}

	if err := r.Client.List(context.TODO(), list, &client.ListOptions{FieldSelector: fieldSelector, LabelSelector: labelSelector}); err != nil {
		if !errors.IsForbidden(err) {
			return podIP
		}
	}
	for _, pod := range list.Items {
		podIP = pod.Status.PodIP
	}

	return podIP

}

// FilterPodList is to get node list from deployment all pods.
func (r *CfsTraitReconciler) FilterAllPodList(cfs *traitv1.CfsTrait) ([]string, error) {
	var nodes []string

	app := cfs.Spec.AppName
	ns := cfs.Spec.Namespace
	labelKey := cfs.Spec.LabelKey

	label := map[string]string{
		labelKey: app,
	}

	opts := &client.ListOptions{}
	set := labels.SelectorFromSet(label)
	opts.LabelSelector = set

	pods := &corev1.PodList{}
	if err := r.Client.List(context.TODO(), pods, opts); err != nil {
		r.Log.Error(err, "fail to get pods.", "namespace", ns, "name", app)
		return nil, err
	}

	for _, pod := range pods.Items {
		nodes = append(nodes, GetNodeName(&pod))
	}

	return nodes, nil
}

// FilterPodList is to get node list from spec.Pods .
func (r *CfsTraitReconciler) FilterPodList(cfs *traitv1.CfsTrait) ([]string, error) {
	var nodes []string

	if len(cfs.Spec.Pods) == 0 {
		return nil, nil
	}

	ns := cfs.Spec.Namespace

	for _, podName := range cfs.Spec.Pods {
		var pod corev1.Pod
		nsName := types.NamespacedName{
			Namespace: ns,
			Name:      podName,
		}

		if err := r.Client.Get(context.TODO(), nsName, &pod); err != nil {
			r.Log.Error(err, "fail to get pods", "namespace", ns, "name", nsName)
			return nil, err
		}

		nodes = append(nodes, GetNodeName(&pod))
	}

	return nodes, nil

}

// GetNodeName is get pod assigned node name.
func GetNodeName(pod *corev1.Pod) string {
	return pod.Spec.NodeName
}

func (r *CfsTraitReconciler) writeStatus(cfs *traitv1.CfsTrait) error {
	err := r.Client.Status().Update(r.CTX, cfs)
	if err != nil {
		// may be it's k8s v1.10 and erlier (e.g. oc3.9) that doesn't support status updates
		// so try to update whole CR
		err := r.Client.Update(r.CTX, cfs)
		if err != nil {
			return fmt.Errorf("fail to send update, %v", err)
		}
	}

	return nil
}

// Unique remove duplicate values from Slice
func Unique(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func (r *CfsTraitReconciler) IsPodCfsPolicyExist(name, ns string) (bool, types.NamespacedName, error) {

	var (
		isExist        bool
		namespacedName types.NamespacedName
	)

	label := map[string]string{
		models.DefaultAppLabelKey: name,
	}
	labelSelector := labels.SelectorFromSet(label)

	list := &traitv1.CfsTraitList{}
	if err := r.Client.List(context.TODO(), list, &client.ListOptions{LabelSelector: labelSelector}); err != nil {
		if !errors.IsForbidden(err) {
			return false, namespacedName, err
		}
	}

	for _, policy := range list.Items {
		if policy.Spec.AppName == name && policy.Spec.Namespace == ns {
			isExist = true
			namespacedName = types.NamespacedName{
				Namespace: policy.GetNamespace(),
				Name:      policy.GetName(),
			}
		}
	}

	return isExist, namespacedName, nil

}
