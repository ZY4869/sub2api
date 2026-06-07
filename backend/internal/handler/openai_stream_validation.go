package handler

import "github.com/tidwall/gjson"

func parseOpenAIStreamFlag(body []byte) (bool, bool) {
	stream := gjson.GetBytes(body, "stream")
	if !stream.Exists() {
		return false, true
	}
	switch stream.Type {
	case gjson.True, gjson.False:
		return stream.Bool(), true
	default:
		return false, false
	}
}
