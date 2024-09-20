# https://github.com/elasticsearch-dump/elasticsearch-dump

#aws s3 cp s3://opengovernance-demo-export/es_backup /tmp/es_backup --recursive
aws s3 cp s3://opengovernance-demo-export/postgres /tmp/postgres --recursive

NEW_ELASTICSEARCH_ADDRESS="https://${ELASTICSEARCH_USERNAME}:${ELASTICSEARCH_PASSWORD}@${ELASTICSEARCH_ADDRESS#https://}"

DIR_PATH="/tmp/es_backup"

echo "$POSTGRESQL_HOST"
echo "$POSTGRESQL_PORT"
echo "$POSTGRESQL_USERNAME"
echo "$POSTGRESQL_PASSWORD"

PGPASSWORD="$POSTGRESQL_PASSWORD" psql --host="$POSTGRESQL_HOST" --port="$POSTGRESQL_PORT" --username "$POSTGRESQL_USERNAME" --dbname "pennywise" < /tmp/postgres/pennywise.sql
PGPASSWORD="$POSTGRESQL_PASSWORD" psql --host="$POSTGRESQL_HOST" --port="$POSTGRESQL_PORT" --username "$POSTGRESQL_USERNAME" --dbname "workspace" < /tmp/postgres/workspace.sql
PGPASSWORD="$POSTGRESQL_PASSWORD" psql --host="$POSTGRESQL_HOST" --port="$POSTGRESQL_PORT" --username "$POSTGRESQL_USERNAME" --dbname "auth" < /tmp/postgres/auth.sql
PGPASSWORD="$POSTGRESQL_PASSWORD" psql --host="$POSTGRESQL_HOST" --port="$POSTGRESQL_PORT" --username "$POSTGRESQL_USERNAME" --dbname "migrator" < /tmp/postgres/migrator.sql
PGPASSWORD="$POSTGRESQL_PASSWORD" psql --host="$POSTGRESQL_HOST" --port="$POSTGRESQL_PORT" --username "$POSTGRESQL_USERNAME" --dbname "describe" < /tmp/postgres/describe.sql
PGPASSWORD="$POSTGRESQL_PASSWORD" psql --host="$POSTGRESQL_HOST" --port="$POSTGRESQL_PORT" --username "$POSTGRESQL_USERNAME" --dbname "onboard" < /tmp/postgres/onboard.sql
PGPASSWORD="$POSTGRESQL_PASSWORD" psql --host="$POSTGRESQL_HOST" --port="$POSTGRESQL_PORT" --username "$POSTGRESQL_USERNAME" --dbname "inventory" < /tmp/postgres/inventory.sql
PGPASSWORD="$POSTGRESQL_PASSWORD" psql --host="$POSTGRESQL_HOST" --port="$POSTGRESQL_PORT" --username "$POSTGRESQL_USERNAME" --dbname "compliance" < /tmp/postgres/compliance.sql
PGPASSWORD="$POSTGRESQL_PASSWORD" psql --host="$POSTGRESQL_HOST" --port="$POSTGRESQL_PORT" --username "$POSTGRESQL_USERNAME" --dbname "metadata" < /tmp/postgres/metadata.sql

#find "$DIR_PATH" -maxdepth 1 -type f | while IFS= read -r file; do
#    file_name=$(basename "$file")
#
#    if [ "${file_name#map_}" = "$file_name" ]; then
#        NODE_TLS_REJECT_UNAUTHORIZED=0 elasticdump \
#          --input="/tmp/es_backup/map_$file_name" \
#          --output="$NEW_ELASTICSEARCH_ADDRESS/$file_name" \
#          --type=mapping
#        NODE_TLS_REJECT_UNAUTHORIZED=0 elasticdump \
#          --input="/tmp/es_backup/$file_name" \
#          --output="$NEW_ELASTICSEARCH_ADDRESS/$file_name" \
#          --type=data
#    fi
#done

rm -rf /tmp/es_backup /tmp/postgres