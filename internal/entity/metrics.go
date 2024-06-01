// Structure of Popcorn Metrics Model.

package entity

type Metrics struct {
	// Current Ingress count
	ActiveIngress int `json:"-" redis:"active_ingress"`
	// Ingress limit exceeded indicator
	IngressQuotaExceeded bool `json:"ingress_quota_exceeded" redis:"ingress_quota_exceeded"`
}
