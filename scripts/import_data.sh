# https://github.com/elasticsearch-dump/elasticsearch-dump

aws s3 cp s3://opengovernance-demo-export/es_backup /tmp/es_backup --recursive
aws s3 cp s3://opengovernance-demo-export/postgres /tmp/postgres --recursive

NEW_ELASTICSEARCH_ADDRESS="https://${ELASTICSEARCH_USERNAME}:${ELASTICSEARCH_PASSWORD}@${ELASTICSEARCH_ADDRESS#https://}"

curl -X GET "$ELASTICSEARCH_ADDRESS/_cat/indices?format=json" -u "$ELASTICSEARCH_USERNAME:$ELASTICSEARCH_PASSWORD" --insecure | jq -r '.[].index' | while read -r index; do
  if [ "$(echo "$index" | cut -c 1)" != "." ]; then
    NODE_TLS_REJECT_UNAUTHORIZED=0 elasticdump \
      --input="/tmp/es_backup/map_$index" \
      --output="$NEW_ELASTICSEARCH_ADDRESS/$index" \
      --type=mapping
    NODE_TLS_REJECT_UNAUTHORIZED=0 elasticdump \
      --input="/tmp/es_backup/$index" \
      --output="$NEW_ELASTICSEARCH_ADDRESS/$index" \
      --type=data
  fi
done

PGPASSWORD="$POSTGRESQL_PASSWORD"
psql --host="$POSTGRES_HOST" --port="$POSTGRES_PORT" --username "$POSTGRES_USER" --dbname "pennywise" < /tmp/postgres/pennywise.sql
psql --host="$POSTGRES_HOST" --port="$POSTGRES_PORT" --username "$POSTGRES_USER" --dbname "workspace" < /tmp/postgres/workspace.sql
psql --host="$POSTGRES_HOST" --port="$POSTGRES_PORT" --username "$POSTGRES_USER" --dbname "auth" < /tmp/postgres/auth.sql
psql --host="$POSTGRES_HOST" --port="$POSTGRES_PORT" --username "$POSTGRES_USER" --dbname "migrator" < /tmp/postgres/migrator.sql
psql --host="$POSTGRES_HOST" --port="$POSTGRES_PORT" --username "$POSTGRES_USER" --dbname "describe" < /tmp/postgres/describe.sql
psql --host="$POSTGRES_HOST" --port="$POSTGRES_PORT" --username "$POSTGRES_USER" --dbname "onboard" < /tmp/postgres/onboard.sql
psql --host="$POSTGRES_HOST" --port="$POSTGRES_PORT" --username "$POSTGRES_USER" --dbname "inventory" < /tmp/postgres/inventory.sql
psql --host="$POSTGRES_HOST" --port="$POSTGRES_PORT" --username "$POSTGRES_USER" --dbname "compliance" < /tmp/postgres/compliance.sql
psql --host="$POSTGRES_HOST" --port="$POSTGRES_PORT" --username "$POSTGRES_USER" --dbname "metadata" < /tmp/postgres/metadata.sql

rm -rf /tmp/es_backup /tmp/postgres