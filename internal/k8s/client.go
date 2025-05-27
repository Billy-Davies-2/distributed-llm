package k8s

import (
    "context"
    "fmt"
    "os"

    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
)

type KubernetesClient struct {
    clientset *kubernetes.Clientset
}

func NewKubernetesClient() (*KubernetesClient, error) {
    kubeconfig := os.Getenv("KUBECONFIG")
    if kubeconfig == "" {
        kubeconfig = "/home/user/.kube/config" // Default kubeconfig path
    }

    config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
    if err != nil {
        return nil, fmt.Errorf("failed to build kubeconfig: %v", err)
    }

    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
    }

    return &KubernetesClient{clientset: clientset}, nil
}

func (k *KubernetesClient) CreatePod(namespace string, pod *corev1.Pod) error {
    _, err := k.clientset.CoreV1().Pods(namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
    return err
}

func (k *KubernetesClient) DeletePod(namespace, name string) error {
    return k.clientset.CoreV1().Pods(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

func (k *KubernetesClient) ListPods(namespace string) (*corev1.PodList, error) {
    return k.clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
}