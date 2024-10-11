# https://github.com/elasticsearch-dump/elasticsearch-dump

curl -O "$DEMO_DATA_S3_URL"

openssl enc -d -aes-256-cbc -md md5 -pass pass:"$OPENSSL_PASSWORD" -base64 -in demo_data.tar.gz.enc -out demo_data.tar.gz
tar -xvf demo_data.tar.gz

echo "$POSTGRESQL_HOST"
echo "$POSTGRESQL_PORT"
echo "$POSTGRESQL_USERNAME"
echo "$POSTGRESQL_PASSWORD"


PGPASSWORD="$POSTGRESQL_PASSWORD" psql --host="$POSTGRESQL_HOST" --port="$POSTGRESQL_PORT" --username "$POSTGRESQL_USERNAME" --dbname "onboard" << EOF
INSERT INTO sources (
    id, source_id, name, email, type, description, lifecycle_state,
    asset_discovery_method, health_state, last_health_check_time,
    health_reason, asset_discovery, spend_discovery, creation_method,
    metadata, created_at, updated_at
)
VALUES
(
    '79302008-5d35-4a83-914c-20ecd80f4228', '941377156806', 'kaytu-test', NULL, 'AWS',
    'Auto onboarded account 941377156806', 'ONBOARD', 'scheduled', 'unhealthy',
    '2024-10-07 13:25:13.585823+00', 'Credential is not healthy',
    false, false, 'auto-onboard',
    '{"account_id": "941377156806", "account_name": "kaytu-test", "account_type": "organization_member", "account_organization": {"Id": "o-ng68d511a2", "Arn": "arn:aws:organizations::861370837605:organization/o-ng68d511a2", "FeatureSet": "ALL", "MasterAccountId": "861370837605", "MasterAccountArn": "arn:aws:organizations::861370837605:account/o-ng68d511a2/861370837605", "MasterAccountEmail": "gulegulzaradnan@gmail.com", "AvailablePolicyTypes": [{"Type": "SERVICE_CONTROL_POLICY", "Status": "ENABLED"}]}, "organization_account": {"Id": "941377156806", "Arn": "arn:aws:organizations::861370837605:account/o-ng68d511a2/941377156806", "Name": "kaytu-test", "Email": "gulegulzaradnan@icloud.com", "Status": "ACTIVE", "JoinedMethod": "CREATED", "JoinedTimestamp": "2024-08-26T20:57:34.382Z"}}',
    '2024-09-26 22:11:27.736433+00', '2024-10-07 13:25:13.585926+00'
),
(
    'e6cb0afa-e624-4ca7-8b47-fa9988831137', '861370837605', 'ADorigi', NULL, 'AWS',
    'Auto onboarded account 861370837605', 'ONBOARD', 'scheduled', 'unhealthy',
    '2024-10-11 14:41:53.27761+00', NULL,
    true, true, 'auto-onboard',
    '{"account_id": "861370837605", "account_name": "ADorigi", "account_type": "organization_manager", "account_organization": {"Id": "o-ng68d511a2", "Arn": "arn:aws:organizations::861370837605:organization/o-ng68d511a2", "FeatureSet": "ALL", "MasterAccountId": "861370837605", "MasterAccountArn": "arn:aws:organizations::861370837605:account/o-ng68d511a2/861370837605", "MasterAccountEmail": "gulegulzaradnan@gmail.com", "AvailablePolicyTypes": [{"Type": "SERVICE_CONTROL_POLICY", "Status": "ENABLED"}]}, "organization_account": {"Id": "861370837605", "Arn": "arn:aws:organizations::861370837605:account/o-ng68d511a2/861370837605", "Name": "ADorigi", "Email": "gulegulzaradnan@gmail.com", "Status": "ACTIVE", "JoinedMethod": "INVITED", "JoinedTimestamp": "2024-08-26T20:39:05.246Z"}}',
    '2024-09-26 22:11:27.262228+00', '2024-10-11 14:41:53.278288+00'
),
(
    '1c2a6b18-ac87-4f5e-a472-1e26f8704f29', '75b0a9a9-3222-4290-bdf9-56127d550563', 'Policy Testing Subscription', NULL, 'Azure',
    'Auto on-boarded subscription 75b0a9a9-3222-4290-bdf9-56127d550563', 'ONBOARD', 'scheduled', 'unhealthy',
    '2024-10-11 14:41:58.592619+00', NULL,
    true, false, 'auto-onboard',
    '{"tenant_id": "4725ad3d-5ab0-4f42-8a4a-fdee5ef586c5", "subscription_id": "75b0a9a9-3222-4290-bdf9-56127d550563", "subscription_tags": {"env": ["Sandbox", "new"], "test": ["true"], "testkey": ["testvalue"], "environment": ["production"]}, "subscription_model": {"id": "/subscriptions/75b0a9a9-3222-4290-bdf9-56127d550563", "state": "Enabled", "displayName": "Policy Testing Subscription", "subscriptionId": "75b0a9a9-3222-4290-bdf9-56127d550563", "authorizationSource": "RoleBased", "subscriptionPolicies": {"quotaId": "PayAsYouGo_2014-09-01", "spendingLimit": "Off", "locationPlacementId": "Public_2014-09-01"}}}',
    '2024-09-27 17:04:58.064271+00', '2024-10-11 14:41:58.593093+00'
),
(
    'c00bb650-f448-41b7-8ccc-bcd6184f78c3', 'df34e0ad-fb1f-4b54-9686-bbcaba2a82fb', 'Sample Sub 1', NULL, 'Azure',
    'Auto on-boarded subscription df34e0ad-fb1f-4b54-9686-bbcaba2a82fb', 'ONBOARD', 'scheduled', 'unhealthy',
    '2024-10-11 14:42:03.09474+00', NULL,
    true, false, 'auto-onboard',
    '{"tenant_id": "4725ad3d-5ab0-4f42-8a4a-fdee5ef586c5", "subscription_id": "df34e0ad-fb1f-4b54-9686-bbcaba2a82fb", "subscription_tags": {}, "subscription_model": {"id": "/subscriptions/df34e0ad-fb1f-4b54-9686-bbcaba2a82fb", "state": "Enabled", "displayName": "Sample Sub 1", "subscriptionId": "df34e0ad-fb1f-4b54-9686-bbcaba2a82fb", "authorizationSource": "RoleBased", "subscriptionPolicies": {"quotaId": "PayAsYouGo_2014-09-01", "spendingLimit": "Off", "locationPlacementId": "Public_2014-09-01"}}}',
    '2024-09-27 17:04:58.400485+00', '2024-10-11 14:42:03.095214+00'
);
EOF

PGPASSWORD="$POSTGRESQL_PASSWORD" psql --host="$POSTGRESQL_HOST" --port="$POSTGRESQL_PORT" --username "$POSTGRESQL_USERNAME" --dbname "metadata" < /demo-data/postgres/metadata.sql

rm -rf /demo-data/postgres