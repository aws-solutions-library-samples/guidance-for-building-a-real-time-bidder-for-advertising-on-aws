package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	vegeta "github.com/tsenart/vegeta/v12/lib"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func loadFromK8s(conf *config) ([]vegeta.Metrics, error) {
	cs, err := createClientset(conf)

	if err != nil {
		return nil, err
	}

	job, err := findJob(conf, cs)

	if err != nil {
		return nil, err
	}

	pods, err := findJobPods(cs, job)

	if err != nil {
		return nil, err
	}

	inputs := make([]vegeta.Metrics, 0)

	for _, pod := range pods.Items {
		data, err := getPodLogs(cs, pod)

		if err != nil {
			return nil, err
		}

		data, err = getMetricsFromLogs(data)

		if err != nil {
			return nil, err
		}

		metrics, err := unmarshallMetrics(data)

		if err != nil {
			return nil, err
		}

		inputs = append(inputs, *metrics)
	}

	return inputs, nil
}

func createClientset(conf *config) (*kubernetes.Clientset, error) {
	var err error
	var k8sConfig *rest.Config

	if conf.K8sKubeConfig == "" {
		k8sConfig, err = rest.InClusterConfig()
	} else {
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", conf.K8sKubeConfig)
	}

	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(k8sConfig)
}

func findJob(conf *config, cs *kubernetes.Clientset) (*batchv1.Job, error) {
	job, err := cs.BatchV1().Jobs(conf.K8sNamespace).Get(context.Background(), conf.K8sJobName, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	if job.Status.Failed > 0 {
		return nil, errors.New("some of load generator jobs failed")
	}

	if job.Status.CompletionTime == nil {
		return nil, errors.New("load generator is still running")
	}

	return job, nil
}

func findJobPods(cs *kubernetes.Clientset, job *batchv1.Job) (*corev1.PodList, error) {
	selector := selectorForLabels(job.Spec.Selector.MatchLabels)

	return cs.CoreV1().Pods(job.Namespace).List(
		context.Background(),
		metav1.ListOptions{
			LabelSelector: selector,
		},
	)
}

func getPodLogs(cs *kubernetes.Clientset, pod corev1.Pod) ([]byte, error) {
	req := cs.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
	res := req.Do(context.Background())

	return res.Raw()
}

func getMetricsFromLogs(logs []byte) ([]byte, error) {
	lines := strings.Split(string(logs), "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "{") {
			return []byte(line), nil
		}
	}

	return nil, errors.New("unable to find metrics in logs")
}

func selectorForLabels(labels map[string]string) string {
	selectors := make([]string, 0)

	for key, value := range labels {
		selector := fmt.Sprintf(
			"%s=%s",
			url.QueryEscape(key),
			url.QueryEscape(value),
		)

		selectors = append(selectors, selector)
	}

	return strings.Join(selectors, ",")
}
