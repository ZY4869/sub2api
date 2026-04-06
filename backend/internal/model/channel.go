package model

import (
	"slices"
	"strings"
	"time"
)

const (
	ChannelStatusActive   = "active"
	ChannelStatusDisabled = "disabled"

	ChannelBillingModeToken      = "token"
	ChannelBillingModePerRequest = "per_request"
	ChannelBillingModeImage      = "image"

	ChannelBillingModelSourceMapped   = "channel_mapped"
	ChannelBillingModelSourceRequested = "requested"
	ChannelBillingModelSourceUpstream  = "upstream"
)

type Channel struct {
	ID                 int64                         `json:"id"`
	Name               string                        `json:"name"`
	Description        string                        `json:"description,omitempty"`
	Status             string                        `json:"status"`
	RestrictModels     bool                          `json:"restrict_models"`
	BillingModelSource string                        `json:"billing_model_source"`
	GroupIDs           []int64                       `json:"group_ids"`
	ModelMapping       map[string]map[string]string  `json:"model_mapping"`
	ModelPricing       []ChannelModelPricing         `json:"model_pricing"`
	CreatedAt          time.Time                     `json:"created_at"`
	UpdatedAt          time.Time                     `json:"updated_at"`
}

type ChannelModelPricing struct {
	ID               int64                    `json:"id"`
	ChannelID        int64                    `json:"channel_id,omitempty"`
	Platform         string                   `json:"platform"`
	Models           []string                 `json:"models"`
	BillingMode      string                   `json:"billing_mode"`
	InputPrice       *float64                 `json:"input_price"`
	OutputPrice      *float64                 `json:"output_price"`
	CacheWritePrice  *float64                 `json:"cache_write_price"`
	CacheReadPrice   *float64                 `json:"cache_read_price"`
	ImageOutputPrice *float64                 `json:"image_output_price"`
	PerRequestPrice  *float64                 `json:"per_request_price"`
	Intervals        []ChannelPricingInterval `json:"intervals"`
	SortOrder        int                      `json:"sort_order"`
}

type ChannelPricingInterval struct {
	ID              int64    `json:"id"`
	PricingID       int64    `json:"pricing_id,omitempty"`
	MinTokens       int64    `json:"min_tokens"`
	MaxTokens       *int64   `json:"max_tokens"`
	TierLabel       string   `json:"tier_label"`
	InputPrice      *float64 `json:"input_price"`
	OutputPrice     *float64 `json:"output_price"`
	CacheWritePrice *float64 `json:"cache_write_price"`
	CacheReadPrice  *float64 `json:"cache_read_price"`
	PerRequestPrice *float64 `json:"per_request_price"`
	SortOrder       int      `json:"sort_order"`
}

func (c *Channel) Normalize() {
	c.Name = strings.TrimSpace(c.Name)
	c.Description = strings.TrimSpace(c.Description)
	c.Status = normalizeChannelStatus(c.Status)
	c.BillingModelSource = normalizeChannelBillingModelSource(c.BillingModelSource)
	c.GroupIDs = normalizeChannelGroupIDs(c.GroupIDs)
	c.ModelMapping = normalizeChannelModelMapping(c.ModelMapping)

	for i := range c.ModelPricing {
		c.ModelPricing[i].normalize(i)
	}
}

func (c *Channel) Validate() error {
	if c == nil {
		return &ValidationError{Field: "channel", Message: "channel is required"}
	}
	c.Normalize()

	if c.Name == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	if c.Status != ChannelStatusActive && c.Status != ChannelStatusDisabled {
		return &ValidationError{Field: "status", Message: "status must be active or disabled"}
	}
	if c.BillingModelSource != ChannelBillingModelSourceMapped &&
		c.BillingModelSource != ChannelBillingModelSourceRequested &&
		c.BillingModelSource != ChannelBillingModelSourceUpstream {
		return &ValidationError{Field: "billing_model_source", Message: "invalid billing_model_source"}
	}

	for i := range c.ModelPricing {
		if err := c.ModelPricing[i].Validate(i); err != nil {
			return err
		}
	}

	return nil
}

func (p *ChannelModelPricing) normalize(idx int) {
	p.Platform = strings.TrimSpace(strings.ToLower(p.Platform))
	p.BillingMode = normalizeChannelBillingMode(p.BillingMode)
	p.SortOrder = idx
	p.Models = normalizeStringSlice(p.Models)
	for i := range p.Intervals {
		p.Intervals[i].SortOrder = i
		p.Intervals[i].TierLabel = strings.TrimSpace(p.Intervals[i].TierLabel)
	}
}

