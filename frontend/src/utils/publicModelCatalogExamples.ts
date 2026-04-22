import type { PublicModelCatalogDetailResponse } from "@/api/meta";
import {
  buildCompletedCodeTabs,
  type DocsCodeExampleGroup,
  type DocsCodeExampleTab,
} from "@/utils/docsCodeExamples";
import {
  normalizeDocsPageId,
  parseDocsMarkdown,
  type DocsPage,
  type DocsPageId,
} from "@/utils/markdownDocs";

const PLACEHOLDER_KEY = "sk-your-key";

export interface PublicModelExampleResult {
  group: DocsCodeExampleGroup | null;
  pageId: DocsPageId;
}

export function buildPublicModelExample(
  detail: PublicModelCatalogDetailResponse | null,
  apiKey: string,
  baseUrl: string,
): PublicModelExampleResult {
  if (!detail) {
    return {
      group: null,
      pageId: "common",
    };
  }

  const pageId = resolveExamplePageID(detail);
  const group =
    detail.example_source === "docs_section"
      ? buildDocsExampleGroup(detail, apiKey, baseUrl)
      : buildOverrideExampleGroup(detail, apiKey, baseUrl);

  return {
    group,
    pageId,
  };
}

function buildDocsExampleGroup(
  detail: PublicModelCatalogDetailResponse,
  apiKey: string,
  baseUrl: string,
): DocsCodeExampleGroup | null {
  const markdown = String(detail.example_markdown || "").trim();
  if (!markdown) {
    return null;
  }
  const parsed = parseDocsMarkdown(markdown);
  const page = parsed.pages.find((item) => item.id === resolveExamplePageID(detail));
  const sourceGroup = findFirstCodeGroup(page);
  if (!sourceGroup) {
    return null;
  }
  return {
    id: sourceGroup.id,
    tabs: sourceGroup.tabs.map((tab) => ({
      ...tab,
      code: adaptExampleCode(tab.code, detail, apiKey, baseUrl),
    })),
  };
}

function findFirstCodeGroup(page: DocsPage | undefined): DocsCodeExampleGroup | null {
  if (!page) {
    return null;
  }
  for (const block of page.introBlocks) {
    if (block.kind === "code-group") {
      return block.group;
    }
  }
  for (const section of page.sections) {
    for (const block of section.contentBlocks) {
      if (block.kind === "code-group") {
        return block.group;
      }
    }
  }
  return null;
}

