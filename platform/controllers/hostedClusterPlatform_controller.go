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

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	hyperv1 "github.com/openshift/hypershift/api/v1alpha1"
	hypv1alpha1 "github.com/openshift/hypershift/api/v1alpha1"
	"github.com/openshift/hypershift/cmd/cluster/aws"
	"github.com/openshift/hypershift/cmd/cluster/core"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

// HostedClusterPlatformReconciler reconciles a HostedCluster object
type HostedClusterPlatformReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const (
	destroyFinalizer = "openshift.io/destroy-cluster"
)

// +kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources=HostedClusterPlatform,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources=HostedClusterPlatform/status,verbs=get;update;patch

func (r *HostedClusterPlatformReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("hostedClusterPlatform", req.NamespacedName)

	// your logic here
	var hc hypv1alpha1.HostedCluster

	err := r.Client.Get(ctx, req.NamespacedName, &hc)
	if err != nil {
		log.Info("Resource has been deleted " + req.NamespacedName.Name)
		return ctrl.Result{}, nil
	}

	condition := metav1.Condition{
		Type:               string(hyperv1.CloudProviderConfigured),
		ObservedGeneration: hc.Generation,
	}

	if hc.Spec.InfraID != "" && hc.DeletionTimestamp == nil {
		//@todo Make a switch with cases as we get new providers
		// This should also probably run a full set of validation for all AWS objects that are created...
		if hc.Spec.Platform.AWS != nil {
			pAWS := hc.Spec.Platform.AWS
			if pAWS.CloudProviderConfig != nil &&
				pAWS.CloudProviderConfig.Subnet != nil &&
				pAWS.CloudProviderConfig.Subnet.ID != nil &&
				len(pAWS.Roles) > 0 {
				condition.Status = metav1.ConditionTrue
				condition.Message = ""
				condition.Reason = hyperv1.CloudProviderConfiguredAsExpected
			} else {
				condition.Status = metav1.ConditionFalse
				condition.Message = "One or more AWS configurations failed, review the logs"
				condition.Reason = hyperv1.CloudProviderMisConfiguredReason
			}
			meta.SetStatusCondition(&hc.Status.Conditions, condition)

			// Persist status updates
			if err := r.Client.Status().Update(ctx, &hc); err != nil {
				if apierrors.IsConflict(err) {
					return ctrl.Result{Requeue: true}, nil
				}
				return ctrl.Result{}, fmt.Errorf("failed to update status: %w", err)
			}
			return ctrl.Result{}, nil
		}
	}
	var providerSecret v1.Secret
	var pullSecret v1.Secret

	err = r.Client.Get(ctx, types.NamespacedName{Namespace: hc.Namespace, Name: "aws"}, &providerSecret)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Client.Get(ctx, types.NamespacedName{Namespace: hc.Namespace, Name: hc.Spec.PullSecret.Name}, &pullSecret)
	if err != nil {
		return ctrl.Result{}, err
	}

	opts := core.CreateOptions{
		Namespace:                        hc.Namespace,
		Name:                             hc.Name,
		ReleaseImage:                     hc.Spec.Release.Image,
		ControlPlaneOperatorImage:        "",
		PullSecretFile:                   "",
		PullSecret:                       pullSecret.Data[".dockerconfigjson"],
		SSHKeyFile:                       "",
		NodePoolReplicas:                 2,
		Render:                           false,
		InfrastructureJSON:               "",
		InfraID:                          hc.Spec.InfraID,
		Annotations:                      []string{},
		NetworkType:                      string(hyperv1.OpenShiftSDN),
		FIPS:                             false,
		AutoRepair:                       false,
		ControlPlaneAvailabilityPolicy:   "SingleReplica",
		InfrastructureAvailabilityPolicy: "HighlyAvailable",
		EtcdStorageClass:                 "",
	}

	opts.AWSPlatform = core.AWSPlatformOptions{
		AWSCredentialsFile: "",
		AWSKey:             string(providerSecret.Data["aws_access_key_id"]),
		AWSSecretKey:       string(providerSecret.Data["aws_secret_access_key"]),
		Region:             hc.Spec.Platform.AWS.Region,
		BaseDomain:         hc.Spec.DNS.BaseDomain,
		InstanceType:       "t3.2xlarge",
		RootVolumeType:     "gp2",
		RootVolumeSize:     16,
		RootVolumeIOPS:     0,
	}

	// Destroying Platform infrastructure used by the HostedCluster scheduled for deletion
	if hc.DeletionTimestamp != nil {
		dOpts := core.DestroyOptions{
			Namespace: opts.Namespace,
			Name:      opts.Name,
			InfraID:   opts.InfraID,
			AWSPlatform: core.AWSPlatformDestroyOptions{
				AWSCredentialsFile: "",
				AWSKey:             opts.AWSPlatform.AWSKey,
				AWSSecretKey:       opts.AWSPlatform.AWSSecretKey,
				PreserveIAM:        false,
				Region:             opts.AWSPlatform.Region,
				BaseDomain:         opts.AWSPlatform.BaseDomain,
			},
			ClusterGracePeriod: 10 * time.Minute,
		}

		if err = aws.DestroyCluster(ctx, &dOpts); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update status: %w", err)
		}
		return ctrl.Result{}, nil
	}

	// Creating Platform infrastructure used by the HostedCluster NodePools and ingress
	if hc.Spec.InfraID == "" {

		err = aws.CreateCluster(ctx, &opts)
		r.Client.Get(ctx, req.NamespacedName, &hc)

		if err == nil {
			condition.Status = metav1.ConditionTrue
			condition.Message = ""
			condition.Reason = hyperv1.CloudProviderConfiguredAsExpected
		} else {
			condition.Status = metav1.ConditionFalse
			condition.Message = err.Error()
			condition.Reason = hyperv1.CloudProviderMisConfiguredReason
		}
		meta.SetStatusCondition(&hc.Status.Conditions, condition)

		// Persist status updates
		if err := r.Client.Status().Update(ctx, &hc); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, fmt.Errorf("failed to update status: %w", err)
		}
	}

	return ctrl.Result{}, err
}

func (r *HostedClusterPlatformReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hypv1alpha1.HostedCluster{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 2,
		}).
		Complete(r)
}
