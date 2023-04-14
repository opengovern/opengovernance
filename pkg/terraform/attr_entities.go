package terraform

var awsRedshiftSnapshot = []Mapping{
	{Name: "manual_snapshot_retention_period", Steampipe: "manual_snapshot_retention_period"},
	{Name: "owner_account", Steampipe: "owner_account"},
	{Name: "snapshot_identifier", Steampipe: "snapshot_identifier"},
	{Name: "arn", Steampipe: "akas"},
	{Name: "cluster_identifier", Steampipe: "cluster_identifier"},
	{Name: "kms_key_id", Steampipe: "kms_key_id"},
}
var awsAuditmanagerControl = []Mapping{
	{Name: "action_plan_title", Steampipe: "action_plan_title"},
	{Name: "arn", Steampipe: "akas"},
	{Name: "testing_information", Steampipe: "testing_information"},
	{Name: "action_plan_instructions", Steampipe: "action_plan_instructions"},
	{Name: "arn", Steampipe: "arn"},
	{Name: "description", Steampipe: "description"},
	{Name: "id", Steampipe: "id"},
	{Name: "name", Steampipe: "name"},
	{Name: "type", Steampipe: "type"},
}
var awsGlacierVault = []Mapping{
	{Name: "arn", Steampipe: "akas"},
	{Name: "name", Steampipe: "vault_name"},
	{Name: "location", DeepField: "Metadata.Location"}, //TODO-fix this mapping
	{Name: "access_policy", Steampipe: "policy"},
	{Name: "notification.events"},    //TODO-fix this mapping
	{Name: "notification.sns_topic"}, //TODO-fix this mapping
}
var awsEc2Routetable = []Mapping{
	{Name: "propagating_vgws", Steampipe: "propagating_vgws"},
	{Name: "vpc_id", Steampipe: "vpc_id"},
	{Name: "arn", Steampipe: "akas"},
	{Name: "owner_id", Steampipe: "owner_id"},
	{Name: "route.cidr_block", DeepField: "SourceJobID"}, //TODO-fix this mapping
	{Name: "route.destination_prefix_list_id", DeepField: "Description.RouteTable.Routes.DestinationPrefixListId"}, //TODO-fix this mapping
	{Name: "route.ipv6_cidr_block", DeepField: "Description.RouteTable.Routes.DestinationIpv6CidrBlock"},           //TODO-fix this mapping
	{Name: "route.carrier_gateway_id", DeepField: "Description.RouteTable.Routes.CarrierGatewayId"},                //TODO-fix this mapping
	{Name: "route.core_network_arn", DeepField: "Description.RouteTable.Routes.CoreNetworkArn"},                    //TODO-fix this mapping
	{Name: "route.egress_only_gateway_id", DeepField: "Description.RouteTable.Routes.EgressOnlyInternetGatewayId"}, //TODO-fix this mapping
	{Name: "route.gateway_id", DeepField: "Description.RouteTable.Routes.GatewayId"},                               //TODO-fix this mapping
	{Name: "route.instance_id", DeepField: "Description.RouteTable.Routes.InstanceId"},                             //TODO-fix this mapping
	{Name: "route.local_gateway_id", DeepField: "Description.RouteTable.Routes.LocalGatewayId"},                    //TODO-fix this mapping
	{Name: "route.nat_gateway_id", DeepField: "Description.RouteTable.Routes.GatewayId"},                           //TODO-fix this mapping
	{Name: "route.network_interface_id", DeepField: "Description.RouteTable.Routes.NetworkInterfaceId"},            //TODO-fix this mapping
	{Name: "route.transit_gateway_id", DeepField: "Description.RouteTable.Routes.TransitGatewayId"},                //TODO-fix this mapping
	{Name: "route.vpc_endpoint_id", DeepField: "Description.RouteTable.Routes.VpcPeeringConnectionId"},             //TODO-fix this mapping
	{Name: "route.vpc_peering_connection_id", DeepField: "Description.RouteTable.Routes.VpcPeeringConnectionId"},   //TODO-fix this mapping
}
var awsFmsPolicy = []Mapping{
	{Name: "arn", Steampipe: "akas"},
	{Name: "arn", Steampipe: "arn"},
	{Name: "delete_all_policy_resources", DeepField: "Description.Policy.ResourceType"},                              //TODO-fix this mapping
	{Name: "delete_unused_fm_managed_resources", DeepField: "Description.Policy.DeleteUnusedFMManagedResources"},     //TODO-fix this mapping
	{Name: "description", DeepField: "Description.Tags.Key"},                                                         //TODO-fix this mapping
	{Name: "exclude_resource_tags", DeepField: "Metadata.ResourceType"},                                              //TODO-fix this mapping
	{Name: "exclude_map.account", DeepField: "Metadata.AccountID"},                                                   //TODO-fix this mapping
	{Name: "exclude_map.orgunit", DeepField: "Metadata.Region"},                                                      //TODO-fix this mapping
	{Name: "include_map.account", DeepField: "Metadata.AccountID"},                                                   //TODO-fix this mapping
	{Name: "include_map.orgunit", DeepField: "Metadata.Region"},                                                      //TODO-fix this mapping
	{Name: "name", DeepField: "Metadata.Name"},                                                                       //TODO-fix this mapping
	{Name: "policy_update_token", DeepField: "Description.Policy.RemediationEnabled"},                                //TODO-fix this mapping
	{Name: "remediation_enabled", DeepField: "Description.Policy.RemediationEnabled"},                                //TODO-fix this mapping
	{Name: "resource_tags", DeepField: "ResourceJobID"},                                                              //TODO-fix this mapping
	{Name: "resource_type", DeepField: "ResourceType"},                                                               //TODO-fix this mapping
	{Name: "resource_type_list", DeepField: "ResourceType"},                                                          //TODO-fix this mapping
	{Name: "security_service_policy_data.managed_service_data", DeepField: "Description.Policy.SecurityServiceType"}, //TODO-fix this mapping
	{Name: "security_service_policy_data.type", DeepField: "Description.Policy.PolicyName"},                          //TODO-fix this mapping
}
var awsInspectorAssessmenttemplate = []Mapping{
	{Name: "arn", Steampipe: "arn"},
	{Name: "arn", Steampipe: "akas"},
	{Name: "name", Steampipe: "name"},
	{Name: "rules_package_arns", Steampipe: "rules_package_arns"},
	{Name: "duration", DeepField: "Description.AssessmentTemplate.DurationInSeconds"},            //TODO-fix this mapping
	{Name: "event_subscription.event", DeepField: "Description.EventSubscriptions.ResourceArn"},  //TODO-fix this mapping
	{Name: "event_subscription.topic_arn", DeepField: "Description.EventSubscriptions.TopicArn"}, //TODO-fix this mapping
	{Name: "target_arn", DeepField: "Metadata.Partition"},                                        //TODO-fix this mapping
}
