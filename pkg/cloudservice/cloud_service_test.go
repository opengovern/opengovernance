package cloudservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCategoryByResourceType(t *testing.T) {
	assert.Equal(t, "Infrastructure", CategoryByResourceType("AwS::Ec2::Instance"))
	assert.Equal(t, "Networking", CategoryByResourceType("Microsoft.Network/applicationGateway"))
}

func TestIsCommonByResourceType(t *testing.T) {
	assert.Equal(t, true, IsCommonByResourceType("AwS::Ec2::Instance"))
	assert.Equal(t, false, IsCommonByResourceType("aws::account::whatever"))
}

func TestServiceNameByResourceType_ResourceList(t *testing.T) {
	assert.Equal(t, "Amazon EC2 Instance", ServiceNameByResourceType("AwS::Ec2::Instance"))
	assert.Equal(t, "Application Gateway", ServiceNameByResourceType("Microsoft.network/APPlicationGateways"))
}

func TestServiceNameByResourceType_Namespace(t *testing.T) {
	assert.Equal(t, "Application Gateway", ServiceNameByResourceType("Microsoft.Network/whatever"))
	assert.Equal(t, "Elastic Cloud Compute (EC2)", ServiceNameByResourceType("AWS::EC2::whatever"))
}

func TestListCategories(t *testing.T) {
	cats := ListCategories()
	assert.Len(t, cats, 32)
	assert.Contains(t, cats, "Infrastructure")
}

func TestResourceListByCategory(t *testing.T) {
	resourceList := ResourceListByCategory("Database")
	assert.Len(t, resourceList, 16)
	assert.Contains(t, resourceList, "aws::dynamodb::table")
	assert.Contains(t, resourceList, "microsoft.sql/servers")
}

func TestResourceListByServiceName(t *testing.T) {
	resourceList := ResourceListByServiceName("Amazon Simple Storage Service (S3)")
	assert.Len(t, resourceList, 4)
	assert.Contains(t, resourceList, "aws::s3::bucket")
}
