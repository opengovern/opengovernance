pg_dump --dbname="postgresql://$OCT_POSTGRESQL_USERNAME:$OCT_POSTGRESQL_PASSWORD@$OCT_POSTGRESQL_HOST:$POSTGRESQL_PORT/pennywise" > /tmp/postgres/pennywise.sql
pg_dump --dbname="postgresql://$OCT_POSTGRESQL_USERNAME:$OCT_POSTGRESQL_PASSWORD@$OCT_POSTGRESQL_HOST:$POSTGRESQL_PORT/workspace" > /tmp/postgres/workspace.sql
pg_dump --dbname="postgresql://$OCT_POSTGRESQL_USERNAME:$OCT_POSTGRESQL_PASSWORD@$OCT_POSTGRESQL_HOST:$POSTGRESQL_PORT/auth" --exclude-table api_keys --exclude-table users --exclude-table configurations > /tmp/postgres/auth.sql
pg_dump --dbname="postgresql://$POSTGRESQL_USERNAME:$POSTGRESQL_PASSWORD@$POSTGRESQL_HOST:$POSTGRESQL_PORT/migrator" > /tmp/postgres/migrator.sql
pg_dump --dbname="postgresql://$POSTGRESQL_USERNAME:$POSTGRESQL_PASSWORD@$POSTGRESQL_HOST:$POSTGRESQL_PORT/describe" > /tmp/postgres/describe.sql
pg_dump --dbname="postgresql://$POSTGRESQL_USERNAME:$POSTGRESQL_PASSWORD@$POSTGRESQL_HOST:$POSTGRESQL_PORT/onboard" > /tmp/postgres/onboard.sql
pg_dump --dbname="postgresql://$POSTGRESQL_USERNAME:$POSTGRESQL_PASSWORD@$POSTGRESQL_HOST:$POSTGRESQL_PORT/inventory" > /tmp/postgres/inventory.sql
pg_dump --dbname="postgresql://$POSTGRESQL_USERNAME:$POSTGRESQL_PASSWORD@$POSTGRESQL_HOST:$POSTGRESQL_PORT/compliance" > /tmp/postgres/compliance.sql
pg_dump --dbname="postgresql://$POSTGRESQL_USERNAME:$POSTGRESQL_PASSWORD@$POSTGRESQL_HOST:$POSTGRESQL_PORT/metadata" > /tmp/postgres/metadata.sql

aws s3 cp /tmp/postgres/ s3://opengovernance-demo-export/postgres --recursive --acl public-read

rm -rf /tmp/postgres
