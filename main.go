package main

import (
	"encoding/base64"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/aws-iam-authenticator/pkg/token"
)

func main() {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))

	svc := eks.New(sess)

	result, err := svc.DescribeCluster(&eks.DescribeClusterInput{
		Name: aws.String("ridiculous-gopher-1588984844"),
	})
	if err != nil {
		log.Fatal(err)
	}

	endpoint := aws.StringValue(result.Cluster.Endpoint)
	data := aws.StringValue(result.Cluster.CertificateAuthority.Data)

	cadata, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Fatal(err)
	}

	gen, err := token.NewGenerator(false, false)
	if err != nil {
		log.Fatal(err)
	}

	tok, err := gen.GetWithOptions(&token.GetTokenOptions{
		ClusterID: "ridiculous-gopher-1588984844",
	})
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(&rest.Config{
		Host:        endpoint,
		BearerToken: tok.Token,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: cadata,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)

	replicas := int32(3)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nginx-deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "nginx",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.18.0-alpine",
							Ports: []apiv1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	result2, err := deploymentsClient.Create(deployment)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created deployment %q.\n", result2.GetObjectMeta().GetName())
}
