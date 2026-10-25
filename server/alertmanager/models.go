package alertmanager

import (
	"time"

	"github.com/prometheus/alertmanager/pkg/labels"
)

// Silence is the Alertmanager API v2 wire format. It is kept locally because
// Alertmanager no longer exports the former types.Silence client type.
type Silence struct {
	ID        string          `json:"id"`
	Matchers  labels.Matchers `json:"matchers"`
	StartsAt  time.Time       `json:"startsAt"`
	EndsAt    time.Time       `json:"endsAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
	CreatedBy string          `json:"createdBy"`
	Comment   string          `json:"comment"`
	Status    struct {
		State string `json:"state"`
	} `json:"status"`
}
