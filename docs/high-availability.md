# Alertmanager high availability

Configure the Alertmanager replicas as one cluster. Running three independent
instances with the same webhook receiver sends three notifications: the plugin
posts each webhook it receives. Alertmanager shares its notification log between
cluster peers to deduplicate deliveries before they reach Mattermost.

Reproduced locally with Alertmanager 0.34.1 and Mattermost 11.7.14, sending the
same alert and its resolution to all three instances:

| Setup | Firing notifications | Resolved notifications |
| --- | --- | --- |
| Three healthy cluster peers | 1 | 1 |
| Three independent instances, clustering disabled | 3 | 3 |

Each Prometheus server should send the same alerts to every Alertmanager replica,
with consistent labels. Configure the same routing and receivers on all peers,
set `--cluster.peer` addresses, and allow cluster communication on port 9094
(TCP and UDP for the default transport). Each advertised address must be reachable
from the other peers. Wait for gossip to settle before evaluating notifications.

Check `/api/v2/status` on **every** replica. A healthy three-instance cluster has
`cluster.status: ready` and three entries in `cluster.peers` on each member. Also
check `alertmanager_cluster_members`, peer health and gossip logs. More pods alone
do not establish a healthy cluster.

The plugin's Alertmanager configurations are destinations, not a cluster-membership
list. Use one configuration per desired webhook/channel destination; setting up
one configuration per replica does not enable HA. Slash commands query the URL
of each configured destination; the local HA example uses the first peer.

## Local reproduction

Use the disposable Docker environment described in the README:

```sh
nvm install
make dev-ha-up dev-setup dev-ha-smoke
```

This adds two Alertmanager peers on localhost ports 9095 and 9096 alongside the
first peer on 9093. Gossip stays on the Docker network; port 9094 is not published
to the host. The smoke test waits for all three members to report a ready
three-peer cluster, then sends the same alert to **all three** API endpoints.
It checks for exactly one yellow firing notification and one green resolved
notification, while also verifying slash commands.

For each notification, it observes 40 seconds after the first delivery. This
covers the last peer's default 30-second notification delay plus the example's
5-second grouping interval. The local repeat interval is 30 minutes. The test
takes around 90 seconds after the cluster is ready and is also included in the
manually triggered integration workflow for Mattermost 11.7 and 12.

Stop the HA stack with `make dev-ha-down`. Its Mattermost/PostgreSQL data is kept.
To return to the single-instance setup, run `make dev-up` afterward.

## Delivery guarantees

Alertmanager provides at-least-once delivery. Network partitions, unsettled gossip
and notification-log replication problems can still produce duplicates. This is
intentional: Alertmanager prioritizes delivering alerts during a partition.

The plugin does not suppress matching webhooks using a time window. Such a window
could hide legitimate repeat notifications, updated alert annotations or status
changes, and would need to coordinate across Mattermost nodes. Check the upstream
cluster before treating duplicates as a plugin bug.

See the official [Alertmanager HA documentation](https://prometheus.io/docs/alerting/latest/high_availability/)
for cluster configuration, troubleshooting and the behavior during partitions.
