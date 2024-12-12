package tasks

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WorkloadType string

const (
	WorkloadTypeJob        WorkloadType = "job"
	WorkloadTypeDeployment WorkloadType = "deployment"
)

type WorkerConfig struct {
	Name         string
	Image        string            `json:"image"`
	Command      string            `json:"command"`
	WorkloadType WorkloadType      `json:"workload_type"`
	EnvVars      map[string]string `json:"env_vars"`
}

func CreateWorker(ctx context.Context, k8client client.Client, config WorkerConfig, namespace string) error {
	var env []corev1.EnvVar
	for k, v := range config.EnvVars {
		env = append(env, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	switch config.WorkloadType {
	case WorkloadTypeJob:
	case WorkloadTypeDeployment:
		// deployment
		deployment := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.Name,
				Namespace: namespace,
				Labels: map[string]string{
					"app": config.Name,
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: aws.Int32(0),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": config.Name,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": config.Name,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  config.Name,
								Image: config.Image,
								Command: []string{
									config.Command,
								},
								ImagePullPolicy: corev1.PullAlways,
								Env:             env,
							},
						},
					},
				},
			},
		}
		err := k8client.Create(ctx, &deployment)
		if err != nil {
			return err
		}

		// scaled-object
		scaledObject := kedav1alpha1.ScaledObject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.Name + "-scaled-object",
				Namespace: namespace,
			},
			Spec: describerScaledObject.Spec,
		}
	default:
		return fmt.Errorf("invalid workload type: %s", config.WorkloadType)
	}

	return nil
}
