#!/usr/bin/env bash
set -euo pipefail

alertmanager_url=${ALERTMANAGER_URL:-http://localhost:9093}
alert_name="LocalTest${RANDOM}"
starts_at=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

send_alert() {
    curl --fail-with-body --silent --show-error \
        -H 'Content-Type: application/json' \
        --data "[{\"labels\":{\"alertname\":\"${alert_name}\",\"service\":\"local-test\",\"severity\":\"warning\"},\"annotations\":{\"summary\":\"Local plugin smoke test\"},\"startsAt\":\"${starts_at}\"${1:-}}]" \
        "${alertmanager_url%/}/api/v2/alerts"
}

send_alert
printf 'Sent %s. Check the alerts channel. Press Enter to resolve it.\n' "$alert_name"
read -r
ends_at=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
send_alert ",\"endsAt\":\"${ends_at}\""
printf 'Resolved %s.\n' "$alert_name"
