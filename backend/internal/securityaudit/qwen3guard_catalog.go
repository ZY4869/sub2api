package securityaudit

type ScannerDefinition struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	LabelZH     string `json:"label_zh"`
	Description string `json:"description"`
}

var AllScannerIDs = []string{
	"violent",
	"non_violent_illegal_acts",
	"sexual_content_or_sexual_acts",
	"pii",
	"suicide_and_self_harm",
	"unethical_acts",
	"politically_sensitive_topics",
	"copyright_violation",
	"jailbreak",
}

var ScannerCatalog = map[string]ScannerDefinition{
	"violent":                       {ID: "violent", Label: "Violent", LabelZH: "暴力", Description: "Violence or threats of violence"},
	"non_violent_illegal_acts":      {ID: "non_violent_illegal_acts", Label: "Non-violent Illegal Acts", LabelZH: "非暴力违法行为", Description: "Non-violent illegal activity"},
	"sexual_content_or_sexual_acts": {ID: "sexual_content_or_sexual_acts", Label: "Sexual Content or Sexual Acts", LabelZH: "性内容或性行为", Description: "Sexual content or sexual acts"},
	"pii":                           {ID: "pii", Label: "PII", LabelZH: "个人敏感信息", Description: "Personal identifying information"},
	"suicide_and_self_harm":         {ID: "suicide_and_self_harm", Label: "Suicide & Self-Harm", LabelZH: "自杀与自残", Description: "Suicide or self-harm"},
	"unethical_acts":                {ID: "unethical_acts", Label: "Unethical Acts", LabelZH: "不道德行为", Description: "Unethical behavior"},
	"politically_sensitive_topics":  {ID: "politically_sensitive_topics", Label: "Politically Sensitive Topics", LabelZH: "政治敏感话题", Description: "Politically sensitive topics"},
	"copyright_violation":           {ID: "copyright_violation", Label: "Copyright Violation", LabelZH: "版权侵权", Description: "Copyright infringement"},
	"jailbreak":                     {ID: "jailbreak", Label: "Jailbreak", LabelZH: "越狱攻击", Description: "Prompt injection or jailbreak attempt"},
}

var categoryAliases = map[string]string{
	"violent": "violent", "violence": "violent",
	"non violent illegal acts":      "non_violent_illegal_acts",
	"non-violent illegal acts":      "non_violent_illegal_acts",
	"non_violent_illegal_acts":      "non_violent_illegal_acts",
	"sexual":                        "sexual_content_or_sexual_acts",
	"sexual content or sexual acts": "sexual_content_or_sexual_acts",
	"pii":                           "pii", "personal identifying information": "pii", "personal identifiable information": "pii",
	"suicide self harm":     "suicide_and_self_harm",
	"suicide and self harm": "suicide_and_self_harm",
	"suicide/self harm":     "suicide_and_self_harm",
	"suicide & self-harm":   "suicide_and_self_harm",
	"unethical":             "unethical_acts", "unethical acts": "unethical_acts",
	"political":                    "politically_sensitive_topics",
	"politically sensitive topics": "politically_sensitive_topics",
	"copyright":                    "copyright_violation", "copyright violation": "copyright_violation",
	"jailbreak": "jailbreak", "prompt injection": "jailbreak",
}

func ScannerDefinitions() []ScannerDefinition {
	out := make([]ScannerDefinition, 0, len(AllScannerIDs))
	for _, id := range AllScannerIDs {
		out = append(out, ScannerCatalog[id])
	}
	return out
}
