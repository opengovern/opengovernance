package recommendation

import (
	"context"
	"fmt"
	"github.com/kaytu-io/open-governance/services/wastage/cost"
	pb "github.com/kaytu-io/plugin-kubernetes-internal/plugin/proto/src/golang"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"testing"
)

func TestKubernetesNodeCost(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	s := &Service{
		costSvc: cost.New("http://localhost:8080"),
		logger:  logger,
	}
	f, err := s.KubernetesNodeCost(context.Background(), pb.KubernetesNode{
		Id:   "aks-main0pool-42603833-vmss000000",
		Name: "aks-main0pool-42603833-vmss000000",
		Annotations: map[string]string{
			"csi.volume.kubernetes.io/nodeid":                        `{"disk.csi.azure.com":"aks-main0pool-42603833-vmss000000","file.csi.azure.com":"aks-main0pool-42603833-vmss000000"}`,
			"node.alpha.kubernetes.io/ttl":                           "0",
			"volumes.kubernetes.io/controller-managed-attach-detach": "true",
		},
		Labels: map[string]string{
			"agentpool":                                               "main0pool",
			"beta.kubernetes.io/arch":                                 "amd64",
			"beta.kubernetes.io/instance-type":                        "Standard_D8s_v5",
			"beta.kubernetes.io/os":                                   "linux",
			"failure-domain.beta.kubernetes.io/region":                "eastus2",
			"failure-domain.beta.kubernetes.io/zone":                  "eastus2-2",
			"kubernetes.azure.com/agentpool":                          "main0pool",
			"kubernetes.azure.com/cluster":                            "MC_kaytu-development_kaytu-aks_eastus2",
			"kubernetes.azure.com/consolidated-additional-properties": "2b9f482d-14b1-11ef-aa6c-e6cb19606ff6",
			"kubernetes.azure.com/kubelet-identity-client-id":         "044e3f42-e1bd-430a-92bb-6820098dbe50",
			"kubernetes.azure.com/mode":                               "system",
			"kubernetes.azure.com/network-policy":                     "azure",
			"kubernetes.azure.com/node-image-version":                 "AKSUbuntu-2204gen2containerd-202405.03.0",
			"kubernetes.azure.com/nodepool-type":                      "VirtualMachineScaleSets",
			"kubernetes.azure.com/os-sku":                             "Ubuntu",
			"kubernetes.azure.com/role":                               "agent",
			"kubernetes.azure.com/storageprofile":                     "managed",
			"kubernetes.azure.com/storagetier":                        "Premium_LRS",
			"kubernetes.io/arch":                                      "amd64",
			"kubernetes.io/hostname":                                  "aks-main0pool-42603833-vmss000000",
			"kubernetes.io/os":                                        "linux",
			"kubernetes.io/role":                                      "agent",
			"node-role.kubernetes.io/agent":                           "",
			"node.kubernetes.io/instance-type":                        "Standard_D8s_v5",
			"storageprofile":                                          "managed",
			"storagetier":                                             "Premium_LRS",
			"topology.disk.csi.azure.com/zone":                        "eastus2-2",
			"topology.kubernetes.io/region":                           "eastus2",
			"topology.kubernetes.io/zone":                             "eastus2-2",
		},
		Capacity: map[string]int64{
			"cpu":               8,
			"ephemeral-storage": 50620216000,
			"hugepages-1Gi":     0,
			"hugepages-2Mi":     0,
			"memory":            32863040000,
			"pods":              100,
		},
		NodeSystemInfo: nil,
		Taints:         nil,
		MaxPodCount:    0,
	})
	assert.NoError(t, err)
	fmt.Println(f)
}
