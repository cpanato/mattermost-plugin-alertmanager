package notification

import (
	"strings"
	"sync"
	"testing"

	amtemplate "github.com/prometheus/alertmanager/template"
	"github.com/stretchr/testify/require"
)

func sampleData() Data {
	return AlertData(amtemplate.Data{
		Status: "firing", Receiver: "platform", ExternalURL: "https://alertmanager.example.com",
		GroupLabels: amtemplate.KV{"cluster": "production"}, CommonLabels: amtemplate.KV{"service": "checkout"}, RouteLabels: amtemplate.KV{"team": "sre"},
	}, amtemplate.Alert{Status: "resolved", Labels: amtemplate.KV{"alertname": "InstanceDown", "instance": "host-1", "severity": "warning"}, Annotations: amtemplate.KV{"summary": "Host recovered"}}, "east")
}

func TestCompactTemplate(t *testing.T) {
	tmpl, err := Parse(`{{ .Status | toUpper }}: {{ .Labels.alertname }}`, `alertname={{ .Labels.alertname }}, instance={{ .Labels.instance }}, severity={{ .Labels.severity }}; {{ .Receiver }}/{{ .GroupLabels.cluster }}/{{ .CommonLabels.service }}/{{ .RouteLabels.team }}/{{ .ConfigID }}`)
	require.NoError(t, err)
	attachment, err := tmpl.Render(sampleData())
	require.NoError(t, err)
	require.Equal(t, "RESOLVED: InstanceDown", attachment.Title)
	require.Equal(t, "alertname=InstanceDown, instance=host-1, severity=warning; platform/production/checkout/sre/east", attachment.Text)
	require.Equal(t, ColorResolved, attachment.Color)
	require.Contains(t, attachment.Fallback, "RESOLVED: InstanceDown\nalertname=InstanceDown")
	require.Empty(t, attachment.Fields)
	require.Empty(t, attachment.Footer)
}

func TestOptionalTemplates(t *testing.T) {
	for _, tc := range []struct {
		name, title, message, wantTitle, wantText string
		detailed                                  bool
	}{
		{"default", "", "", "RESOLVED: InstanceDown", "**Host recovered**", true},
		{"whitespace", " \n ", "\t", "RESOLVED: InstanceDown", "**Host recovered**", true},
		{"title only", "{{ .Annotations.summary }}", "", "Host recovered", "**Host recovered**", false},
		{"message only", "", "{{ .Annotations.summary }}", "RESOLVED: InstanceDown", "Host recovered", false},
		{"missing label", "", "instance={{ .Labels.absent }}", "RESOLVED: InstanceDown", "instance=", false},
		{"named templates", "", "{{define \"body\"}}{{ .Status }}{{end}}{{template \"body\" .}}", "RESOLVED: InstanceDown", "resolved", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := Parse(tc.title, tc.message)
			require.NoError(t, err)
			attachment, err := tmpl.Render(sampleData())
			require.NoError(t, err)
			require.Equal(t, tc.wantTitle, attachment.Title)
			require.Equal(t, tc.wantText, attachment.Text)
			if tc.detailed {
				require.Equal(t, Default(sampleData()), attachment)
			} else {
				require.Empty(t, attachment.Fields)
			}
		})
	}
}

func TestTemplateErrors(t *testing.T) {
	for _, source := range []string{"{{", "{{ unknownFunction }}", strings.Repeat("x", MaxTemplateBytes+1), strings.Repeat(" ", MaxTemplateBytes+1)} {
		_, err := Parse("", source)
		require.Error(t, err)
	}
	for _, source := range []string{`{{ .UnknownField }}`, `{{ index .Labels.Names 99 }}`, `{{ reReplaceAll "[" "x" "y" }}`, `{{ .Annotations.large }}`, `{{ range .Labels }}{{ $.Annotations.large }}{{ end }}`} {
		t.Run(source, func(t *testing.T) {
			data := sampleData()
			data.Annotations["large"] = strings.Repeat("x", MaxTemplateBytes+1)
			tmpl, err := Parse("", source)
			require.NoError(t, err)
			attachment, err := tmpl.Render(data)
			require.Error(t, err)
			require.Nil(t, attachment)
		})
	}
	tmpl, err := Parse(`{{ "" }}`, `{{ "" }}`)
	require.NoError(t, err)
	_, err = tmpl.Render(sampleData())
	require.ErrorContains(t, err, "empty title and message")
	// Check literals as well as dynamic values: text/template may use WriteString.
	tmpl, err = Parse("", strings.Repeat("x", MaxTemplateBytes))
	require.NoError(t, err)
	attachment, err := tmpl.Render(sampleData())
	require.NoError(t, err)
	require.Len(t, attachment.Text, MaxTemplateBytes)
	data := sampleData()
	data.Annotations["chunk"] = strings.Repeat("x", MaxTemplateBytes/2)
	tmpl, err = Parse("", `{{ .Annotations.chunk }}{{ .Annotations.chunk }}x`)
	require.NoError(t, err)
	_, err = tmpl.Render(data)
	require.ErrorContains(t, err, "rendered output exceeds")
}

func TestTemplateDataIsNotReevaluated(t *testing.T) {
	data := sampleData()
	data.Annotations["summary"] = `{{ .Token }} <script> & **markdown**`
	tmpl, err := Parse("", `{{ .Annotations.summary }}`)
	require.NoError(t, err)
	attachment, err := tmpl.Render(data)
	require.NoError(t, err)
	require.Equal(t, data.Annotations["summary"], attachment.Text)
}

func TestConcurrentRendering(t *testing.T) {
	tmpl, err := Parse("{{ .Status | title }}", `{{ .Labels.instance }}`)
	require.NoError(t, err)
	var wg sync.WaitGroup
	data := sampleData()
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			attachment, err := tmpl.Render(data)
			if err != nil || attachment == nil || attachment.Text != "host-1" {
				t.Errorf("render: %v %v", attachment, err)
			}
		}()
	}
	wg.Wait()
}
