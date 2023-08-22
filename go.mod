module github.com/kaytu-io/kaytu-engine

go 1.21

require (
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.3.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources v1.1.1
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription v1.1.0
	github.com/aws/aws-sdk-go v1.44.153
	github.com/aws/aws-sdk-go-v2 v1.20.0
	github.com/aws/aws-sdk-go-v2/credentials v1.13.24
	github.com/aws/aws-sdk-go-v2/service/iam v1.18.9
	github.com/aws/aws-sdk-go-v2/service/organizations v1.16.8
	github.com/aws/aws-sdk-go-v2/service/s3 v1.27.1
	github.com/aws/aws-sdk-go-v2/service/sts v1.19.0
	github.com/aws/smithy-go v1.14.0
	github.com/brpaz/echozap v1.1.3
	github.com/confluentinc/confluent-kafka-go/v2 v2.1.1
	github.com/coreos/go-oidc/v3 v3.1.0
	github.com/elastic/go-elasticsearch/v7 v7.17.10
	github.com/envoyproxy/go-control-plane v0.11.1-0.20230524094728-9239064ad72f
	github.com/fluxcd/helm-controller/api v0.21.0
	github.com/fluxcd/pkg/apis/meta v0.13.0
	github.com/go-errors/errors v1.4.2
	github.com/go-git/go-git/v5 v5.6.1
	github.com/go-redis/cache/v8 v8.4.3
	github.com/go-redis/redis/v8 v8.11.5
	github.com/gogo/googleapis v1.4.1
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/google/uuid v1.3.0
	github.com/haoel/downsampling v0.0.0-20221012062717-1132fe8afe24
	github.com/kaytu-io/kaytu-aws-describer v0.0.55
	github.com/kaytu-io/kaytu-azure-describer v0.0.68
	github.com/kaytu-io/kaytu-util v0.0.0-20230822101604-7d81664d74f9
	github.com/kaytu-io/terraform-package v0.0.0-20230520063842-ebccc514e5a1
	github.com/labstack/echo-contrib v0.15.0
	github.com/labstack/echo/v4 v4.10.2
	github.com/labstack/gommon v0.4.0
	github.com/lib/pq v1.10.3
	github.com/microsoft/kiota-abstractions-go v0.9.1
	github.com/microsoft/kiota-authentication-azure-go v0.4.1
	github.com/microsoftgraph/msgraph-sdk-go v0.37.0
	github.com/ory/dockertest/v3 v3.10.0
	github.com/projectcontour/contour v1.22.0
	github.com/prometheus/client_golang v1.15.1
	github.com/sony/sonyflake v1.1.0
	github.com/spf13/cobra v1.7.0
	github.com/streadway/amqp v1.0.0
	github.com/stretchr/testify v1.8.3
	github.com/swaggo/echo-swagger v1.3.0
	github.com/swaggo/swag v1.16.1
	github.com/turbot/steampipe-plugin-sdk/v5 v5.5.1
	go.uber.org/zap v1.24.0
	golang.org/x/net v0.11.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230629202037-9506855d4529
	google.golang.org/grpc v1.56.1
	gopkg.in/go-playground/validator.v9 v9.31.0
	gorm.io/datatypes v1.1.0
	gorm.io/gorm v1.25.1
	k8s.io/api v0.27.4
	k8s.io/apiextensions-apiserver v0.27.2
	k8s.io/apimachinery v0.27.4
	sigs.k8s.io/controller-runtime v0.15.0
)

