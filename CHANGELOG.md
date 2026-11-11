#### Changelog 1.3.0-app - 11-11-2026
- 合入 BoxProxy App 26.11.05.d5c365e 的 `boxctl` / `boxbpf` 运行时
- 新增 `boxdb`：安装与开机时把 `settings.ini`、`package.list.cfg`、`gid.list.cfg` 同步到 App 版 v13 数据库 `box.db`
- 服务与透明代理规则改由 `boxctl` 驱动，保留原 `box.service` / `box.iptables` 命令接口，旧脚本调用方无需修改
- 保留核心、订阅、Geo、WebUI 更新脚本；暂仅支持 arm64-v8a
- 已知差异：App 版数据库不再保存 `force_proxy_cidrs`，该模块旧字段暂不下发

#### Changelog 1.2.7 - 04-04-2026
- 重构网络与 iptables 规则框架，增强 TProxy/IPv6 与网络匹配稳定性
- 新增 CNIP（ipset）分流能力，并提供 upcnip 更新命令
- 安装恢复流程补充 gid.list.cfg

#### Changelog 1.2.6 - 06-11-2025
- 优化 WebUI 不再是固定 dashboard 目录
- 为模块安装新增删除更新残留选项，新增超时自动选择
- 为更新订阅新增自动修改 mihomo 的 proxy-providers 配置内容

#### Changelog 1.2.5 - 02-10-2025
- 优化安装脚本优化 service 的 singbox 部分内容
- 增加 GitHub Token 密钥填写
- 精简优化默认 mihomo 配置

#### Changelog 1.2.4 - 23-08-2025
- 优化与重构

# Changelog