function buildOverrideExampleGroup(
  detail: PublicModelCatalogDetailResponse,
  apiKey: string,
  baseUrl: string,
): DocsCodeExampleGroup | null {
  const overrideID = String(detail.example_override_id || "").trim();
  const protocol = String(detail.example_protocol || "").trim();
  const model = detail.item.model;

  switch (overrideID) {
    case "embeddings":
      return createExampleGroup("embeddings", [
        createTab(
          "Python",
          "python",
          [
            "import requests",
            "",
            `base_url = "${baseUrl}"`,
            `api_key = "${apiKey}"`,
            "",
            "response = requests.post(",
            '    f"{base_url}/v1/embeddings",',
            "    headers={",
            '        "Authorization": f"Bearer {api_key}",',
            '        "Content-Type": "application/json",',
            "    },",
            "    json={",
            `        "model": "${model}",`,
            '        "input": "Summarize the latest release notes.",',
            "    },",
            "    timeout=60,",
            ")",
            "",
            "print(response.status_code)",
            "print(response.json())",
          ].join("\n"),
        ),
        createTab(
          "JavaScript",
          "javascript",
          [
            `const baseUrl = "${baseUrl}";`,
            `const apiKey = "${apiKey}";`,
            "",
            'const response = await fetch(`${baseUrl}/v1/embeddings`, {',
            '  method: "POST",',
            "  headers: {",
            "    Authorization: `Bearer ${apiKey}`,",
            '    "Content-Type": "application/json",',
            "  },",
            "  body: JSON.stringify({",
            `    model: "${model}",`,
            '    input: "Summarize the latest release notes.",',
            "  }),",
            "});",
            "",
            "console.log(response.status, await response.json());",
          ].join("\n"),
        ),
        createTab(
          "REST",
          "rest",
          [
            `curl ${baseUrl}/v1/embeddings \\`,
            `  -H "Authorization: Bearer ${apiKey}" \\`,
            '  -H "Content-Type: application/json" \\',
            "  -d '{",
            `    "model": "${model}",`,
            '    "input": "Summarize the latest release notes."',
            "  }'",
          ].join("\n"),
        ),
      ]);
    case "tts":
      return createExampleGroup("tts", [
        createTab(
          "Python",
          "python",
          [
            "import requests",
            "",
            `base_url = "${baseUrl}"`,
            `api_key = "${apiKey}"`,
            "",
            "response = requests.post(",
            '    f"{base_url}/v1/audio/speech",',
            "    headers={",
            '        "Authorization": f"Bearer {api_key}",',
            '        "Content-Type": "application/json",',
            "    },",
            "    json={",
            `        "model": "${model}",`,
            '        "voice": "alloy",',
            '        "input": "Read this status update out loud.",',
            "    },",
            "    timeout=60,",
            ")",
            "",
            "print(response.status_code)",
            'print(response.headers.get("content-type"))',
          ].join("\n"),
        ),
        createTab(
          "JavaScript",
          "javascript",
          [
            `const baseUrl = "${baseUrl}";`,
            `const apiKey = "${apiKey}";`,
            "",
            'const response = await fetch(`${baseUrl}/v1/audio/speech`, {',
            '  method: "POST",',
            "  headers: {",
            "    Authorization: `Bearer ${apiKey}`,",
            '    "Content-Type": "application/json",',
            "  },",
            "  body: JSON.stringify({",
            `    model: "${model}",`,
            '    voice: "alloy",',
            '    input: "Read this status update out loud.",',
            "  }),",
            "});",
            "",
            'console.log(response.status, response.headers.get("content-type"));',
          ].join("\n"),
        ),
        createTab(
          "REST",
          "rest",
          [
            `curl ${baseUrl}/v1/audio/speech \\`,
            `  -H "Authorization: Bearer ${apiKey}" \\`,
            '  -H "Content-Type: application/json" \\',
            "  -d '{",
            `    "model": "${model}",`,
            '    "voice": "alloy",',
            '    "input": "Read this status update out loud."',
            "  }'",
          ].join("\n"),
        ),
      ]);
    case "image-generation":
      if (protocol === "grok") {
        return createExampleGroup("image-generation-grok", [
          createTab(
            "Python",
            "python",
            [
              "import requests",
              "",
              `base_url = "${baseUrl}"`,
              `api_key = "${apiKey}"`,
              "",
              "response = requests.post(",
              '    f"{base_url}/grok/v1/images/generations",',
              "    headers={",
              '        "Authorization": f"Bearer {api_key}",',
              '        "Content-Type": "application/json",',
              "    },",
              "    json={",
              `        "model": "${model}",`,
              '        "prompt": "Create a clean product hero image.",',
              "    },",
              "    timeout=60,",
              ")",
              "",
              "print(response.status_code)",
              "print(response.json())",
            ].join("\n"),
          ),
          createTab(
            "REST",
            "rest",
            [
              `curl ${baseUrl}/grok/v1/images/generations \\`,
              `  -H "Authorization: Bearer ${apiKey}" \\`,
              '  -H "Content-Type: application/json" \\',
              "  -d '{",
              `    "model": "${model}",`,
              '    "prompt": "Create a clean product hero image."',
              "  }'",
            ].join("\n"),
          ),
        ]);
      }
      if (protocol === "gemini") {
        return createExampleGroup("image-generation-gemini", [
          createTab(
            "Python",
            "python",
            [
              "import requests",
              "",
              `base_url = "${baseUrl}"`,
              `api_key = "${apiKey}"`,
              "",
              "response = requests.post(",
              '    f"{base_url}/v1beta/openai/images/generations",',
              "    headers={",
              '        "Authorization": f"Bearer {api_key}",',
              '        "Content-Type": "application/json",',
              "    },",
              "    json={",
              `        "model": "${model}",`,
              '        "prompt": "Create a clean product hero image.",',
              "    },",
              "    timeout=60,",
              ")",
              "",
              "print(response.status_code)",
              "print(response.json())",
            ].join("\n"),
          ),
          createTab(
            "REST",
            "rest",
            [
              `curl ${baseUrl}/v1beta/openai/images/generations \\`,
              `  -H "Authorization: Bearer ${apiKey}" \\`,
              '  -H "Content-Type: application/json" \\',
              "  -d '{",
              `    "model": "${model}",`,
              '    "prompt": "Create a clean product hero image."',
              "  }'",
            ].join("\n"),
          ),
        ]);
      }
      return createExampleGroup("image-generation", [
        createTab(
          "Python",
          "python",
          [
            "import requests",
            "",
            `base_url = "${baseUrl}"`,
            `api_key = "${apiKey}"`,
            "",
            "response = requests.post(",
            '    f"{base_url}/v1/images/generations",',
            "    headers={",
            '        "Authorization": f"Bearer {api_key}",',
            '        "Content-Type": "application/json",',
            "    },",
            "    json={",
            `        "model": "${model}",`,
            '        "prompt": "Create a clean product hero image.",',
            "    },",
            "    timeout=60,",
            ")",
            "",
            "print(response.status_code)",
            "print(response.json())",
          ].join("\n"),
        ),
        createTab(
          "REST",
          "rest",
          [
            `curl ${baseUrl}/v1/images/generations \\`,
            `  -H "Authorization: Bearer ${apiKey}" \\`,
            '  -H "Content-Type: application/json" \\',
            "  -d '{",
            `    "model": "${model}",`,
            '    "prompt": "Create a clean product hero image."',
            "  }'",
          ].join("\n"),
        ),
      ]);
    case "image-generation-tool":
      return createExampleGroup("image-generation-tool", [
        createTab(
          "Python",
          "python",
          [
            "import requests",
            "",
            `base_url = "${baseUrl}"`,
            `api_key = "${apiKey}"`,
            "",
            "response = requests.post(",
            '    f"{base_url}/v1/responses",',
            "    headers={",
            '        "Authorization": f"Bearer {api_key}",',
            '        "Content-Type": "application/json",',
            "    },",
            "    json={",
            `        "model": "${model}",`,
            '        "input": "Create a clean product hero image.",',
            '        "tools": [{"type": "image_generation", "model": "gpt-image-2"}],',
            "    },",
            "    timeout=60,",
            ")",
            "",
            "print(response.status_code)",
            "print(response.json())",
          ].join("\n"),
        ),
        createTab(
          "JavaScript",
          "javascript",
          [
            `const baseUrl = "${baseUrl}";`,
            `const apiKey = "${apiKey}";`,
            "",
            'const response = await fetch(`${baseUrl}/v1/responses`, {',
            '  method: "POST",',
            "  headers: {",
            "    Authorization: `Bearer ${apiKey}`,",
            '    "Content-Type": "application/json",',
            "  },",
            "  body: JSON.stringify({",
            `    model: "${model}",`,
            '    input: "Create a clean product hero image.",',
            '    tools: [{ type: "image_generation", model: "gpt-image-2" }],',
            "  }),",
            "});",
            "",
            "console.log(response.status, await response.json());",
          ].join("\n"),
        ),
        createTab(
          "REST",
          "rest",
          [
            `curl ${baseUrl}/v1/responses \\`,
            `  -H "Authorization: Bearer ${apiKey}" \\`,
            '  -H "Content-Type: application/json" \\',
            "  -d '{",
            `    "model": "${model}",`,
            '    "input": "Create a clean product hero image.",',
            '    "tools": [{"type": "image_generation", "model": "gpt-image-2"}]',
            "  }'",
          ].join("\n"),
        ),
      ]);
    case "video-generation":
      if (protocol !== "grok") {
        return null;
      }
      return createExampleGroup("video-generation", [
        createTab(
          "Python",
          "python",
          [
            "import requests",
            "",
            `base_url = "${baseUrl}"`,
            `api_key = "${apiKey}"`,
            "",
            "response = requests.post(",
            '    f"{base_url}/grok/v1/videos/generations",',
            "    headers={",
            '        "Authorization": f"Bearer {api_key}",',
            '        "Content-Type": "application/json",',
            "    },",
            "    json={",
            `        "model": "${model}",`,
            '        "prompt": "Generate a short cinematic teaser.",',
            "    },",
            "    timeout=60,",
            ")",
            "",
            "print(response.status_code)",
            "print(response.json())",
          ].join("\n"),
        ),
        createTab(
          "REST",
          "rest",
          [
            `curl ${baseUrl}/grok/v1/videos/generations \\`,
            `  -H "Authorization: Bearer ${apiKey}" \\`,
            '  -H "Content-Type: application/json" \\',
            "  -d '{",
            `    "model": "${model}",`,
            '    "prompt": "Generate a short cinematic teaser."',
            "  }'",
          ].join("\n"),
        ),
      ]);
    default:
      return null;
  }
}

