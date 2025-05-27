package k8s

import (
	"context"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNewClient(t *testing.T) {
	// Test with fake client for unit testing
	fakeClientset := fake.NewSimpleClientset()

	client, err := NewClientWithClientset(fakeClientset)
	if err != nil {
		t.Fatalf("Failed to create K8s client: %v", err)
	}

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.clientset != fakeClientset {
		t.Error("Expected client to use provided clientset")
	}
}

func TestGetPods(t *testing.T) {
	// Create fake clientset with some test pods
	fakeClientset := fake.NewSimpleClientset(
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-1",
				Namespace: "default",
				Labels: map[string]string{
					"app": "distributed-llm-agent",
				},
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
			},
		},
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-2",
				Namespace: "default",
				Labels: map[string]string{
					"app": "distributed-llm-agent",
				},
			},
			Status: v1.PodStatus{
				Phase: v1.PodPending,
			},
		},
	)

	client, err := NewClientWithClientset(fakeClientset)
	if err != nil {
		t.Fatalf("Failed to create K8s client: %v", err)
	}

	ctx := context.Background()
	pods, err := client.GetPods(ctx, "default", "app=distributed-llm-agent")
	if err != nil {
		t.Fatalf("Failed to get pods: %v", err)
	}

	if len(pods.Items) != 2 {
		t.Errorf("Expected 2 pods, got %d", len(pods.Items))
	}

	// Check first pod
	if pods.Items[0].Name != "test-pod-1" {
		t.Errorf("Expected pod name 'test-pod-1', got %s", pods.Items[0].Name)
	}

	if pods.Items[0].Status.Phase != v1.PodRunning {
		t.Errorf("Expected pod phase Running, got %s", pods.Items[0].Status.Phase)
	}
}

func TestGetNodes(t *testing.T) {
	// Create fake clientset with test nodes
	fakeClientset := fake.NewSimpleClientset(
		&v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-1",
				Labels: map[string]string{
					"kubernetes.io/os": "linux",
				},
			},
			Status: v1.NodeStatus{
				Conditions: []v1.NodeCondition{
					{
						Type:   v1.NodeReady,
						Status: v1.ConditionTrue,
					},
				},
			},
		},
		&v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-2",
				Labels: map[string]string{
					"kubernetes.io/os": "linux",
				},
			},
			Status: v1.NodeStatus{
				Conditions: []v1.NodeCondition{
					{
						Type:   v1.NodeReady,
						Status: v1.ConditionFalse,
					},
				},
			},
		},
	)

	client, err := NewClientWithClientset(fakeClientset)
	if err != nil {
		t.Fatalf("Failed to create K8s client: %v", err)
	}

	ctx := context.Background()
	nodes, err := client.GetNodes(ctx)
	if err != nil {
		t.Fatalf("Failed to get nodes: %v", err)
	}

	if len(nodes.Items) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(nodes.Items))
	}

	// Check first node
	if nodes.Items[0].Name != "node-1" {
		t.Errorf("Expected node name 'node-1', got %s", nodes.Items[0].Name)
	}
}

func TestWatchPods(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()

	client, err := NewClientWithClientset(fakeClientset)
	if err != nil {
		t.Fatalf("Failed to create K8s client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start watching in a goroutine
	eventChan := make(chan PodEvent, 10)
	go func() {
		err := client.WatchPods(ctx, "default", "app=distributed-llm-agent", eventChan)
		if err != nil && err != context.DeadlineExceeded {
			t.Errorf("Watch pods failed: %v", err)
		}
	}()

	// Create a pod to trigger an event
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app": "distributed-llm-agent",
			},
		},
	}

	_, err = fakeClientset.CoreV1().Pods("default").Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test pod: %v", err)
	}

	// Wait for the context to timeout
	<-ctx.Done()

	// Close the channel
	close(eventChan)
}

func TestCreateConfigMap(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()

	client, err := NewClientWithClientset(fakeClientset)
	if err != nil {
		t.Fatalf("Failed to create K8s client: %v", err)
	}

	ctx := context.Background()
	data := map[string]string{
		"config.yaml": "test: value",
		"version":     "1.0.0",
	}

	configMap, err := client.CreateConfigMap(ctx, "default", "test-config", data)
	if err != nil {
		t.Fatalf("Failed to create ConfigMap: %v", err)
	}

	if configMap.Name != "test-config" {
		t.Errorf("Expected ConfigMap name 'test-config', got %s", configMap.Name)
	}

	if configMap.Namespace != "default" {
		t.Errorf("Expected ConfigMap namespace 'default', got %s", configMap.Namespace)
	}

	if len(configMap.Data) != 2 {
		t.Errorf("Expected 2 data entries, got %d", len(configMap.Data))
	}

	if configMap.Data["config.yaml"] != "test: value" {
		t.Errorf("Expected config.yaml value 'test: value', got %s", configMap.Data["config.yaml"])
	}
}

func TestGetConfigMap(t *testing.T) {
	// Create a test ConfigMap
	testConfigMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"config.yaml": "test: value",
		},
	}

	fakeClientset := fake.NewSimpleClientset(testConfigMap)

	client, err := NewClientWithClientset(fakeClientset)
	if err != nil {
		t.Fatalf("Failed to create K8s client: %v", err)
	}

	ctx := context.Background()
	configMap, err := client.GetConfigMap(ctx, "default", "test-config")
	if err != nil {
		t.Fatalf("Failed to get ConfigMap: %v", err)
	}

	if configMap.Name != "test-config" {
		t.Errorf("Expected ConfigMap name 'test-config', got %s", configMap.Name)
	}

	if configMap.Data["config.yaml"] != "test: value" {
		t.Errorf("Expected config.yaml value 'test: value', got %s", configMap.Data["config.yaml"])
	}
}

func TestDeleteConfigMap(t *testing.T) {
	// Create a test ConfigMap
	testConfigMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	}

	fakeClientset := fake.NewSimpleClientset(testConfigMap)

	client, err := NewClientWithClientset(fakeClientset)
	if err != nil {
		t.Fatalf("Failed to create K8s client: %v", err)
	}

	ctx := context.Background()
	err = client.DeleteConfigMap(ctx, "default", "test-config")
	if err != nil {
		t.Fatalf("Failed to delete ConfigMap: %v", err)
	}

	// Verify it's deleted
	_, err = client.GetConfigMap(ctx, "default", "test-config")
	if err == nil {
		t.Error("Expected error when getting deleted ConfigMap")
	}
}
