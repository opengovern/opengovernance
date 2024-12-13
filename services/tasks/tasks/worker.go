package tasks

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateWorker(ctx context.Context, k8client client.Client, config Task, namespace string) error {
	var env []corev1.EnvVar
	for k, v := range config.EnvVars {
		env = append(env, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	switch config.WorkloadType {
	case WorkloadTypeJob:
		if config.ScaleConfig.Stream == "" {
		}
		// deployment
		deployment := v1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.Name,
				Namespace: namespace,
				Labels: map[string]string{
					"app": config.Name,
				},
			},
			Spec: v1.JobSpec{
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
						RestartPolicy: corev1.RestartPolicyNever,
						Containers: []corev1.Container{
							{
								Name:  config.Name,
								Image: config.ImageURL,
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
		scaledObject := kedav1alpha1.ScaledJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.Name + "-scaled-object",
				Namespace: namespace,
			},
			Spec: kedav1alpha1.ScaledJobSpec{
				JobTargetRef: &v1.JobSpec{
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
							RestartPolicy: corev1.RestartPolicyNever,
							Containers: []corev1.Container{
								{
									Name:  config.Name,
									Image: config.ImageURL,
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
				PollingInterval: aws.Int32(30),
				MinReplicaCount: aws.Int32(config.ScaleConfig.MinReplica),
				MaxReplicaCount: aws.Int32(config.ScaleConfig.MaxReplica),
				Triggers: []kedav1alpha1.ScaleTriggers{
					{
						Type: "nats-jetstream",
						Metadata: map[string]string{
							"account":                      "$G",
							"natsServerMonitoringEndpoint": config.ScaleConfig.NatsServerMonitoringEndpoint,
							"stream":                       config.ScaleConfig.Stream,
							"consumer":                     config.ScaleConfig.Consumer,
							"lagThreshold":                 config.ScaleConfig.LagThreshold,
							"useHttps":                     "false",
						},
					},
				},
			},
		}
		err = k8client.Create(ctx, &scaledObject)
		if err != nil {
			return err
		}
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
								Image: config.ImageURL,
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
			Spec: kedav1alpha1.ScaledObjectSpec{
				ScaleTargetRef: &kedav1alpha1.ScaleTarget{
					Name:       config.Name,
					Kind:       "Deployment",
					APIVersion: appsv1.SchemeGroupVersion.Version,
				},
				PollingInterval: aws.Int32(30),
				CooldownPeriod:  aws.Int32(300),
				MinReplicaCount: aws.Int32(config.ScaleConfig.MinReplica),
				MaxReplicaCount: aws.Int32(config.ScaleConfig.MaxReplica),
				Fallback: &kedav1alpha1.Fallback{
					FailureThreshold: 1,
					Replicas:         1,
				},
				Triggers: []kedav1alpha1.ScaleTriggers{
					{
						Type: "nats-jetstream",
						Metadata: map[string]string{
							"account":                      "$G",
							"natsServerMonitoringEndpoint": config.ScaleConfig.NatsServerMonitoringEndpoint,
							"stream":                       config.ScaleConfig.Stream,
							"consumer":                     config.ScaleConfig.Consumer,
							"lagThreshold":                 config.ScaleConfig.LagThreshold,
							"useHttps":                     "false",
						},
					},
				},
			},
		}
		err = k8client.Create(ctx, &scaledObject)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid workload type: %s", config.WorkloadType)
	}

	return nil
}
