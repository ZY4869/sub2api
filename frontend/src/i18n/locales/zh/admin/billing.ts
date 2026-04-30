export default {
    title: "计费中心",
    description: "统一管理模型定价、规则矩阵与计费模拟。",
    pages: {
        pricing: {
            nav: "模型定价",
            title: "模型定价",
            description: "默认列表模式分页展示模型价格，并支持切换到供应商九宫格巡检。",
        },
        publicCatalog: {
            nav: "对外模型展示",
            title: "对外模型展示",
            description: "维护公开模型库的草稿配置，并手动推送发布快照。",
        },
        rules: {
            nav: "规则与模拟",
            title: "规则与模拟",
            description: "维护高级计费规则，并用模拟器验证命中与回退路径。",
        },
    },
}
