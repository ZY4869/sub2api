package service

import "github.com/tidwall/gjson"

func extractChannelMonitorResponseText(protocol string, _ string, raw []byte) string {
	switch protocol {
	case ChannelMonitorRequestProtocolOpenAI:
		if v := gjson.GetBytes(raw, "choices.0.message.content"); v.Exists() {
			return v.String()
		}
		if v := gjson.GetBytes(raw, "choices.0.text"); v.Exists() {
			return v.String()
		}
		if v := gjson.GetBytes(raw, "output_text"); v.Exists() {
			return v.String()
		}
		if v := gjson.GetBytes(raw, "output.0.content.0.text"); v.Exists() {
			return v.String()
		}
	case ChannelMonitorRequestProtocolAnthropic:
		if v := gjson.GetBytes(raw, "content.0.text"); v.Exists() {
			return v.String()
		}
		if v := gjson.GetBytes(raw, "completion"); v.Exists() {
			return v.String()
		}
	case ChannelMonitorRequestProtocolGemini:
		if v := gjson.GetBytes(raw, "candidates.0.content.parts.0.text"); v.Exists() {
			return v.String()
		}
	}
	return ""
}

func extractChannelMonitorErrorMessage(raw []byte) string {
	if v := gjson.GetBytes(raw, "error.message"); v.Exists() {
		return v.String()
	}
	if v := gjson.GetBytes(raw, "message"); v.Exists() {
		return v.String()
	}
	return ""
}
