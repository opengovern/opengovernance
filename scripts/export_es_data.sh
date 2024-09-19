# https://github.com/elasticsearch-dump/elasticsearch-dump

ELASTICSEARCH_ADDRESS="https://${ELASTICSEARCH_USERNAME}:${ELASTICSEARCH_PASSWORD}@${ELASTICSEARCH_ADDRESS#https://}"


NODE_TLS_REJECT_UNAUTHORIZED=0 multielasticdump \
  --direction=dump \
  --match='^.*$' \
  --input="$ELASTICSEARCH_ADDRESS" \
  --output=/tmp/es_backup

aws s3 cp /tmp/es_backup s3://opengovernance-demo-export/es_backup --recursive --acl public-read

rm -rf /tmp/es_backup
