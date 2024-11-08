#!/bin/bash
set -e

dt=$(date '+%d/%m/%Y %H:%M:%S');
echo "$dt - Running init script the 1st time Primary PostgreSql container is created...";

pennywiseDatabaseName="pennywise"
pennywiseUserName="pennywise_service"

workspaceDatabaseName="workspace"
workspaceUserName="workspace_service"

authDatabaseName="auth"
authUserName="auth_service"

subscriptionDatabaseName="subscription"
subscriptionUserName="subscription_service"

informationDatabaseName="information"
informationUserName="information_service"

dexDatabaseName="dex"
dexUserName="dex_service"

exporterUserName="postgres_exporter"
steampipeUserName="steampipe_user"

migratorDatabaseName="migrator"
migratorUserName="migrator_worker"

describeDatabaseName="describe"
describeUserName="describe_scheduler"

assistantDatabaseName="assistant"
assistantUserName="assistant_service"

onboardDatabaseName="onboard"
onboardUserName="onboard_service"

policyDatabaseName="policy"
policyUserName="policy_service"

inventoryDatabaseName="inventory"
inventoryUserName="inventory_service"

complianceDatabaseName="compliance"
complianceUserName="compliance_service"

metadataDatabaseName="metadata"
metadataUserName="metadata_service"

reporterDatabaseName="reporter"
reporterUserName="reporter_service"

alertingDatabaseName="alerting"
alertingUserName="alerting_service"

integrationDatabaseName="integration"
integrationUserName="integration_service"

echo "$dt - Running: psql -v ON_ERROR_STOP=1 --username postgres --dbname postgres ...";

PGPASSWORD="postgres" psql -v ON_ERROR_STOP=1 --username "postgres" --dbname "postgres" <<-EOSQL

   

CREATE DATABASE $workspaceDatabaseName;
CREATE USER $workspaceUserName WITH PASSWORD '$POSTGRES_WORKSPACE_DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE "$workspaceDatabaseName" to $workspaceUserName;

\c "$workspaceDatabaseName"
CREATE EXTENSION "uuid-ossp" WITH SCHEMA public;
GRANT ALL ON SCHEMA public TO $workspaceUserName;


CREATE DATABASE $subscriptionDatabaseName;
CREATE USER $subscriptionUserName WITH PASSWORD '$POSTGRES_SUBSCRIPTION_DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE "$subscriptionDatabaseName" to $subscriptionUserName;

\c "$subscriptionDatabaseName"
CREATE EXTENSION "uuid-ossp" WITH SCHEMA public;
GRANT ALL ON SCHEMA public TO $subscriptionUserName;

CREATE DATABASE $pennywiseDatabaseName;
CREATE USER $pennywiseUserName WITH PASSWORD '$POSTGRES_PENNYWISE_DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE "$pennywiseDatabaseName" to $pennywiseUserName;

\c "$pennywiseDatabaseName"
CREATE EXTENSION "uuid-ossp" WITH SCHEMA public;
GRANT ALL ON SCHEMA public TO $pennywiseUserName;

CREATE DATABASE $informationDatabaseName;
CREATE USER $informationUserName WITH PASSWORD '$POSTGRES_INFORMATION_DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE "$informationDatabaseName" to $informationUserName;

\c "$informationDatabaseName"
CREATE EXTENSION "uuid-ossp" WITH SCHEMA public;
GRANT ALL ON SCHEMA public TO $informationUserName;

CREATE DATABASE $dexDatabaseName;
CREATE USER $dexUserName WITH PASSWORD '$DEX_DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE "$dexDatabaseName" to $dexUserName;

\c "$dexDatabaseName"
CREATE EXTENSION "uuid-ossp" WITH SCHEMA public;
GRANT ALL ON SCHEMA public TO $dexUserName;

CREATE DATABASE $describeDatabaseName;
CREATE USER $describeUserName WITH PASSWORD '$POSTGRES_DESCRIBE_DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE "$describeDatabaseName" to $describeUserName;

\c $describeDatabaseName
CREATE EXTENSION "uuid-ossp" WITH SCHEMA public;
CREATE EXTENSION citext;
GRANT ALL ON SCHEMA public TO $describeUserName;

