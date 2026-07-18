package securityaudit

import "strings"

func (m *ConfigManager) buildEndpoint(input UpdateEndpoint, current storageConfig) (storageEndpoint, error) {
	baseURL, err := NormalizeBaseURL(input.BaseURL)
	if err != nil {
		return storageEndpoint{}, err
	}
	endpoint := storageEndpoint{
		ID: strings.TrimSpace(input.ID), Name: strings.TrimSpace(input.Name), Protocol: strings.TrimSpace(input.Protocol),
		BaseURL: baseURL, Model: strings.TrimSpace(input.Model), TimeoutMS: input.TimeoutMS,
		InputLimit: input.InputLimit, Enabled: input.Enabled,
	}
	applyEndpointDefaults(&endpoint)
	ciphertext, err := m.endpointTokenCiphertext(input, endpoint, current)
	if err != nil {
		return storageEndpoint{}, err
	}
	endpoint.TokenCiphertext = ciphertext
	return endpoint, nil
}

func applyEndpointDefaults(endpoint *storageEndpoint) {
	if endpoint.Protocol == "" {
		endpoint.Protocol = "openai_compatible"
	}
	if endpoint.Model == "" {
		endpoint.Model = DefaultGuardModel
	}
	if endpoint.TimeoutMS == 0 {
		endpoint.TimeoutMS = DefaultTimeoutMS
	}
	if endpoint.InputLimit == 0 {
		endpoint.InputLimit = DefaultInputLimit
	}
}

func (m *ConfigManager) endpointTokenCiphertext(
	input UpdateEndpoint,
	endpoint storageEndpoint,
	current storageConfig,
) (string, error) {
	switch token := strings.TrimSpace(input.Token); {
	case input.ClearToken:
		return "", nil
	case token != "":
		return m.Encrypt(token)
	default:
		return preservedTokenCiphertext(endpoint, current), nil
	}
}
