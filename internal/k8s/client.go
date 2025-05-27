package k8s

import (
	"context"
	"fmt"
	"time"

	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Client wraps the Kubernetes client with additional functionality
type Client struct {
	clientset kubernetes.Interface
	config    *rest.Config
}

// NewKubernetesClient creates a new Kubernetes client
func NewKubernetesClient() (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig if not running in cluster
		home := homedir.HomeDir()
		kubeconfig := filepath.Join(home, ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Client{
		clientset: clientset,
		config:    config,
	}, nil
}

// GetClientset returns the underlying Kubernetes clientset
func (c *Client) GetClientset() kubernetes.Interface {
	return c.clientset
}

// ListPods returns a list of pods in the specified namespace
func (c *Client) ListPods(namespace string) (*v1.PodList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
	}

	return pods, nil
}

// ListDeployments returns a list of deployments in the specified namespace
func (c *Client) ListDeployments(namespace string) (*appsv1.DeploymentList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	deployments, err := c.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments in namespace %s: %w", namespace, err)
	}

	return deployments, nil
}

// GetPod gets a specific pod by name in the specified namespace
func (c *Client) GetPod(namespace, name string) (*v1.Pod, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s in namespace %s: %w", name, namespace, err)
	}

	return pod, nil
}

// CreateNamespace creates a new namespace
func (c *Client) CreateNamespace(name string) (*v1.Namespace, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	result, err := c.clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create namespace %s: %w", name, err)
	}

	return result, nil
}

// DeleteNamespace deletes a namespace
func (c *Client) DeleteNamespace(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := c.clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete namespace %s: %w", name, err)
	}

	return nil
}

// IsAvailable checks if the Kubernetes cluster is available
func (c *Client) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
	return err == nil
}
