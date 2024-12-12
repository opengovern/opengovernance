#!/bin/bash


export PGPASSWORD="postgres"

pg_dump integration --host "localhost" --port "5432" --username "postgres" --no-password --format=t --blobs --no-owner --no-privileges --no-comments --no-subscriptions --verbose "integration" --exclude-table=integrations --exclude-table=integration_type_setups > integration.bak
pg_dump auth --host "localhost" --port "5432" --username "postgres" --no-password --format=t --blobs --no-owner --no-privileges --no-comments --no-subscriptions --verbose "auth" > auth.bak
pg_dump compliance --host "localhost" --port "5432" --username "postgres" --no-password --format=t --blobs --no-owner --no-privileges --no-comments --no-subscriptions --verbose "compliance" > compliance.bak
pg_dump dex --host "localhost" --port "5432" --username "postgres" --no-password --format=t --blobs --no-owner --no-privileges --no-comments --no-subscriptions --verbose "dex" > dex.bak
pg_dump inventory --host "localhost" --port "5432" --username "postgres" --no-password --format=t --blobs --no-owner --no-privileges --no-comments --no-subscriptions --verbose "inventory" > inventory.bak
pg_dump metadata --host "localhost" --port "5432" --username "postgres" --no-password --format=t --blobs --no-owner --no-privileges --no-comments --no-subscriptions --verbose "metadata" > metadata.bak

