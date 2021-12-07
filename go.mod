module gitlab.com/keibiengine/keibi-engine

go 1.16

require (
	github.com/Azure/azure-sdk-for-go v59.3.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.22
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.9
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/DataDog/zstd v1.5.0 // indirect
	github.com/Shopify/toxiproxy v2.1.4+incompatible // indirect
	github.com/armon/go-metrics v0.3.10 // indirect
	github.com/aws/aws-sdk-go-v2 v1.11.1
	github.com/aws/aws-sdk-go-v2/config v1.10.2
	github.com/aws/aws-sdk-go-v2/credentials v1.6.2
	github.com/aws/aws-sdk-go-v2/service/acm v1.9.1
	github.com/aws/aws-sdk-go-v2/service/applicationinsights v1.8.0
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.15.1
	github.com/aws/aws-sdk-go-v2/service/backup v1.9.1
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.11.1
	github.com/aws/aws-sdk-go-v2/service/cloudtrail v1.10.0
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.12.0
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.10.1
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.23.0
	github.com/aws/aws-sdk-go-v2/service/ecr v1.10.1
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.8.1
	github.com/aws/aws-sdk-go-v2/service/ecs v1.12.1
	github.com/aws/aws-sdk-go-v2/service/efs v1.10.1
	github.com/aws/aws-sdk-go-v2/service/eks v1.14.0
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.9.1
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.12.1
	github.com/aws/aws-sdk-go-v2/service/iam v1.13.1
	github.com/aws/aws-sdk-go-v2/service/kms v1.11.0
	github.com/aws/aws-sdk-go-v2/service/lambda v1.13.0
	github.com/aws/aws-sdk-go-v2/service/rds v1.12.1
	github.com/aws/aws-sdk-go-v2/service/redshift v1.15.0
	github.com/aws/aws-sdk-go-v2/service/route53 v1.14.1
	github.com/aws/aws-sdk-go-v2/service/route53resolver v1.10.1
	github.com/aws/aws-sdk-go-v2/service/s3 v1.19.1
	github.com/aws/aws-sdk-go-v2/service/s3control v1.14.1
	github.com/aws/aws-sdk-go-v2/service/ses v1.9.1
	github.com/aws/aws-sdk-go-v2/service/sesv2 v1.8.1
	github.com/aws/aws-sdk-go-v2/service/sns v1.12.0
	github.com/aws/aws-sdk-go-v2/service/sqs v1.12.1
	github.com/aws/aws-sdk-go-v2/service/ssm v1.16.0
	github.com/aws/aws-sdk-go-v2/service/sts v1.10.1
	github.com/aws/aws-sdk-go-v2/service/synthetics v1.9.1
	github.com/aws/aws-sdk-go-v2/service/wafregional v1.8.1
	github.com/aws/aws-sdk-go-v2/service/wafv2 v1.14.0
	github.com/aws/aws-sdk-go-v2/service/workspaces v1.10.1
	github.com/aws/smithy-go v1.9.0
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/eapache/go-resiliency v1.2.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/frankban/quicktest v1.14.0 // indirect
	github.com/go-openapi/spec v0.20.4 // indirect
	github.com/go-playground/universal-translator v0.18.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/uuid v1.3.0
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.0.0 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.0 // indirect
	github.com/hashicorp/go-secure-stdlib/mlock v0.1.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.2 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-version v1.3.0 // indirect
	github.com/hashicorp/vault/api v1.3.0 // indirect
	github.com/hashicorp/vault/api/auth/kubernetes v0.1.0 // indirect
	github.com/hashicorp/yamux v0.0.0-20211028200310-0bc27b27de87 // indirect
	github.com/labstack/echo/v4 v4.6.1
	github.com/labstack/gommon v0.3.1
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/spf13/cobra v1.2.1
	github.com/streadway/amqp v1.0.0
	github.com/swaggo/echo-swagger v1.1.4
	github.com/swaggo/files v0.0.0-20210815190702-a29dd2bc99b2 // indirect
	github.com/swaggo/swag v1.7.4 // indirect
	golang.org/x/crypto v0.0.0-20211202192323-5770296d904e // indirect
	golang.org/x/net v0.0.0-20211203184738-4852103109b8 // indirect
	golang.org/x/sys v0.0.0-20211204120058-94396e421777 // indirect
	golang.org/x/time v0.0.0-20211116232009-f0f3c7e86c11 // indirect
	golang.org/x/tools v0.1.7 // indirect
	google.golang.org/genproto v0.0.0-20211203200212-54befc351ae9 // indirect
	google.golang.org/grpc v1.42.0 // indirect
	gopkg.in/Shopify/sarama.v1 v1.20.1
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.31.0
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gorm.io/driver/postgres v1.2.2
	gorm.io/gorm v1.22.3
)
