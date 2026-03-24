package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	tenantv1 "github.com/zbn0922/tenant-operator/api/v1"
)

func TestEnsureResourceQuotaCreatesQuotaFromTenantSpec(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := tenantv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add tenant scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}

	tenant := &tenantv1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
		},
		Spec: tenantv1.TenantSpec{
			Namespace: tenantv1.NamespaceSpec{Name: "tenant-demo"},
			Quota: &tenantv1.QuotaSpec{
				CPU:                    "2",
				Memory:                 "4Gi",
				Pods:                   10,
				Services:               5,
				PersistentVolumeClaims: 3,
			},
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(tenant, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "tenant-demo"}}).
		Build()

	r := &TenantReconciler{Client: cl, Scheme: scheme}

	res, err := r.EnsureResourceQuota(context.Background(), tenant)
	if err != nil {
		t.Fatalf("EnsureResourceQuota returned error: %v", err)
	}
	if res == nil || !res.Requeue {
		t.Fatalf("expected requeue result after create, got %#v", res)
	}

	var rq corev1.ResourceQuota
	key := client.ObjectKey{Name: tenantResourceQuotaName, Namespace: "tenant-demo"}
	if err := cl.Get(context.Background(), key, &rq); err != nil {
		t.Fatalf("get resourcequota: %v", err)
	}

	assertQuantityEqual(t, rq.Spec.Hard[corev1.ResourceLimitsCPU], "2")
	assertQuantityEqual(t, rq.Spec.Hard[corev1.ResourceLimitsMemory], "4Gi")
	assertQuantityEqual(t, rq.Spec.Hard[corev1.ResourcePods], "10")
	assertQuantityEqual(t, rq.Spec.Hard[corev1.ResourceServices], "5")
	assertQuantityEqual(t, rq.Spec.Hard[corev1.ResourcePersistentVolumeClaims], "3")
}

func TestEnsureResourceQuotaUpdatesExistingQuota(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := tenantv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add tenant scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}

	tenant := &tenantv1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
		},
		Spec: tenantv1.TenantSpec{
			Namespace: tenantv1.NamespaceSpec{Name: "tenant-demo"},
			Quota: &tenantv1.QuotaSpec{
				CPU:    "4",
				Memory: "8Gi",
				Pods:   20,
			},
		},
	}

	existing := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenantResourceQuotaName,
			Namespace: "tenant-demo",
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceLimitsCPU:    resource.MustParse("1"),
				corev1.ResourceLimitsMemory: resource.MustParse("1Gi"),
				corev1.ResourcePods:         resource.MustParse("1"),
			},
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(tenant, existing, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "tenant-demo"}}).
		Build()

	r := &TenantReconciler{Client: cl, Scheme: scheme}

	res, err := r.EnsureResourceQuota(context.Background(), tenant)
	if err != nil {
		t.Fatalf("EnsureResourceQuota returned error: %v", err)
	}
	if res == nil || !res.Requeue {
		t.Fatalf("expected requeue result after update, got %#v", res)
	}

	var rq corev1.ResourceQuota
	key := client.ObjectKey{Name: tenantResourceQuotaName, Namespace: "tenant-demo"}
	if err := cl.Get(context.Background(), key, &rq); err != nil {
		t.Fatalf("get resourcequota: %v", err)
	}

	assertQuantityEqual(t, rq.Spec.Hard[corev1.ResourceLimitsCPU], "4")
	assertQuantityEqual(t, rq.Spec.Hard[corev1.ResourceLimitsMemory], "8Gi")
	assertQuantityEqual(t, rq.Spec.Hard[corev1.ResourcePods], "20")
}

func assertQuantityEqual(t *testing.T, got resource.Quantity, want string) {
	t.Helper()
	expected := resource.MustParse(want)
	if got.Cmp(expected) != 0 {
		t.Fatalf("quantity mismatch: got %s want %s", got.String(), expected.String())
	}
}
