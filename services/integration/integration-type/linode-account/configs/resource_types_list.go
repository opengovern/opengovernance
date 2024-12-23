package configs

var TablesToResourceTypes = map[string]string{
	"linode_account":            "Linode/Account",
	"linode_database":           "Linode/Database",
	"linode_domain":             "Linode/Domain",
	"linode_firewall":           "Linode/Firewall",
	"linode_image":              "Linode/Image",
	"linode_kubernetes_cluster": "Linode/Kubernetes/Cluster",
	"linode_event":              "Linode/Event",
	"linode_instance":           "Linode/Instance",
	"linode_longview_client":    "Linode/Longview/Client",
	"linode_node_balancer":      "Linode/NodeBalancer",
	"linode_object_storage":     "Linode/ObjectStorage",
	"linode_stack_script":       "Linode/StackScript",
	"linode_vpc":                "Linode/Vpc",
	"linode_volume":             "Linode/Volume",
	"linode_ip_address":         "Linode/IPAddress",
}

var ResourceTypesList = []string{
	"Linode/Account",
	"Linode/Database",
	"Linode/Domain",
	"Linode/Firewall",
	"Linode/Image",
	"Linode/Kubernetes/Cluster",
	"Linode/Event",
	"Linode/Instance",
	"Linode/Longview/Client",
	"Linode/NodeBalancer",
	"Linode/ObjectStorage",
	"Linode/StackScript",
	"Linode/Vpc",
	"Linode/Volume",
	"Linode/IPAddress",
}
