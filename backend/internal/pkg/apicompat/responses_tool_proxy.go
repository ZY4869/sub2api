package apicompat

import (
	"encoding/json"
	"fmt"
	"strings"
)

const responsesToolProxyPrefix = "rspx_"

type ResponsesToolProxy struct {
	Type      string
	Name      string
	Namespace string
	ProxyName string
}

func (p ResponsesToolProxy) OutputType() string {
	switch strings.TrimSpace(p.Type) {
	case "custom":
		return "custom_tool_call"
	case "tool_search":
		return "tool_search_call"
	case "namespace":
		return "mcp_tool_call"
	default:
		return "function_call"
	}
}

func (p ResponsesToolProxy) OutputName() string {
	if strings.TrimSpace(p.Namespace) != "" && strings.TrimSpace(p.Name) != "" {
		return strings.TrimSpace(p.Namespace) + "." + strings.TrimSpace(p.Name)
	}
	return strings.TrimSpace(p.Name)
}

func (t *ResponsesTool) UnmarshalJSON(data []byte) error {
	var shorthand string
	if err := json.Unmarshal(data, &shorthand); err == nil {
		shorthand = strings.TrimSpace(shorthand)
		switch shorthand {
		case "tool_search":
			*t = ResponsesTool{Type: "tool_search", Name: "tool_search"}
		default:
			*t = ResponsesTool{Type: "function", Name: shorthand}
		}
		return nil
	}

	type alias ResponsesTool
	var raw alias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*t = ResponsesTool(raw)
	if strings.TrimSpace(t.Type) == "" && strings.TrimSpace(t.Name) != "" {
		t.Type = "function"
	}
	return nil
}

func ResponsesToolProxyMap(tools []ResponsesTool) (map[string]ResponsesToolProxy, error) {
	converted, err := convertResponsesToolsToChat(tools)
	if err != nil {
		return nil, err
	}
	return converted.byProxy, nil
}

type responsesToolConversion struct {
	chatTools []ChatTool
	byProxy   map[string]ResponsesToolProxy
	byChoice  map[string]string
}

func convertResponsesToolsToChat(tools []ResponsesTool) (*responsesToolConversion, error) {
	out := &responsesToolConversion{
		byProxy:  map[string]ResponsesToolProxy{},
		byChoice: map[string]string{},
	}
	seenDisplay := map[string]struct{}{}
	seenChat := map[string]struct{}{}
	if err := appendResponsesTools(out, tools, "", seenDisplay, seenChat); err != nil {
		return nil, err
	}
	return out, nil
}

func appendResponsesTools(out *responsesToolConversion, tools []ResponsesTool, namespace string, seenDisplay, seenChat map[string]struct{}) error {
	for _, tool := range tools {
		tool.Type = normalizeResponsesToolType(tool.Type)
		if tool.Type == "namespace" {
			childNamespace := firstNonEmptyCompat(tool.Namespace, tool.Name, namespace)
			if err := appendResponsesTools(out, tool.Tools, childNamespace, seenDisplay, seenChat); err != nil {
				return err
			}
			continue
		}
		if tool.Type == "" {
			tool.Type = "function"
		}

		displayName := flattenedResponsesToolName(namespace, tool.Name)
		if displayName == "" && tool.Type == "tool_search" {
			displayName = flattenedResponsesToolName(namespace, "tool_search")
			tool.Name = "tool_search"
		}
		if displayName == "" {
			return fmt.Errorf("responses tool name is required")
		}
		if _, exists := seenDisplay[displayName]; exists {
			return fmt.Errorf("responses tool name conflict after namespace flatten: %s", displayName)
		}
		seenDisplay[displayName] = struct{}{}

		proxy := ResponsesToolProxy{
			Type:      tool.Type,
			Name:      strings.TrimSpace(tool.Name),
			Namespace: strings.TrimSpace(namespace),
		}
		proxy.ProxyName = responsesToolChatName(proxy)
		if tool.Type == "function" && proxy.Namespace == "" {
			proxy.ProxyName = proxy.Name
		}
		if _, exists := seenChat[proxy.ProxyName]; exists {
			return fmt.Errorf("responses tool proxy name conflict: %s", proxy.ProxyName)
		}
		seenChat[proxy.ProxyName] = struct{}{}

		out.chatTools = append(out.chatTools, ChatTool{
			Type: "function",
			Function: &ChatFunction{
				Name:        proxy.ProxyName,
				Description: responsesToolDescription(tool, proxy),
				Parameters:  responsesToolParameters(tool),
				Strict:      tool.Strict,
			},
		})
		if proxy.ProxyName != proxy.Name || proxy.Type != "function" || proxy.Namespace != "" {
			out.byProxy[proxy.ProxyName] = proxy
		}
		out.byChoice[responsesToolChoiceKey(proxy.Type, proxy.Name, proxy.Namespace)] = proxy.ProxyName
		out.byChoice[responsesToolChoiceKey(proxy.Type, proxy.OutputName(), "")] = proxy.ProxyName
		out.byChoice[responsesToolChoiceKey("", proxy.Name, proxy.Namespace)] = proxy.ProxyName
		out.byChoice[responsesToolChoiceKey("", proxy.OutputName(), "")] = proxy.ProxyName
	}
	return nil
}
