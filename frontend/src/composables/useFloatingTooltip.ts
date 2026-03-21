import { computed, nextTick, onBeforeUnmount, ref } from "vue";

type FloatingTooltipPlacement = "top" | "bottom";

const TOOLTIP_GAP = 10;
const VIEWPORT_PADDING = 12;

export function useFloatingTooltip() {
  const tooltipVisible = ref(false);
  const tooltipRef = ref<HTMLElement | null>(null);
  const triggerRef = ref<HTMLElement | null>(null);
  const position = ref({
    top: 0,
    left: 0,
    placement: "bottom" as FloatingTooltipPlacement,
  });

  let listenersAttached = false;

  const updateTooltipPosition = () => {
    if (!tooltipVisible.value) return;
    const triggerEl = triggerRef.value;
    const tooltipEl = tooltipRef.value;
    if (!triggerEl || !tooltipEl) return;

    const triggerRect = triggerEl.getBoundingClientRect();
    const tooltipRect = tooltipEl.getBoundingClientRect();
    const tooltipWidth = tooltipRect.width || tooltipEl.offsetWidth || 0;
    const tooltipHeight = tooltipRect.height || tooltipEl.offsetHeight || 0;

    let placement: FloatingTooltipPlacement = "bottom";
    let top = triggerRect.bottom + TOOLTIP_GAP;

    if (top + tooltipHeight > window.innerHeight - VIEWPORT_PADDING) {
      const nextTop = triggerRect.top - tooltipHeight - TOOLTIP_GAP;
      if (nextTop >= VIEWPORT_PADDING) {
        top = nextTop;
        placement = "top";
      } else {
        top = Math.max(
          VIEWPORT_PADDING,
          window.innerHeight - VIEWPORT_PADDING - tooltipHeight,
        );
      }
    }

    let left = triggerRect.left + triggerRect.width / 2 - tooltipWidth / 2;
    left = Math.max(
      VIEWPORT_PADDING,
      Math.min(left, window.innerWidth - VIEWPORT_PADDING - tooltipWidth),
    );

    position.value = { top, left, placement };
  };

  const attachListeners = () => {
    if (listenersAttached || typeof window === "undefined") return;
    window.addEventListener("resize", updateTooltipPosition);
    window.addEventListener("scroll", updateTooltipPosition, true);
    listenersAttached = true;
  };

  const detachListeners = () => {
    if (!listenersAttached || typeof window === "undefined") return;
    window.removeEventListener("resize", updateTooltipPosition);
    window.removeEventListener("scroll", updateTooltipPosition, true);
    listenersAttached = false;
  };

  const showFloatingTooltip = async (target: HTMLElement | null) => {
    if (!target) return;
    triggerRef.value = target;
    tooltipVisible.value = true;
    attachListeners();
    await nextTick();
    updateTooltipPosition();
  };

  const hideFloatingTooltip = () => {
    tooltipVisible.value = false;
    triggerRef.value = null;
    detachListeners();
  };

  onBeforeUnmount(() => {
    hideFloatingTooltip();
  });

  return {
    tooltipVisible,
    tooltipRef,
    tooltipPlacement: computed(() => position.value.placement),
    tooltipStyle: computed(() => ({
      top: `${position.value.top}px`,
      left: `${position.value.left}px`,
    })),
    triggerRef,
    showFloatingTooltip,
    hideFloatingTooltip,
    updateTooltipPosition,
  };
}