CREATE DATABASE $onboardDatabaseName;
CREATE USER $onboardUserName WITH PASSWORD '$POSTGRES_ONBOARD_DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE "$onboardDatabaseName" to $onboardUserName;

\c $onboardDatabaseName
CREATE EXTENSION "uuid-ossp" WITH SCHEMA public;
CREATE EXTENSION citext;
GRANT ALL ON SCHEMA public TO $onboardUserName;

CREATE DATABASE $policyDatabaseName;
CREATE USER $policyUserName WITH PASSWORD '$POSTGRES_POLICY_DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE "$policyDatabaseName" to $policyUserName;

\c $policyDatabaseName
CREATE EXTENSION "uuid-ossp" WITH SCHEMA public;
CREATE EXTENSION citext;
GRANT ALL ON SCHEMA public TO $policyUserName;

CREATE DATABASE $inventoryDatabaseName ;
CREATE USER $inventoryUserName WITH PASSWORD '$POSTGRES_INVENTORY_DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE "$inventoryDatabaseName" to $inventoryUserName;

\c $inventoryDatabaseName
CREATE EXTENSION "uuid-ossp" WITH SCHEMA public;
CREATE EXTENSION citext;
GRANT ALL ON SCHEMA public TO $inventoryUserName;

CREATE DATABASE $assistantDatabaseName ;
CREATE USER $assistantUserName WITH PASSWORD '$POSTGRES_ASSISTANT_DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE "$assistantDatabaseName" to $assistantUserName;

\c $assistantDatabaseName
CREATE EXTENSION "uuid-ossp" WITH SCHEMA public;
CREATE EXTENSION citext;
GRANT ALL ON SCHEMA public TO $assistantUserName;

CREATE DATABASE $complianceDatabaseName ;
CREATE USER $complianceUserName WITH PASSWORD '$POSTGRES_COMPLIANCE_DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE "$complianceDatabaseName" to $complianceUserName;

\c $complianceDatabaseName
CREATE EXTENSION "uuid-ossp" WITH SCHEMA public;
CREATE EXTENSION citext;
GRANT ALL ON SCHEMA public TO $complianceUserName;

CREATE DATABASE $authDatabaseName;
CREATE USER $authUserName WITH PASSWORD '$POSTGRES_AUTH_DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE "$authDatabaseName" to $authUserName;

\c "$authDatabaseName"
CREATE EXTENSION "uuid-ossp" WITH SCHEMA public;
CREATE EXTENSION citext;
GRANT ALL ON SCHEMA public TO $authUserName;

CREATE DATABASE $metadataDatabaseName;
CREATE USER $metadataUserName WITH PASSWORD '$POSTGRES_METADATA_DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE "$metadataDatabaseName" to $metadataUserName;

\c "$metadataDatabaseName"
CREATE EXTENSION "uuid-ossp" WITH SCHEMA public;
CREATE EXTENSION citext;
GRANT ALL ON SCHEMA public TO $metadataUserName;

CREATE DATABASE $reporterDatabaseName;
CREATE USER $reporterUserName WITH PASSWORD '$POSTGRES_REPORTER_DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE "$reporterDatabaseName" to $reporterUserName;

\c "$reporterDatabaseName"
CREATE EXTENSION "uuid-ossp" WITH SCHEMA public;
CREATE EXTENSION citext;
GRANT ALL ON SCHEMA public TO $reporterUserName;

CREATE DATABASE $alertingDatabaseName;
CREATE USER $alertingUserName WITH PASSWORD '$POSTGRES_ALERTING_DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE "$alertingDatabaseName" to $alertingUserName;

\c "$alertingDatabaseName"
CREATE EXTENSION "uuid-ossp" WITH SCHEMA public;
CREATE EXTENSION citext;
GRANT ALL ON SCHEMA public TO $alertingUserName;

CREATE DATABASE $integrationDatabaseName;
CREATE USER $integrationUserName WITH PASSWORD '$POSTGRES_INTEGRATION_DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE "$integrationDatabaseName" to $integrationUserName;

