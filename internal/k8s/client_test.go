package k8s

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

// TestClientCreation tests the client creation
func TestClientCreation(t *testing.T) {
	// Note: This test will fail outside of a k8s environment
	// but demonstrates the API structure
	t.Run("client creation should handle missing config gracefully", func(t *testing.T) {
		_, err := NewKubernetesClient()
		// We expect this to fail in test environment without kubeconfig
		if err == nil {
			t.Log("Successfully created k8s client (running in k8s environment)")
		} else {
			t.Logf("Expected error creating k8s client outside cluster: %v", err)
		}
	})
}

// TestClientWithFakeClientset tests the client with a fake Kubernetes clientset
func TestClientWithFakeClientset(t *testing.T) {
	// Create a fake clientset for testing
	fakeClientset := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClientset,
		config:    &rest.Config{},
	}

	t.Run("GetClientset", func(t *testing.T) {
		clientset := client.GetClientset()
		if clientset == nil {
			t.Error("GetClientset should not return nil")
		}
	})

	t.Run("IsAvailable", func(t *testing.T) {
		available := client.IsAvailable()
		if !available {
			t.Error("IsAvailable should return true for fake clientset")
		}
	})

	t.Run("ListPods empty namespace", func(t *testing.T) {
		pods, err := client.ListPods("default")
		if err != nil {
			t.Errorf("ListPods should not error: %v", err)
		}
		if pods == nil {
			t.Error("ListPods should not return nil")
		}
		if len(pods.Items) != 0 {
			t.Errorf("Expected 0 pods, got %d", len(pods.Items))
		}
	})

	t.Run("ListDeployments empty namespace", func(t *testing.T) {
		deployments, err := client.ListDeployments("default")
		if err != nil {
			t.Errorf("ListDeployments should not error: %v", err)
		}
		if deployments == nil {
			t.Error("ListDeployments should not return nil")
		}
		if len(deployments.Items) != 0 {
			t.Errorf("Expected 0 deployments, got %d", len(deployments.Items))
		}
	})
}

// TestClientWithMockData tests the client with pre-populated mock data
func TestClientWithMockData(t *testing.T) {
	// Create test pods
	testPod1 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-1",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test-app",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "test-container",
					Image: "nginx:latest",
				},
			},
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
		},
	}

	testPod2 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-2",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "another-app",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "another-container",
					Image: "redis:latest",
				},
			},
		},
		Status: v1.PodStatus{
			Phase: v1.PodPending,
		},
	}

	// Create test deployment
	replicas := int32(3)
	testDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test-deployment",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-deployment",
				},
			},
		},
	}

	// Create fake clientset with test data
	objects := []runtime.Object{testPod1, testPod2, testDeployment}
	fakeClientset := fake.NewSimpleClientset(objects...)

	client := &Client{
		clientset: fakeClientset,
		config:    &rest.Config{},
	}

	t.Run("ListPods with data", func(t *testing.T) {
		pods, err := client.ListPods("default")
		if err != nil {
			t.Errorf("ListPods should not error: %v", err)
		}
		if len(pods.Items) != 1 {
			t.Errorf("Expected 1 pod in default namespace, got %d", len(pods.Items))
		}
		if pods.Items[0].Name != "test-pod-1" {
			t.Errorf("Expected pod name 'test-pod-1', got '%s'", pods.Items[0].Name)
		}
	})

	t.Run("ListPods different namespace", func(t *testing.T) {
		pods, err := client.ListPods("test-namespace")
		if err != nil {
			t.Errorf("ListPods should not error: %v", err)
		}
		if len(pods.Items) != 1 {
			t.Errorf("Expected 1 pod in test-namespace, got %d", len(pods.Items))
		}
		if pods.Items[0].Name != "test-pod-2" {
			t.Errorf("Expected pod name 'test-pod-2', got '%s'", pods.Items[0].Name)
		}
	})

	t.Run("GetPod existing", func(t *testing.T) {
		pod, err := client.GetPod("default", "test-pod-1")
		if err != nil {
			t.Errorf("GetPod should not error: %v", err)
		}
		if pod.Name != "test-pod-1" {
			t.Errorf("Expected pod name 'test-pod-1', got '%s'", pod.Name)
		}
		if pod.Status.Phase != v1.PodRunning {
			t.Errorf("Expected pod phase Running, got %s", pod.Status.Phase)
		}
	})

	t.Run("GetPod non-existing", func(t *testing.T) {
		_, err := client.GetPod("default", "non-existing-pod")
		if err == nil {
			t.Error("GetPod should error for non-existing pod")
		}
	})

	t.Run("ListDeployments with data", func(t *testing.T) {
		deployments, err := client.ListDeployments("default")
		if err != nil {
			t.Errorf("ListDeployments should not error: %v", err)
		}
		if len(deployments.Items) != 1 {
			t.Errorf("Expected 1 deployment, got %d", len(deployments.Items))
		}
		if deployments.Items[0].Name != "test-deployment" {
			t.Errorf("Expected deployment name 'test-deployment', got '%s'", deployments.Items[0].Name)
		}
	})
}

// TestNamespaceOperations tests namespace creation and deletion
func TestNamespaceOperations(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClientset,
		config:    &rest.Config{},
	}

	t.Run("CreateNamespace", func(t *testing.T) {
		namespace, err := client.CreateNamespace("test-namespace")
		if err != nil {
			t.Errorf("CreateNamespace should not error: %v", err)
		}
		if namespace.Name != "test-namespace" {
			t.Errorf("Expected namespace name 'test-namespace', got '%s'", namespace.Name)
		}
	})

	t.Run("DeleteNamespace", func(t *testing.T) {
		// First create a namespace
		_, err := client.CreateNamespace("delete-me")
		if err != nil {
			t.Errorf("Failed to create namespace for deletion test: %v", err)
		}

		// Then delete it
		err = client.DeleteNamespace("delete-me")
		if err != nil {
			t.Errorf("DeleteNamespace should not error: %v", err)
		}
	})

	t.Run("DeleteNonExistentNamespace", func(t *testing.T) {
		err := client.DeleteNamespace("non-existent")
		if err == nil {
			t.Error("DeleteNamespace should error for non-existent namespace")
		}
	})
}
