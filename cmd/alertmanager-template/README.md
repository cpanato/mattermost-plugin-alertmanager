# alertmanager-template

Validate and preview notifications using an Alertmanager webhook JSON file and
optional Go template files. The CLI shares the plugin's parser and renderer, so
a valid preview produces the same attachment title, body, fallback text and
colors as a live webhook.

Requires **Go 1.27+**. The CLI itself makes no network calls and does not need
Mattermost, Alertmanager or Docker running.

## Quick start

From the repository root:

```sh
make template-preview
```

Or invoke the CLI directly:

```sh
go run ./cmd/alertmanager-template \
  -input examples/templates/webhook.json \
  -title-template examples/templates/title.tmpl \
  -message-template examples/templates/message.tmpl
```

From this directory (`cmd/alertmanager-template`):

```sh
go run . \
  -input ../../examples/templates/webhook.json \
  -title-template ../../examples/templates/title.tmpl \
  -message-template ../../examples/templates/message.tmpl
```

To build a standalone executable, run from the repository root:

```sh
go build -o /tmp/alertmanager-template ./cmd/alertmanager-template
/tmp/alertmanager-template -help
```

## Flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `-input` | `-` | Webhook JSON file; `-` reads stdin |
| `-title-template` | unset | Optional title template file |
| `-message-template` | unset | Optional message template file |
| `-config-id` | `preview` | Destination ID exposed as `.ConfigID` |
| `-help` | | Show usage and exit successfully |

The template flags take **file paths**, not inline template strings. For example,
a message template file can contain:

```gotemplate
alertname={{ .Labels.alertname }}, instance={{ .Labels.instance }}, severity={{ .Labels.severity }}
```

A title template can contain:

```gotemplate
{{ .Status | toUpper }}: {{ .Annotations.summary }}
```

Templates run once per alert, so `.Status` reflects the individual alert even in
a mixed firing/resolved group. Missing map keys render empty. See the
[template guide](../../docs/templates.md#available-data) for all data and helpers.

## Input and output

Input must contain exactly one webhook JSON object with at least one alert:

```json
{
  "status": "firing",
  "receiver": "platform",
  "alerts": [{
    "status": "firing",
    "labels": {
      "alertname": "InstanceDown",
      "instance": "host-1",
      "severity": "warning"
    },
    "annotations": {"summary": "Host unreachable"}
  }]
}
```

To read stdin from the repository root:

```sh
cat examples/templates/webhook.json | go run ./cmd/alertmanager-template \
  -message-template examples/templates/message.tmpl
```

Stdout is a JSON array with one Mattermost attachment per alert. For the input
above with the compact message template, its relevant fields are:

```json
[
  {
    "title": "FIRING: InstanceDown",
    "text": "alertname=InstanceDown, instance=host-1, severity=warning",
    "color": "#FFCC00",
    "fallback": "FIRING: InstanceDown\nalertname=InstanceDown, instance=host-1, severity=warning"
  }
]
```

The actual JSON also includes the remaining attachment fields. Status/severity
colors are automatic; resolved alerts stay green.

Omit both template flags to preview the detailed default format. Setting either
enables compact output without metadata fields or a footer; an omitted title or
body retains its default text. Each template source and rendered custom field
is limited to 16 KiB.

## Errors and applying templates

Exit status is **0** for a successful preview or `-help`, and **1** for errors.
Errors go to stderr. Invalid JSON, syntax errors, execution failures and empty
combined title/body are rejected without partial attachment JSON on stdout.

The CLI fails on template errors so you can fix them. The live plugin instead
logs the error and uses the default format to preserve alert delivery.

Once validated, open **System Console → Plugins → AlertManager**, expand
**Custom notification templates** for your destination, paste the template file
contents into **Title template** and/or **Message template**, and save. There is
no need to upload files or the preview JSON. Leave both fields blank to restore
the default format.
