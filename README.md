# Box for Root

> **本分支：App 运行时合入模块版**
>
> - 内置 BoxProxy App `26.11.05.d5c365e` 的 `boxctl` / `boxbpf`，模块服务与透明代理规则由 `boxctl` 驱动
> - `settings.ini`、`package.list.cfg`、`gid.list.cfg` 仍是配置入口；新增 `boxdb` 在安装/开机时把它们同步到 App 版 v13 数据库 `/data/adb/box/box.db`
> - 保留原 `box.service` / `box.iptables` 命令接口，`box.tool`、`action.sh` 等旧调用方无需改
> - 仅支持 **arm64-v8a**；App 版数据库暂不包含模块旧的 `force_proxy_cidrs`
> - 不要同时用 BoxProxy App 的“启动服务”，模块与应用版运行时都改 iptables，会互相覆盖

`Box for Root` 是一个面向 Android Root 环境（Magisk / KernelSU / APatch）的透明代理工具箱模块。

项目深受以下仓库启发并在其基础上持续演进：
- [CHIZI-0618/box4magisk](https://github.com/CHIZI-0618/box4magisk)
- [CHIZI-0618/AndroidTProxyShell](https://github.com/CHIZI-0618/AndroidTProxyShell)
- [taamarin/box_for_magisk](https://github.com/taamarin/box_for_magisk)

## 项目定位

本仓库主要提供：
- 统一的代理核心运行管理（mihomo / sing-box / xray / v2fly / hysteria）
- 多网络模式下的透明代理规则编排（TProxy / Redirect / Tun / Mixed / Enhance）
- 订阅、Geo 资源、核心二进制与 WebUI 的统一维护脚本
- 适配 Android Root 生态的模块化目录与服务生命周期管理

## 主要目录

模块工作目录：`/data/adb/box/`

```text
/data/adb/box/
├── bin/                # 代理核心与工具二进制
├── mihomo/             # mihomo 配置目录
├── sing-box/           # sing-box 配置目录
├── xray/               # xray 配置目录
├── v2fly/              # v2fly 配置目录
├── hysteria/           # hysteria 配置目录
├── scripts/            # 核心脚本
│   ├── box.service     # 服务生命周期管理
│   ├── box.iptables    # 透明代理规则管理
│   └── box.tool        # 更新与维护工具集
├── run/                # 运行时状态与日志
└── settings.ini        # 全局配置文件
```

## 核心脚本

- `box.service`: 启停、重启、状态与定时任务控制
- `box.iptables`: 透明代理规则启用、重建、清理
- `box.tool`: 订阅更新、Geo 更新、核心更新、配置检查、WebUI 相关维护

## 文档与社区

- Wiki: <https://github.com/boxproxy/box/wiki>
- 更新日志: [CHANGELOG.md](./CHANGELOG.md)
- 配套 APP / 通知频道: <https://t.me/zero_o0>

## 致谢

感谢开源社区与上述项目作者提供的设计思路与实现参考。

## 本分支安装与使用（App 运行时合入）

1. 在 Magisk / KernelSU / APatch 管理器中刷入 `box-1.3.0-app.20261111.zip`（仅 arm64-v8a）。
2. 原有配置入口不变：`/data/adb/box/settings.ini`、`package.list.cfg`、`gid.list.cfg`；安装时和每次启动会由 `boxdb` 同步到 `/data/adb/box/box.db`（App 版 v13 结构）。
3. `box.service` / `box.iptables` 的命令行接口保持不变；实际启动、规则、WiFi 策略由 App 版 `boxctl` 执行。
4. 首次使用若没有内核，请在安装器里下载，或用 `box.tool` 更新；日志在 `/data/adb/box/run/runs.log`。
5. 不要同时启动 BoxProxy App 的“服务”，App 版和模块版都会改 iptables，二选一。

运行时二进制来源：[boxproxy/boxproxy](https://github.com/boxproxy/boxproxy)（GPL-3.0）；本分支从 BoxProxy App `26.11.05.d5c365e` 提取 `boxctl` / `boxbpf`。

开发构建（boxdb，需 Go 1.25+）：

```sh
sh tools/build-boxdb.sh
```
