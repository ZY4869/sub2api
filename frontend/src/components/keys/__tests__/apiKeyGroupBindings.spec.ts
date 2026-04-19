import { describe, expect, it } from "vitest";
import { buildApiKeyGroupBindingPayload } from "../apiKeyGroupBindings";

describe("buildApiKeyGroupBindingPayload", () => {
  it("treats an empty structured selection as all models", () => {
    const payload = buildApiKeyGroupBindingPayload(
      [
        {
          group_id: 10,
          quota: 0,
          model_patterns_text: "legacy-*",
          selected_models: [],
          model_selection_dirty: true,
        },
      ],
      false,
    );

    expect(payload).toEqual([{ group_id: 10 }]);
  });

  it("submits the selected models as model_patterns", () => {
    const payload = buildApiKeyGroupBindingPayload(
      [
        {
          group_id: 12,
          quota: 0,
          model_patterns_text: "",
          selected_models: ["gpt-5.4", "claude-sonnet-4-5"],
          model_selection_dirty: true,
        },
      ],
      false,
    );

    expect(payload).toEqual([
      {
        group_id: 12,
        model_patterns: ["gpt-5.4", "claude-sonnet-4-5"],
      },
    ]);
  });
});