\c "$integrationDatabaseName"
CREATE EXTENSION "uuid-ossp" WITH SCHEMA public;
CREATE EXTENSION citext;
GRANT ALL ON SCHEMA public TO $integrationUserName;

CREATE DATABASE $migratorDatabaseName;
CREATE USER $migratorUserName WITH PASSWORD '$POSTGRES_MIGRATOR_DB_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE "$migratorDatabaseName" to $migratorUserName;

\c "$migratorDatabaseName"
CREATE EXTENSION "uuid-ossp" WITH SCHEMA public;
CREATE EXTENSION citext;
GRANT ALL ON SCHEMA public TO $migratorUserName;
GRANT pg_read_all_data TO $workspaceUserName;
GRANT pg_write_all_data TO $workspaceUserName;
GRANT ALL ON SCHEMA public TO $workspaceUserName;
GRANT pg_read_all_data TO $metadataUserName;
GRANT pg_write_all_data TO $metadataUserName;
GRANT ALL ON SCHEMA public TO $metadataUserName;
GRANT pg_read_all_data TO $complianceUserName;
GRANT pg_write_all_data TO $complianceUserName;
GRANT ALL ON SCHEMA public TO $complianceUserName;

CREATE USER $steampipeUserName WITH PASSWORD '$POSTGRES_STEAMPIPE_USER_PASSWORD';

\connect "$complianceDatabaseName";
GRANT pg_read_all_data TO $migratorUserName;
GRANT pg_write_all_data TO $migratorUserName;
GRANT ALL ON SCHEMA public TO $migratorUserName;
\connect "$metadataDatabaseName";
GRANT pg_read_all_data TO $migratorUserName;
GRANT pg_write_all_data TO $migratorUserName;
GRANT ALL ON SCHEMA public TO $migratorUserName;
\connect "$authDatabaseName";
GRANT pg_read_all_data TO $migratorUserName;
GRANT pg_write_all_data TO $migratorUserName;
GRANT ALL ON SCHEMA public TO $migratorUserName; 
\connect "$onboardDatabaseName";
GRANT pg_read_all_data TO $migratorUserName;
GRANT pg_write_all_data TO $migratorUserName;
GRANT ALL ON SCHEMA public TO $migratorUserName;
GRANT pg_read_all_data TO $steampipeUserName;
\connect "$inventoryDatabaseName";
GRANT pg_read_all_data TO $migratorUserName;
GRANT pg_write_all_data TO $migratorUserName;
GRANT ALL ON SCHEMA public TO $migratorUserName;
GRANT pg_read_all_data TO $steampipeUserName;
\connect "$integrationDatabaseName";
GRANT pg_read_all_data TO $migratorUserName;
GRANT pg_write_all_data TO $migratorUserName;
GRANT ALL ON SCHEMA public TO $migratorUserName;
GRANT pg_read_all_data TO $steampipeUserName;


\connect "postgres";
CREATE EXTENSION pg_stat_statements;
CREATE USER $exporterUserName WITH PASSWORD '$POSTGRES_EXPORTER_PASSWORD';
GRANT pg_monitor TO $exporterUserName;

ALTER USER $migratorUserName WITH SUPERUSER;

EOSQL

echo "$dt - Init script is completed";
export PGPASSWORD='postgres'
set "PGPASSWORD=postgres"
PGPASSWORD="postgres"
pg_restore -h localhost -p 5432 -U postgres -d $authDatabaseName -v "$authDatabaseName.bak";
pg_restore -h localhost -p 5432 -U postgres -d $onboardDatabaseName -v "$onboardDatabaseName.bak";
pg_restore -h localhost -p 5432 -U postgres -d $inventoryDatabaseName -v "$inventoryDatabaseName.bak";
pg_restore -h localhost -p 5432 -U postgres -d $complianceDatabaseName -v "$complianceDatabaseName.bak";
pg_restore -h localhost -p 5432 -U postgres -d $metadataDatabaseName -v "$metadataDatabaseName.bak";
pg_restore -h localhost -p 5432 -U postgres -d $dexDatabaseName -v "$dexDatabaseName.bak";

echo "$dt - Restore is completed";  
