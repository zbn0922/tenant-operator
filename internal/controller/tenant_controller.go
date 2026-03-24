/*
Copyright 2026.

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

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	tenantv1 "github.com/zbn0922/tenant-operator/api/v1"
)

// TenantReconciler reconciles a Tenant object
type TenantReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const TenantFinalizer = "tenant.zbn0922.github.com/finalizer"
const tenantResourceQuotaName = "tenant-resource-quota"

// +kubebuilder:rbac:groups=tenant.my.domain,resources=tenants,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tenant.my.domain,resources=tenants/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tenant.my.domain,resources=tenants/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Tenant object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.1/pkg/reconcile
func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)
	// 1、从cache获取资源
	var tenant tenantv1.Tenant
	if err := r.Get(ctx, req.NamespacedName, &tenant); err != nil {
		// 如果获取不到资源应该已经删除
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// 2、 清理资源
	if !tenant.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&tenant, TenantFinalizer) {
			// TODO: 删除时清理资源，优雅退出，备份、日志、通知之类的,
			if err := r.cleanupTenant(ctx, &tenant); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(&tenant, TenantFinalizer)
			if err := r.Update(ctx, &tenant); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}
	// 3、添加finalizer
	if tenant.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&tenant, TenantFinalizer) {
			controllerutil.AddFinalizer(&tenant, TenantFinalizer)

			if err := r.Update(ctx, &tenant); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}
	// 4、创建命名空间
	if res, err := r.EnsureNamespace(ctx, &tenant); err != nil || res != nil {
		return DefaultIfEmpty(res), err
	}
	// 5、创建resource quota
	if res, err := r.EnsureResourceQuota(ctx, &tenant); err != nil || res != nil {
		return DefaultIfEmpty(res), err
	}
	// 6、创建limit range
	// 7、创建network policy
	// 8、创建role binding
	// 9、更新状态为Ready
	return ctrl.Result{}, nil
}

func desiredNamespaceName(tenant *tenantv1.Tenant) string {
	if tenant.Spec.Namespace.Name != "" {
		return tenant.Spec.Namespace.Name
	}
	return "tenant-" + tenant.Name
}

// DefaultIfEmpty 如果 res 为 nil，返回默认的 empty Result，否则返回 res
func DefaultIfEmpty(res *ctrl.Result) ctrl.Result {
	if res != nil {
		return *res
	}
	return ctrl.Result{}
}

func (r *TenantReconciler) EnsureNamespace(ctx context.Context, tenant *tenantv1.Tenant) (*ctrl.Result, error) {
	var ns corev1.Namespace
	key := client.ObjectKey{Name: desiredNamespaceName(tenant)}
	if err := r.Get(ctx, key, &ns); err != nil {
		if apierrors.IsNotFound(err) {
			// 创建命名空间
			namespace := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: desiredNamespaceName(tenant),
				},
			}
			if err := r.Create(ctx, &namespace); err != nil {
				return nil, err
			}
			return &ctrl.Result{Requeue: true}, nil
		}
		return nil, err
	}

	// 已存在时，做最小校正
	updated := false
	if ns.Labels == nil {
		ns.Labels = map[string]string{}
	}
	if ns.Labels["tenant.zbn0922.github.com/name"] != tenant.Name {
		ns.Labels["tenant.zbn0922.github.com/name"] = tenant.Name
		updated = true
	}
	if ns.Labels["tenant.zbn0922.github.com/managed-by"] != "tenant-operator" {
		ns.Labels["tenant.zbn0922.github.com/managed-by"] = "tenant-operator"
		updated = true
	}

	if updated {
		if err := r.Update(ctx, &ns); err != nil {
			return nil, err
		}
		return &ctrl.Result{Requeue: true}, nil
	}

	// 如果既没有更新也没有创建，直接取下一步进行判断
	return nil, nil
}
func (r *TenantReconciler) EnsureResourceQuota(ctx context.Context, tenant *tenantv1.Tenant) (*ctrl.Result, error) {
	if tenant.Spec.Quota == nil {
		return nil, nil
	}

	// 查询缓存中是否存在
	key := client.ObjectKey{Name: tenant.Spec.Owner, Namespace: desiredNamespaceName(tenant)}
	var rq corev1.ResourceQuota
	if err := r.Get(ctx, key, &rq); err != nil {
		if apierrors.IsNotFound(err) {
			// 创建 ResourceQuota
			rq := corev1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tenant.Spec.Owner,
					Namespace: desiredNamespaceName(tenant),
				},
				Spec: corev1.ResourceQuotaSpec{
					Hard: quotaSpecToResourceList(tenant.Spec.Quota),
				},
			}
			if err := r.Create(ctx, &rq); err != nil {
				return nil, err
			}
			return &ctrl.Result{Requeue: true}, nil
		}
		return nil, err
	}

	// 已存在时，做最小校正
	desiredHard := quotaSpecToResourceList(tenant.Spec.Quota)
	if !apiequality.Semantic.DeepEqual(rq.Spec.Hard, desiredHard) {
		rq.Spec.Hard = desiredHard
		if err := r.Update(ctx, &rq); err != nil {
			return nil, err
		}
		return &ctrl.Result{Requeue: true}, nil
	}
	return nil, nil
}

func quotaSpecToResourceList(quota *tenantv1.QuotaSpec) corev1.ResourceList {
	hard := corev1.ResourceList{}
	if quota == nil {
		return hard
	}

	if quota.CPU != "" {
		hard[corev1.ResourceLimitsCPU] = resource.MustParse(quota.CPU)
	}
	if quota.Memory != "" {
		hard[corev1.ResourceLimitsMemory] = resource.MustParse(quota.Memory)
	}
	if quota.Pods > 0 {
		hard[corev1.ResourcePods] = *resource.NewQuantity(int64(quota.Pods), resource.DecimalSI)
	}
	if quota.Services > 0 {
		hard[corev1.ResourceServices] = *resource.NewQuantity(int64(quota.Services), resource.DecimalSI)
	}
	if quota.PersistentVolumeClaims > 0 {
		hard[corev1.ResourcePersistentVolumeClaims] = *resource.NewQuantity(int64(quota.PersistentVolumeClaims), resource.DecimalSI)
	}

	return hard
}

func (r *TenantReconciler) cleanupTenant(ctx context.Context, tenant *tenantv1.Tenant) error {
	/*
		更新 status.phase = Deleting
		删除 RoleBinding
		删除 NetworkPolicy
		删除 LimitRange
		删除 ResourceQuota
		删除 Namespace
	*/
	return nil
}

func (r *TenantReconciler) updateTenantStatus(ctx context.Context, tenant *tenantv1.Tenant, phase tenantv1.TenantPhase, ns string, lastErr string) error {
	tenant.Status.Phase = phase
	tenant.Status.Namespace = ns
	tenant.Status.ObservedGeneration = tenant.Generation
	tenant.Status.LastError = lastErr
	return r.Status().Update(ctx, tenant)
}

// SetupWithManager sets up the controller with the Manager.
func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tenantv1.Tenant{}).
		Named("tenant").
		Complete(r)
}