func (p *ChannelModelPricing) Validate(idx int) error {
	if p.Platform == "" {
		return &ValidationError{Field: "model_pricing.platform", Message: "platform is required"}
	}
	if len(p.Models) == 0 {
		return &ValidationError{Field: "model_pricing.models", Message: "at least one model is required"}
	}
	if p.BillingMode != ChannelBillingModeToken &&
		p.BillingMode != ChannelBillingModePerRequest &&
		p.BillingMode != ChannelBillingModeImage {
		return &ValidationError{Field: "model_pricing.billing_mode", Message: "invalid billing_mode"}
	}
	if err := validateNullablePrice("input_price", p.InputPrice); err != nil {
		return err
	}
	if err := validateNullablePrice("output_price", p.OutputPrice); err != nil {
		return err
	}
	if err := validateNullablePrice("cache_write_price", p.CacheWritePrice); err != nil {
		return err
	}
	if err := validateNullablePrice("cache_read_price", p.CacheReadPrice); err != nil {
		return err
	}
	if err := validateNullablePrice("image_output_price", p.ImageOutputPrice); err != nil {
		return err
	}
	if err := validateNullablePrice("per_request_price", p.PerRequestPrice); err != nil {
		return err
	}

	if (p.BillingMode == ChannelBillingModePerRequest || p.BillingMode == ChannelBillingModeImage) &&
		p.PerRequestPrice == nil && len(p.Intervals) == 0 {
		return &ValidationError{Field: "model_pricing.per_request_price", Message: "per_request/image pricing requires a default price or intervals"}
	}

	for i := range p.Intervals {
		if err := p.Intervals[i].Validate(i); err != nil {
			return err
		}
	}

	sorted := slices.Clone(p.Intervals)
	slices.SortFunc(sorted, func(a, b ChannelPricingInterval) int {
		switch {
		case a.MinTokens < b.MinTokens:
			return -1
		case a.MinTokens > b.MinTokens:
			return 1
		default:
			return 0
		}
	})
	for i := 1; i < len(sorted); i++ {
		prev := sorted[i-1]
		curr := sorted[i]
		if prev.MaxTokens == nil || *prev.MaxTokens > curr.MinTokens {
			return &ValidationError{Field: "model_pricing.intervals", Message: "pricing intervals overlap"}
		}
	}

	return nil
}

func (iv *ChannelPricingInterval) Validate(idx int) error {
	if iv.MinTokens < 0 {
		return &ValidationError{Field: "interval.min_tokens", Message: "min_tokens cannot be negative"}
	}
	if iv.MaxTokens != nil && *iv.MaxTokens <= iv.MinTokens {
		return &ValidationError{Field: "interval.max_tokens", Message: "max_tokens must be greater than min_tokens"}
	}
	if err := validateNullablePrice("input_price", iv.InputPrice); err != nil {
		return err
	}
	if err := validateNullablePrice("output_price", iv.OutputPrice); err != nil {
		return err
	}
	if err := validateNullablePrice("cache_write_price", iv.CacheWritePrice); err != nil {
		return err
	}
	if err := validateNullablePrice("cache_read_price", iv.CacheReadPrice); err != nil {
		return err
	}
	if err := validateNullablePrice("per_request_price", iv.PerRequestPrice); err != nil {
		return err
	}
	if idx < 0 {
		return &ValidationError{Field: "interval.sort_order", Message: "invalid sort order"}
	}
	return nil
}

func normalizeChannelStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return ChannelStatusActive
	}
	return status
}

func normalizeChannelBillingMode(mode string) string {
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		return ChannelBillingModeToken
	}
	return mode
}

func normalizeChannelBillingModelSource(source string) string {
	source = strings.TrimSpace(strings.ToLower(source))
	if source == "" {
		return ChannelBillingModelSourceMapped
	}
	return source
}

func normalizeChannelGroupIDs(groupIDs []int64) []int64 {
	if len(groupIDs) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(groupIDs))
	result := make([]int64, 0, len(groupIDs))
	for _, id := range groupIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func normalizeChannelModelMapping(input map[string]map[string]string) map[string]map[string]string {
	if len(input) == 0 {
		return map[string]map[string]string{}
	}
	output := make(map[string]map[string]string, len(input))
	for platform, mapping := range input {
		platform = strings.TrimSpace(strings.ToLower(platform))
		if platform == "" || len(mapping) == 0 {
			continue
		}
		normalized := make(map[string]string, len(mapping))
		for source, target := range mapping {
			source = strings.TrimSpace(source)
			target = strings.TrimSpace(target)
			if source == "" || target == "" {
				continue
			}
			normalized[source] = target
		}
		if len(normalized) > 0 {
			output[platform] = normalized
		}
	}
	if len(output) == 0 {
		return map[string]map[string]string{}
	}
	return output
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func validateNullablePrice(field string, value *float64) error {
	if value == nil {
		return nil
	}
	if *value < 0 {
		return &ValidationError{Field: field, Message: "price cannot be negative"}
	}
	return nil
}
