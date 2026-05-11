# Sub2API 步骤 3：前端静态资源与 API 反向分析

审计日期：2026-05-09  
执行边界：本地构建产物 / 授权远程静态入口  
当前状态：本地静态反向已完成，远程静态抓取阻塞

## 当前说明

- 本文件用于承接步骤 3 的静态资源、前端 API 路径、source map、敏感字符串与暴露入口分析结果。
- 本轮已完成 `frontend` `typecheck` 与 `pnpm --dir frontend build`，并对 `backend/internal/web/dist` 与 `frontend/src` 做了本地静态反向分析。
- 远程首页与静态资源抓取仍缺测试窗口、测试账号与授权补充信息，因此本文件当前只收录本地静态证据与阻塞说明。

## 本地构建与入口基线

- `pnpm --dir frontend build` 成功，说明当前前端源码在步骤 2 修复后可正常产出静态资源。
- 当前 `backend/internal/web/dist/index.html` 只引用本轮入口资产：
  - `/assets/index-DH0CVHr4.js`
  - `/assets/vendor-vue-Dhsqe4n_.js`
  - `/assets/vendor-misc-B99KfT-n.js`
  - `/assets/vendor-i18n-Ds2miYOs.js`
  - `/assets/vendor-misc-DB0Q8XAf.css`
  - `/assets/index-mWFxjuHT.css`
- `dist` 目录中存在大量历史 hash 资产，但当前 live entrypoint 只加载上述 6 个入口资源；历史文件不应直接当作当前页面必经路径。

## Source Map、敏感字符串与暴露入口核对

- `backend/internal/web/dist` 下未发现 `.map` 文件，本轮未见 source map 暴露。
- 对 `dist` 扫描以下敏感串均未命中：
  - `token=`
  - `src_url`
  - `src_host`
  - `admin123`
  - `sk-ant-mock-audit`
- `frontend/src/utils/embedded-url.ts` 当前只追加：
  - `theme`
  - 可选 `lang`
  - 固定 `ui_mode=embedded`
- 本地源码中的高风险管理入口映射已核对：
  - `frontend/src/api/admin/backup.ts` 暴露 `/admin/backups/*`
  - `frontend/src/api/admin/dataManagement.ts` 暴露 `/admin/data-management/*`
  - `frontend/src/generated/protocolGateway.ts` 暴露 `/v1/messages` 与 `/v1/messages/count_tokens`
  - `frontend/src/router/index.ts` 与相关 `composables` 暴露 `/admin/accounts` 等管理入口

## 本地静态反向结论

- 当前本地构建产物未暴露 source map。
- 当前 `dist` 未发现本轮关注的明文敏感串。
- iframe 嵌入 URL 参数面已收缩为展示安全参数，不再静态拼接 token、user_id、src_url、src_host。
- `backup`、`data-management`、`accounts` 与协议网关入口在前端源码中可见，但其服务端访问控制已在 `security/findings/auth-matrix.md` 中按运行态完成交叉验证。
- 本地静态反向分析未单独新增步骤 3 漏洞；相关结果已作为步骤 3 收口证据并入正式问题单。

## 阻塞项

- 远程首页与静态资源的低风险抓取仍缺测试窗口、测试账号、测试 API Key 与授权范围补充信息。
- 在阻塞解除前，不执行对 `https://api.zyxai.de`、`https://demo.sub2api.org/` 等远程资产的主动抓取。

## 当前暂停点

- 步骤 3 的本地静态反向部分已完成并留痕。
- 后续仅在你修复步骤 3 问题并进入步骤 4，或补充远程授权信息后，再追加远程静态核查结果。
