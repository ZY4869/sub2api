export default {
    label: "图片协议兜底模式",
    hint: "仅对 OpenAI 分组生效：优先级高于账号配置，可强制原生生图或兼容生图。",
    options: {
        inherit: "继承账号",
        native: "强制原生",
        compat: "强制兼容",
    },
}
