package service

import "time"

type ChannelMonitorTimelineItem struct {
	Status    string    `json:"status"`
	LatencyMs int64     `json:"latency_ms"`
	CheckedAt time.Time `json:"checked_at"`
}

type ChannelMonitorModelLastStatus struct {
	ModelID    string     `json:"model_id"`
	Status     string     `json:"status"`
	LatencyMs  int64      `json:"latency_ms"`
	CheckedAt  *time.Time `json:"checked_at,omitempty"`
	HTTPStatus *int       `json:"http_status,omitempty"`
}

type ChannelMonitorUserListItem struct {
	ID                    int64                           `json:"id"`
	Name                  string                          `json:"name"`
	Provider              string                          `json:"provider"`
	PrimaryModelID        string                          `json:"primary_model_id"`
	PrimaryLast           *ChannelMonitorModelLastStatus  `json:"primary_last,omitempty"`
	PrimaryAvailability7d *float64                        `json:"primary_availability_7d,omitempty"`
	Timeline              []ChannelMonitorTimelineItem    `json:"timeline"`
	AdditionalLast        []ChannelMonitorModelLastStatus `json:"additional_last"`
}

type ChannelMonitorUserModelDetail struct {
	ModelID         string                         `json:"model_id"`
	Last            *ChannelMonitorModelLastStatus `json:"last,omitempty"`
	Availability7d  *float64                       `json:"availability_7d,omitempty"`
	Availability15d *float64                       `json:"availability_15d,omitempty"`
	Availability30d *float64                       `json:"availability_30d,omitempty"`
}

type ChannelMonitorUserDetail struct {
	ID             int64                           `json:"id"`
	Name           string                          `json:"name"`
	Provider       string                          `json:"provider"`
	PrimaryModelID string                          `json:"primary_model_id"`
	Models         []ChannelMonitorUserModelDetail `json:"models"`
}
