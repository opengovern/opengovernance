package configs

var ResourceTypesList = []string{
	"OCI::Image",
	"OCI::ImageTag",
}

var TablesToResourceTypes = map[string]string{
	"oci_image":     "OCI::Image",
	"oci_image_tag": "OCI::ImageTag",
}