function createExampleGroup(
  id: string,
  tabs: DocsCodeExampleTab[],
): DocsCodeExampleGroup {
  return {
    id,
    tabs: buildCompletedCodeTabs(tabs, id),
  };
}

function createTab(
  label: DocsCodeExampleTab["label"],
  language: string,
  code: string,
): DocsCodeExampleTab {
  return {
    code,
    focusLines: [],
    id: `${label.toLowerCase()}-tab`,
    label,
    language,
  };
}

function adaptExampleCode(
  code: string,
  detail: PublicModelCatalogDetailResponse,
  apiKey: string,
  baseUrl: string,
): string {
  const normalizedBaseUrl = String(baseUrl || "").trim().replace(/\/+$/g, "");
  const normalizedAPIKey = String(apiKey || PLACEHOLDER_KEY).trim() || PLACEHOLDER_KEY;
  const model = detail.item.model;

  return String(code || "")
    .replace(/https:\/\/api\.zyxai\.de/g, normalizedBaseUrl)
    .replace(/sk-your-key/g, normalizedAPIKey)
    .replace(/sk-你的站内Key/g, normalizedAPIKey)
    .replace(/sk-你的站内密钥/g, normalizedAPIKey)
    .replace(
      /(["'`])model\1\s*:\s*(["'`])[^"'`\n]+\2/g,
      (_match, quote, valueQuote) => `${quote}model${quote}: ${valueQuote}${model}${valueQuote}`,
    )
    .replace(
      /\bmodel\s*=\s*(["'`])[^"'`\n]+\1/g,
      `model = "${model}"`,
    )
    .replace(
      /(\/v1beta\/models\/)[^:"'`\s]+(:[A-Za-z]+)/g,
      `$1${model}$2`,
    )
    .replace(
      /(\/v1\/models\/)[^:"'`\s]+(:[A-Za-z]+)/g,
      `$1${model}$2`,
    );
}

function resolveExamplePageID(
  detail: PublicModelCatalogDetailResponse,
): DocsPageId {
  const explicitPageID = String(detail.example_page_id || "").trim();
  if (explicitPageID) {
    return normalizeDocsPageId(explicitPageID);
  }
  switch (detail.example_protocol) {
    case "anthropic":
      return "anthropic";
    case "gemini":
      return "gemini";
    case "grok":
      return "grok";
    case "antigravity":
      return "antigravity";
    case "vertex-batch":
      return "vertex-batch";
    case "openai":
    default:
      return "common";
  }
}