require (
	cloud.google.com/go v0.110.0 // indirect
	cloud.google.com/go/compute v1.19.1 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/iam v0.13.0 // indirect
	cloud.google.com/go/longrunning v0.4.1 // indirect
	cloud.google.com/go/storage v1.29.0 // indirect
	github.com/Azure/azure-pipeline-go v0.2.3 // indirect
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.7.0-beta.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.3.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/analysisservices/armanalysisservices v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/apimanagement/armapimanagement v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appconfiguration/armappconfiguration v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/applicationinsights/armapplicationinsights v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice v1.0.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2 v2.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/automation/armautomation v0.8.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch v1.2.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/blueprint/armblueprint v0.6.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/botservice/armbotservice v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cdn/armcdn v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cognitiveservices/armcognitiveservices v1.4.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4 v4.2.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerinstance/armcontainerinstance v1.0.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry v1.0.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice v1.0.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2 v2.5.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/costmanagement/armcostmanagement v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/databoxedge/armdataboxedge v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/databricks/armdatabricks v0.7.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/datafactory/armdatafactory/v2 v2.0.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/datalake-analytics/armdatalakeanalytics v0.7.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/datalake-store/armdatalakestore v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/datamigration/armdatamigration v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dataprotection/armdataprotection v1.0.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/desktopvirtualization/armdesktopvirtualization v1.0.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/deviceprovisioningservices/armdeviceprovisioningservices v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/devtestlabs/armdevtestlabs v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/eventgrid/armeventgrid/v2 v2.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/eventhub/armeventhub v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/frontdoor/armfrontdoor v1.2.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/guestconfiguration/armguestconfiguration v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/hdinsight/armhdinsight v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/healthcareapis/armhealthcareapis v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/hybridcompute/armhybridcompute v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/hybridcontainerservice/armhybridcontainerservice v0.2.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/hybridkubernetes/armhybridkubernetes v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/iothub/armiothub v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault v1.2.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/kubernetesconfiguration/armkubernetesconfiguration v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/kusto/armkusto v1.3.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/logic/armlogic v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/machinelearning/armmachinelearning v1.0.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/managementgroups/armmanagementgroups v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mariadb/armmariadb v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor v0.10.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysql v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysqlflexibleservers v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/netapp/armnetapp/v2 v2.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork v1.0.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2 v2.2.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/operationalinsights/armoperationalinsights/v2 v2.0.0-beta.3 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresql v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/powerbidedicated/armpowerbidedicated v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/purview/armpurview v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservices v1.4.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis/v2 v2.3.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redisenterprise/armredisenterprise v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph v0.8.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armlinks v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armlocks v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy v0.7.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions v1.2.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/search/armsearch v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/security/armsecurity v0.11.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/servicebus/armservicebus v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/servicefabric/armservicefabric v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/signalr/armsignalr v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sqlvirtualmachine/armsqlvirtualmachine v0.9.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage v1.3.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storagecache/armstoragecache/v2 v2.0.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storagesync/armstoragesync v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/streamanalytics/armstreamanalytics v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/synapse/armsynapse v0.7.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/timeseriesinsights/armtimeseriesinsights v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/virtualmachineimagebuilder/armvirtualmachineimagebuilder v1.2.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.0.0 // indirect
	github.com/Azure/azure-storage-blob-go v0.14.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.24 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.23 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.9 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.4 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.0.0 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/Microsoft/go-winio v0.6.0 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20230217124315-7d5c6f04bbb8 // indirect
	github.com/XiaoMi/pegasus-go-client v0.0.0-20210427083443-f3b6b08bc4c2 // indirect
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/acomagu/bufpipe v1.0.4 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/allegro/bigcache/v3 v3.1.0 // indirect
	github.com/apparentlymart/go-cidr v1.1.0 // indirect
	github.com/apparentlymart/go-textseg/v13 v13.0.0 // indirect
	github.com/apparentlymart/go-versions v1.0.1 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.4.9 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.18.25 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.37 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.31 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.0.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/accessanalyzer v1.16.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/account v1.10.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/acm v1.14.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/acmpca v1.21.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/amp v1.15.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/amplify v1.11.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.15.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/apigatewayv2 v1.12.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/appconfig v1.13.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/applicationautoscaling v1.15.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/applicationinsights v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/appstream v1.17.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/athena v1.31.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/auditmanager v1.20.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.23.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/backup v1.18.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/batch v1.19.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/cloudcontrol v1.10.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.23.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.20.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/cloudsearch v1.13.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/cloudtrail v1.21.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.21.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.23.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/codeartifact v1.13.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/codebuild v1.19.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/codecommit v1.13.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/codedeploy v1.15.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/codepipeline v1.13.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/codestar v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/configservice v1.28.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/costexplorer v1.19.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/databasemigrationservice v1.21.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/dax v1.11.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/directconnect v1.18.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/directoryservice v1.15.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/dlm v1.15.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/docdb v1.19.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/drs v1.9.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.15.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.13.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.74.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecr v1.17.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.13.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecs v1.21.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/efs v1.17.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/eks v1.26.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.22.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk v1.14.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.14.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.18.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/elasticsearchservice v1.16.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/emr v1.20.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/eventbridge v1.16.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/firehose v1.16.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/fms v1.22.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/fsx v1.24.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/glacier v1.13.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/globalaccelerator v1.15.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/glue v1.38.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/grafana v1.9.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/guardduty v1.15.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/health v1.15.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/identitystore v1.15.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/imagebuilder v1.23.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/inspector v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/inspector2 v1.16.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.9.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.1.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.7.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.27 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.13.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/kafka v1.18.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/keyspaces v1.1.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.15.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/kinesisanalyticsv2 v1.14.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/kinesisvideo v1.15.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/kms v1.21.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/lambda v1.26.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/lightsail v1.26.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/macie2 v1.27.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/mediastore v1.13.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/memorydb v1.12.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/mgn v1.18.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/mq v1.13.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/mwaa v1.13.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/neptune v1.18.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/networkfirewall v1.20.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/oam v1.1.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/opensearch v1.10.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/opsworkscm v1.15.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/pinpoint v1.19.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/pipes v1.2.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/pricing v1.16.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/ram v1.18.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/rds v1.26.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/redshift v1.26.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/redshiftserverless v1.2.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/resourceexplorer2 v1.2.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi v1.14.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/route53 v1.24.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/route53domains v1.14.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/route53resolver v1.15.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3control v1.21.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sagemaker v1.48.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.16.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/securityhub v1.26.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/securitylake v1.3.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/serverlessapplicationrepository v1.12.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/servicecatalog v1.20.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/servicediscovery v1.22.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/servicequotas v1.14.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/ses v1.14.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/sesv2 v1.16.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sfn v1.17.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/shield v1.18.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/simspaceweaver v1.2.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sns v1.17.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.19.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssm v1.36.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.12.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssoadmin v1.15.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.14.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/storagegateway v1.18.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/support v1.11.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/synthetics v1.9.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/waf v1.11.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/wafregional v1.12.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/wafv2 v1.22.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/wellarchitected v1.20.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/workspaces v1.23.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bgentry/go-netrc v0.0.0-20140422174119-9fd32a8b3d3d // indirect
	github.com/bmatcuk/doublestar v1.1.5 // indirect
	github.com/bradfitz/gomemcache v0.0.0-20221031212613-62deef7fc822 // indirect
	github.com/btubbs/datetime v0.1.1 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cjlapao/common-go v0.0.25 // indirect
	github.com/cloudflare/circl v1.3.3 // indirect
	github.com/cncf/xds/go v0.0.0-20230607035331-e9ce68804cb4 // indirect
	github.com/containerd/continuity v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/danwakefield/fnmatch v0.0.0-20160403171240-cbb64ac3d964 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgraph-io/ristretto v0.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/docker/cli v20.10.17+incompatible // indirect
	github.com/docker/docker v24.0.4+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/eko/gocache/v3 v3.1.2 // indirect
	github.com/emicklei/go-restful/v3 v3.9.0 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/envoyproxy/protoc-gen-validate v0.10.1 // indirect
	github.com/evanphx/json-patch v4.12.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/fatih/color v1.15.0 // indirect
	github.com/fluxcd/pkg/apis/kustomize v0.3.3 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/gertd/go-pluralize v0.2.1 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/globocom/echo-prometheus v0.1.2 // indirect
	github.com/go-git/gcfg v1.5.0 // indirect
	github.com/go-git/go-billy/v5 v5.4.1 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/spec v0.20.9 // indirect
	github.com/go-openapi/swag v0.22.4 // indirect
	github.com/go-playground/locales v0.14.0 // indirect
	github.com/go-playground/universal-translator v0.18.0 // indirect
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/gocarina/gocsv v0.0.0-20211203214250-4735fba0c1d9 // indirect
	github.com/gofrs/uuid v4.2.0+incompatible // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang/glog v1.1.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/gnostic v0.5.7-v3refs // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.3 // indirect
	github.com/googleapis/gax-go/v2 v2.7.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.11.3 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-getter v1.7.1 // indirect
	github.com/hashicorp/go-hclog v1.4.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.4.10 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.2 // indirect
	github.com/hashicorp/go-safetemp v1.0.0 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.6.0 // indirect
	github.com/hashicorp/hcl/v2 v2.16.2 // indirect
	github.com/hashicorp/terraform-registry-address v0.0.0-20220623143253-7d51757b572c // indirect
	github.com/hashicorp/terraform-svchost v0.1.0 // indirect
	github.com/hashicorp/yamux v0.0.0-20211028200310-0bc27b27de87 // indirect
	github.com/iancoleman/strcase v0.2.0 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.13.0 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.3.1 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgtype v1.12.0 // indirect
	github.com/jackc/pgx/v4 v4.17.2 // indirect
	github.com/jackc/pgx/v5 v5.3.0 // indirect
	github.com/jackc/puddle v1.3.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kaytu-io/steampipe-plugin-aws v0.0.0-20230813122148-8f0e3fd85341 // indirect
	github.com/kaytu-io/steampipe-plugin-azure v0.0.0-20230811120400-81b9fefee739 // indirect
	github.com/kaytu-io/steampipe-plugin-azuread v0.0.0-20230811120442-8d8b60f7749e // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/compress v1.15.11 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/manicminer/hamilton v0.44.0 // indirect
	github.com/manicminer/hamilton-autorest v0.3.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-ieproxy v0.0.1 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/microsoft/kiota-http-go v0.7.0 // indirect
	github.com/microsoft/kiota-serialization-json-go v0.5.6 // indirect
	github.com/microsoft/kiota-serialization-text-go v0.4.2 // indirect
	github.com/microsoftgraph/msgraph-sdk-go-core v0.28.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/term v0.0.0-20221205130635-1aeaba878587 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.3-0.20211202183452-c5a74bcca799 // indirect
	github.com/opencontainers/runc v1.1.5 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pegasus-kv/thrift v0.13.0 // indirect
	github.com/pganalyze/pg_query_go/v4 v4.2.1 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/common v0.42.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/sendgrid/rest v2.6.9+incompatible // indirect
	github.com/sendgrid/sendgrid-go v3.12.0+incompatible // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	github.com/sethvargo/go-retry v0.2.4 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/skeema/knownhosts v1.1.0 // indirect
	github.com/spf13/afero v1.9.2 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stevenle/topsort v0.2.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/swaggo/files v0.0.0-20210815190702-a29dd2bc99b2 // indirect
	github.com/tkrajina/go-reflector v0.5.6 // indirect
	github.com/tombuildsstuff/giovanni v0.18.0 // indirect
	github.com/turbot/go-kit v0.6.0 // indirect
	github.com/uber/jaeger-client-go v2.30.0+incompatible // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	github.com/ulikunitz/xz v0.5.11 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/vmihailenco/go-tinylfu v0.2.2 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.4 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	github.com/zclconf/go-cty v1.12.1 // indirect
	github.com/zclconf/go-cty-yaml v1.0.3 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/otel v1.16.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/internal/retry v1.16.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric v0.39.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.39.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.16.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.16.0 // indirect
	go.opentelemetry.io/otel/metric v1.16.0 // indirect
	go.opentelemetry.io/otel/sdk v1.16.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.39.0 // indirect
	go.opentelemetry.io/otel/trace v1.16.0 // indirect
	go.opentelemetry.io/proto/otlp v0.19.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	golang.org/x/crypto v0.10.0 // indirect
	golang.org/x/exp v0.0.0-20230522175609-2e198f4a06a1 // indirect
	golang.org/x/mod v0.11.0 // indirect
	golang.org/x/oauth2 v0.7.0 // indirect
	golang.org/x/sync v0.3.0 // indirect
	golang.org/x/sys v0.9.0 // indirect
	golang.org/x/term v0.9.0 // indirect
	golang.org/x/text v0.10.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.10.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	gomodules.xyz/jsonpatch/v2 v2.3.0 // indirect
	google.golang.org/api v0.114.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230525234025-438c736192d0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230525234020-1aefcd67740a // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/driver/mysql v1.4.4 // indirect
	gorm.io/driver/postgres v1.5.0 // indirect
	gorm.io/plugin/prometheus v0.0.0-20230504115745-1aec2356381b // indirect
	k8s.io/client-go v0.27.4 // indirect
	k8s.io/component-base v0.27.2 // indirect
	k8s.io/klog/v2 v2.90.1 // indirect
	k8s.io/kube-openapi v0.0.0-20230501164219-8b0f38b5fd1f // indirect
	k8s.io/utils v0.0.0-20230209194617-a36077c30491 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)
