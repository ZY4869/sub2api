# SECURITY_SCOPE.md

## 授权范围

本次安全审计仅限我自己拥有或明确授权的资产。

### 本地源码
- 当前本地仓库：`sub2api`

### 本地隔离环境
- Windows 主机下的 WSL `Ubuntu`
- WSL 本地 Docker 环境
- WSL 本地 PostgreSQL
- WSL 本地 Redis
- WSL 本地 mock upstream
- WSL 本地 ZAP / Nuclei / scanner 容器

### Staging
- 形态：WSL 内 staging-like 隔离环境
- 域名：待步骤 3 搭建后填写
- IP：待步骤 3 搭建后填写
- 允许测试窗口：待执行时填写
- 测试账号：待执行时填写
- 测试 API Key：待执行时填写
- 允许主动 DAST：是

### Production
- 域名：
  - `https://api.zyxai.de`
  - `https://demo.sub2api.org/`
- IP：待执行时填写
- 允许测试窗口：待执行时填写
- 测试账号：待执行时填写
- 测试 API Key：待执行时填写
- 只允许低风险检查：是

## 禁止行为

- 不扫描第三方无授权资产
- 不暴力破解
- 不撞库
- 不高并发压测
- 不破坏生产数据
- 不删除或篡改线上数据
- 不使用真实上游额度做压力验证
- 不泄露完整密钥、Token、Cookie、密码
- 不对 production 执行主动攻击型扫描
