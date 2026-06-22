import { mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import AccountDaily5HTriggerSettingsDialog from "../AccountDaily5HTriggerSettingsDialog.vue";

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        if (key === "admin.accounts.daily5h.candidateCount") {
          return `candidate-${params?.count ?? ""}`;
        }
        if (key === "admin.accounts.daily5h.modelCount") {
          return `model-${params?.count ?? ""}`;
        }
        if (key === "admin.accounts.daily5h.supportedAccountsCount") {
          return `accounts-${params?.count ?? ""}`;
        }
        if (key === "common.save") return "save";
        if (key === "common.cancel") return "cancel";
        if (key === "common.saving") return "saving";
        return key;
      },
    }),
  };
});

function mountDialog() {
  return mount(AccountDaily5HTriggerSettingsDialog, {
    props: {
      show: true,
      saving: false,
      settings: {
        enabled: true,
        selected_account_types: ["chatgpt_oauth"],
        include_paused_accounts: false,
        ignore_free_accounts: false,
        skip_cn_holidays_and_weekends: false,
        openai_model_mode: { mode: "auto", fixed_model_id: "" },
        anthropic_model_mode: { mode: "auto", fixed_model_id: "" },
        gemini_model_mode: { mode: "auto", fixed_model_id: "" },
      },
      candidates: [
        {
          account_type: "chatgpt_oauth",
          count: 3,
          models: [
            {
              model_id: "gpt-5.4-mini",
              display_name: "GPT-5.4 Mini",
              provider: "openai",
              account_count: 3,
            },
          ],
        },
        {
          account_type: "claude_code_oauth_setup_token",
          count: 1,
          models: [
            {
              model_id: "claude-3.5-haiku",
              display_name: "Claude 3.5 Haiku",
              provider: "anthropic",
              account_count: 1,
            },
          ],
        },
        {
          account_type: "google_oauth",
          count: 2,
          models: [
            {
              model_id: "gemini-2.5-flash",
              display_name: "Gemini 2.5 Flash",
              provider: "gemini",
              account_count: 2,
            },
          ],
        },
      ],
    },
    global: {
      stubs: {
        BaseDialog: {
          props: ["show", "title"],
          template: '<div><slot /><slot name="footer" /></div>',
        },
        Icon: true,
        PlatformLabel: {
          props: ["label", "description"],
          template: "<div>{{ label }} {{ description }}</div>",
        },
        ModelIcon: true,
        Select: {
          props: ["modelValue", "options"],
          emits: ["update:modelValue"],
          template: `
            <button
              type="button"
              class="select-stub"
              @click="$emit('update:modelValue', options[0]?.model_id || '')"
            >
              {{ modelValue || 'select' }}
            </button>
          `,
        },
      },
    },
  });
}

describe("AccountDaily5HTriggerSettingsDialog", () => {
  it("renders candidate summaries and default selection", () => {
    const wrapper = mountDialog();

    expect(wrapper.text()).toContain("candidate-3");
    expect(wrapper.text()).toContain("model-1");

    const checkboxes = wrapper.findAll('input[type="checkbox"]');
    expect((checkboxes[0].element as HTMLInputElement).checked).toBe(true);
    expect((checkboxes[1].element as HTMLInputElement).checked).toBe(false);
    expect((checkboxes[2].element as HTMLInputElement).checked).toBe(false);
    expect(wrapper.text()).toContain("admin.accounts.daily5h.ignoreFreeLabel");
  });

  it("emits normalized settings after toggling account types, free skip, and fixed model mode", async () => {
    const wrapper = mountDialog();

    const fixedButtons = wrapper
      .findAll("button")
      .filter((button) =>
        button.text().includes("admin.accounts.daily5h.modelModeFixed"),
      );
    await fixedButtons[0].trigger("click");
    await wrapper.get(".select-stub").trigger("click");

    const checkboxes = wrapper.findAll('input[type="checkbox"]');
    await checkboxes[1].setValue(true);

    const ignoreFreeButton = wrapper.findAll("button")[2];
    expect(ignoreFreeButton).toBeTruthy();
    await ignoreFreeButton.trigger("click");

    const saveButton = wrapper
      .findAll("button")
      .find((button) => button.text().includes("save"));
    expect(saveButton).toBeTruthy();
    await saveButton!.trigger("click");

    expect(wrapper.emitted("save")).toEqual([
      [
        {
          enabled: true,
          selected_account_types: [
            "chatgpt_oauth",
            "claude_code_oauth_setup_token",
          ],
          include_paused_accounts: false,
          ignore_free_accounts: true,
          skip_cn_holidays_and_weekends: false,
          openai_model_mode: {
            mode: "fixed",
            fixed_model_id: "gpt-5.4-mini",
          },
          anthropic_model_mode: { mode: "auto", fixed_model_id: "" },
          gemini_model_mode: { mode: "auto", fixed_model_id: "" },
        },
      ],
    ]);
  });

  it("emits the non-workday skip flag when enabled", async () => {
    const wrapper = mountDialog();

    expect(wrapper.text()).toContain("admin.accounts.daily5h.skipNonWorkdaysLabel");

    const skipNonWorkdayButton = wrapper
      .findAll("button")
      .filter((button) => button.classes().includes("relative"))[3];
    expect(skipNonWorkdayButton).toBeTruthy();
    await skipNonWorkdayButton!.trigger("click");

    const saveButton = wrapper
      .findAll("button")
      .find((button) => button.text().includes("save"));
    expect(saveButton).toBeTruthy();
    await saveButton!.trigger("click");

    expect(wrapper.emitted("save")?.[0]?.[0]).toMatchObject({
      enabled: true,
      selected_account_types: ["chatgpt_oauth"],
      ignore_free_accounts: false,
      skip_cn_holidays_and_weekends: true,
    });
  });
});
