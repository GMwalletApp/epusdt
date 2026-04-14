# Epusdt — Easy Payment USDT
 
<p align="center">
  <img src="wiki/img/usdtlogo.png" alt="Epusdt Logo - USDT Payment Gateway for Crypto Payments" width="120">
</p>
 
<p align="center">
  <strong>开源 USDT 支付网关 · Crypto 支付工具实际采用率 Top 1</strong>
</p>
 
<p align="center">
  <a href="https://epusdt.com"><img src="https://img.shields.io/badge/官网文档-epusdt.com-blue?style=for-the-badge" alt="Official Docs"></a>
  <a href="https://t.me/epusdt"><img src="https://img.shields.io/badge/Telegram-频道-26A5E4?style=for-the-badge&logo=telegram&logoColor=white" alt="Telegram Channel"></a>
  <a href="https://t.me/epusdt_group"><img src="https://img.shields.io/badge/Telegram-交流群-26A5E4?style=for-the-badge&logo=telegram&logoColor=white" alt="Telegram Group"></a>
</p>
 
<p align="center">
  <a href="https://github.com/assimon/epusdt/stargazers"><img src="https://img.shields.io/github/stars/assimon/epusdt?style=flat-square&color=f5c542" alt="GitHub Stars 3000+"></a>
  <a href="https://www.gnu.org/licenses/gpl-3.0.html"><img src="https://img.shields.io/badge/License-GPLv3-blue?style=flat-square" alt="GPLv3 License"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.16+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go 1.16+"></a>
  <a href="https://github.com/assimon/epusdt/releases"><img src="https://img.shields.io/github/v/release/assimon/epusdt?style=flat-square&color=green" alt="Latest Release"></a>
</p>
 
---
 
## 🌍 What is Epusdt?
 
**Epusdt** (Easy Payment USDT) is a self-hosted **USDT payment gateway** built with Go, enabling any website or application to accept **crypto payments** via the **TRC20 network**. No third-party fees, no custodial risk — USDT goes directly into your wallet.
 
> **GitHub Star 3000+** · **已支持站点解决方案 10+** · **Crypto 支付工具实际采用率 Top 1**
 
Deploy it privately, integrate via HTTP API, and start receiving **USDT payments** in minutes. That's it. 🎉
 
---
 
## 🔌 广泛兼容，即插即用
 
无论你运营的是哪类系统，Epusdt 均可基于现有接口方案，**无需重构业务逻辑**，快速接入，立即获得 Crypto 收款能力，低成本扩展全球支付场景：
 
| 领域 | 已支持系统 |
|------|-----------|
| **AI 分发** | OneAPI、NewAPI |
| **发卡系统** | 独角数卡（Dujiaoka）、异次元发卡 |
| **代理面板** | V2Board、XBoard、SSPanel |
| **建站生态** | WordPress、WHMCS |
| **更多** | 任何支持 HTTP Callback 的系统均可接入 |
 
👉 查看完整集成列表与插件：[plugins/](plugins/)
 
---
 
## ✨ 核心特性
 
- **私有化部署** — 无需担心钱包被篡改、吞单，资金完全自主掌控
- **零依赖运行** — 单个二进制文件即可启动，仅需 MySQL + Redis
- **跨平台** — 支持 x86 / ARM 架构的 Windows / Linux 设备
- **多钱包轮询** — 自动轮换收款地址，提高订单并发处理能力
- **异步队列** — 高性能消息回调，优雅处理高并发场景
- **HTTP API** — 标准化接口，任何语言 / 框架均可快速集成
- **Telegram Bot** — 实时支付通知，便捷管理与监控
 
---
 
## 📖 文档与教程
 
完整文档请访问 👉 **[epusdt.com](https://epusdt.com)**
 
快速入门：
 
| 教程 | 说明 |
|------|------|
| [Docker 部署](wiki/docker-RUN.md) | 推荐方式，一键启动 |
| [宝塔面板部署](wiki/BT_RUN.md) | 适合宝塔用户 |
| [手动部署](wiki/manual_RUN.md) | 完全手动控制 |
| [开发者 API 文档](wiki/API.md) | 接口集成指南 |
| [自定义汇率接口](wiki/LEGACY_API_MIGRATION.md) | API 进阶用法 |
| [PHP 极速接入](https://github.com/BlueSkyXN/PHPAPI-for-epusdt) | HTML + PHP 快速集成 |
 
---
 
## 🏗️ 项目结构
 
```
Epusdt
├── plugins/    → 已集成的系统插件（独角数卡等）
├── src/        → 项目核心代码
├── sdk/        → 接入 SDK
├── sql/        → 数据库安装 / 升级脚本
└── wiki/       → 文档与知识库
```
 
---
 
## 🔧 实现原理
 
Epusdt 通过监听 TRC20 网络 API，实时捕获钱包地址的 USDT 入账事件，利用**金额差异**与**时效性**精确匹配交易归属：
 
```
工作流程：
1. 客户发起支付，需支付 20.05 USDT
2. 系统在哈希表中查找可用的 钱包地址 + 金额 组合
3. 若 address_1:20.05 未被占用 → 锁定该组合（有效期 10 分钟），返回给客户
4. 若已被占用 → 自动累加 0.0001 尝试下一个金额组合（最多 100 次）
5. 后台线程持续监听所有钱包的入账事件，金额匹配则确认支付成功
```
 
![Epusdt 支付流程图](wiki/img/implementation_principle.jpg)
 
---
 
## 💬 社区与支持
 
**遇到问题？** 请优先在 GitHub 提交 [Issue](https://github.com/assimon/epusdt/issues)，我们会**优先处理** Issue 中的反馈。
 
加入 Telegram 社区，获取最新动态、交流使用经验：
 
| 渠道 | 链接 |
|------|------|
| 📢 **Epusdt 频道** | [https://t.me/epusdt](https://t.me/epusdt) |
| 💬 **Epusdt 交流群** | [https://t.me/epusdt_group](https://t.me/epusdt_group) |
| 📚 **官方文档站** | [https://epusdt.com](https://epusdt.com) |
 
---
 
## 📜 开源协议
 
Epusdt 遵守 [GPLv3](https://www.gnu.org/licenses/gpl-3.0.html) 开源协议。
 
---
 
## ⚠️ 免责声明
 
本项目仅供学习与技术交流使用，用户在使用过程中需自行遵守所在地法律法规。由于涉及加密资产及资金安全，用户应自行审查相关代码与风险。加密资产属于高风险新兴资产（包括稳定币），其价值可能波动甚至归零，GMwallet 不对任何资产或使用结果作出保证。本内容不构成任何投资、税务、法律或金融建议，仅供教育用途，相关决策请咨询专业人士。
 
---
 
<p align="center">
  <sub>
    <b>Keywords:</b> USDT Payment Gateway · Crypto Payment · TRC20 Payment · Self-hosted USDT · 
    OneAPI Payment · NewAPI Payment · 独角数卡支付 · 异次元发卡支付方式 · 
    V2Board Payment · XBoard Payment · SSPanel 支付接口 · 
    WordPress Crypto Payment · WHMCS USDT Payment · 
    Epusdt · Easy Payment USDT · Open Source Payment Gateway
  </sub>
</p>
 
