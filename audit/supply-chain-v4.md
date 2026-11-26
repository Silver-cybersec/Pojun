# 供应链补查 v4

基准 SHA：`44a36480df29a2ea353c3ff5f4d3a7a1e759c046`（https://github.com/chu0119/Pojun/commit/44a36480df29a2ea353c3ff5f4d3a7a1e759c046）。已有长文见 `audit/supply-chain.md`，此处只按八项归类并补缺口。未安装依赖、未拉镜像层、未跑仓库脚本。结论只用规定四词。漏洞公告命中不等于本产品可被利用。

| # | 项 | 结论 |
|---|---|---|
| 1 | 依赖清单与 lockfile（含间接） | 已确认 |
| 2 | 版本钉、校验和、git／URL 直装 | 已确认 |
| 3 | 仿冒名、安装钩子、非官方源 | 可疑 |
| 4 | 容器镜像来源、tag、digest | 已确认 |
| 5 | 子模块、下载脚本、curl\|bash、运行时远程拉取 | 已确认 |
| 6 | CI：Action SHA、密钥、危险触发 | 可疑 |
| 7 | 公开库：恶意包／已知漏洞 | 已确认 |
| 8 | 主要依赖维护状态 | 未发现问题 |

查询：OSV `https://api.osv.dev/v1/querybatch`，2026-11-26（香港时间，见 `audit/supply-chain.md`，959 个锁坐标）。npm `https://registry.npmjs.org/<name>` 与 `https://proxy.golang.org/<module>/@latest`、GitHub 仓库元数据，2026-11-26 11:30:43（香港时间，UTC+8）。

## 1. 清单与 lock

结论：**已确认**有语言没有 lock。

- Go：`go.mod:6-22` 直接依赖，`go.sum` 108 个版本坐标（含间接，也含只出现在 `/go.mod` 行的旧版本）。
- npm：`web/package-lock.json` 766 个不重复包，`skills/api-recon/scripts/package-lock.json` 85 个。`resolved` 均为 `registry.npmjs.org`。`package.json` 里是 `^` 范围，锁文件再钉版本和 `integrity`。
- 无 Python／Rust lock。`Dockerfile:114` 与 `docker/install-tools.sh:108-125` 的 pip／源码安装不进锁。

## 2. 版本钉与直装

结论：**已确认**存在未钉版本、无校验和的直装。锁内 Go `h1` 与 npm `integrity` 只覆盖已解析坐标。

- 已钉且有摘要：`go.sum`；两份 npm lock。`go install` kerbrute `v1.0.3`、waybackurls `v0.1.0`（`Dockerfile:18-19`）版本固定，但不走本仓 `go.sum`。
- 未钉：NodeSource 管道（`Dockerfile:53`）、nuclei `releases/latest`（`Dockerfile:75-80`）、npm `@latest`（`Dockerfile:87`）、无版本 pip（`Dockerfile:114`）、`docker/install-tools.sh:69-83` 的 `releases/latest`、impacket `master` tarball（同文件 `:125`）、KillWxapkg 只钉 tag `v2.4.1`（`Dockerfile:21-23`）、Homebrew `HEAD` 安装脚本（`install.sh:900,1598`）。
- 本仓发布包用同一次 Release 的 `SHA256SUMS`（`install.sh:682`；`selfupdate/stage.go`）。校验和与制品同源，无独立签名。

## 3. 仿冒、钩子、非官方源

结论：**可疑**。未看到仿冒包名；有安装钩子和第三方下载前缀，钩子正文与镜像字节未取回。

- 包名：直接依赖与锁内抽查未见 `node-ipc`、`event-stream`、`flatmap-stream`、`peacenotwar`。`colors` 命中的是 `ansi-colors`／`picocolors`／`yoctocolors`。
- 钩子：`web/package.json:18` 为 `"prepare": "husky"`。`web/package-lock.json:9877-9881` 的 `sharp@0.34.5` 标了 `hasInstallScript`（optional）。未执行安装，未读脚本正文。api-recon 锁无 install 钩子。Go 模块无 install 脚本。
- 非官方源：`mihomo/install_sources.go:41-44` 的 `gh-proxy.com`、`ghfast.top`、`ghproxy.net`，仅在持有 GitHub API 给出的 SHA-256 时作字节搬运。Dockerfile 可用构建参数改 Debian／PyPI 源。

## 4. 镜像

结论：**已确认** tag 未钉 digest。镜像层未拉取，不能写成层内无恶意。

| 位置 | 镜像 | tag | digest |
|---|---|---|---|
| `Dockerfile:16` | `golang` | `1.26-bookworm` | 无 |
| `Dockerfile:25` | `python` | `3.11-slim-bookworm` | 无 |
| `docker-compose.yml:9` | `postgres` | `16-alpine` | 无 |
| `docker-compose.yml:26` | `ghcr.io/chu0119/pojun`（可覆盖） | `${POJUN_TAG:-latest}`（`.env.example:2`） | 无 |
| `.github/workflows/ci.yml:17` | `postgres` | `16` | 无 |

## 5. 子模块与远程拉取

结论：**已确认**有管道安装和运行时外联。无 git 子模块（无 `.gitmodules`）。

- `Dockerfile:53`：`curl \| bash` NodeSource。`install.sh:900,1598`：`curl \| bash` Homebrew `HEAD`。
- 运行时：`llmpresets/modelsdev.go:31` 拉 `https://models.dev/api.json`（失败用内嵌快照）。`server/hostenv_api.go:332` 在探测接口被调用时请求 `api.ipify.org`。`docker/docker-entrypoint.sh:39-47` 默认首次启动跑 `pojun-hostd -install-all`，配方在 `hosttools/hosttools.go`（含 `releases/latest`，如 `:296`、`:346`）。
- 这些 URL 的响应体未取回，内容是否恶意：**未核实**。

## 6. CI

结论：**可疑**（Action 用浮动 tag）。未发现危险触发或把密钥写进仓库。

- 触发：`ci.yml:4-7` 为 `push` main 与 `pull_request`。`release.yml:2-3` 为 tag `v*`。`macos-install-smoke.yml:17-29` 为 `workflow_dispatch`、`push`/`pull_request` 且路径限于 `install.sh` 与该 workflow。未见 `pull_request_target`、`workflow_run`。
- Action 均为 tag，不是提交 SHA：`actions/checkout@v5`、`actions/setup-go@v6`、`actions/setup-node@v5`、`actions/upload-artifact@v4`、`docker/*@v3`–`v6`、`softprops/action-gh-release@v2`（`release.yml` 与 `ci.yml`）。
- 密钥：`release.yml:204-245` 引用 `secrets.DOCKERHUB_USERNAME`、`secrets.DOCKERHUB_TOKEN`、`secrets.GITHUB_TOKEN`。仓库文件里没有这些值。`ci.yml:9` 权限为 `contents: read`。未在日志里核对运行时是否打印密钥：**未核实**。

## 7. 公开漏洞库

查询 URL：`https://api.osv.dev/v1/querybatch`。时间：2026-11-26（香港时间）。范围：`go.sum` 108 + 两份 npm lock 766 与 85，共 959 个坐标。0 条 `MAL-`。41 个坐标有漏洞公告。未查 OSV 以外的商业恶意包库。未做可达性验证。

### 依赖是否恶意

结论：**未发现问题**（上述 959 个锁坐标没有 `MAL-`）。无 lock 的 `latest`／pip／管道脚本／镜像层：**未核实**。`replace` 的 `github.com/chu0119/Norma v0.4.4`（`go.mod:85`，`go.sum:13-14`）源码未审：**未核实**。

### 已知漏洞

结论：**已确认**锁定版本命中公告。下表「影响版本」指本次查询命中的锁定版本，不是公告的全部区间。`x/sys@v0.6.0` 与 `yaml.v3` 的 2020 伪版本只在 `go.sum` 的 `/go.mod` 行，选定版本是 `x/sys v0.47.0`、`yaml.v3 v3.0.1`。

| 依赖（锁定版本） | 编号（多的只列前几条） |
|---|---|
| `golang.org/x/crypto@v0.55.0` | `GO-2026-5932`、`GO-2026-6354`、`GO-2026-6355` |
| `golang.org/x/net@v0.58.0` | `GO-2026-6603` 等 7 条 |
| `github.com/klauspost/compress@v1.18.6` | `GHSA-259r-337f-4rfw`、`GO-2026-5841`、`GO-2026-6865` |
| `github.com/gorilla/websocket@v1.5.3` | `GO-2026-6278` |
| `google.golang.org/grpc@v1.71.0` | `GHSA-2v4p-qf9q-27wj` 等 10 条 |
| `go.opentelemetry.io/otel@v1.35.0` 及 sdk／trace／otlp exporter | `GO-2026-6975`、`GHSA-8wmf-6v46-5gfg` 等 |
| `next@16.2.9` | 25 条，含 `GHSA-2464-fhr4-w9xw`、`GHSA-4jqv-mc3x-m676` |
| `undici@7.28.0` | 23 条，含 `GHSA-2gqq-gqf2-x968` |
| `hono@4.12.27` | 14 条 |
| `sharp@0.34.5` | `GHSA-f88m-g3jw-g9cj`、`GHSA-rgj7-g3m4-5g8c`、`GHSA-wq5f-xc86-pv6w` |
| 其余 npm 命中 | `@hono/node-server@1.19.14`、`@modelcontextprotocol/sdk@1.29.0`、`postcss@8.4.31` 与 `@8.5.16`、`zod@3.23.8`／`3.25.76`／`4.4.3`（`GHSA-mrhx-c7xc-m3w5`）、`qs@6.15.3`、`fast-uri@3.1.3` 等，见同次 querybatch |

## 8. 维护状态

结论：**未发现问题**（抽查对象未归档、npm `latest` 未标 deprecated）。锁定版本落后于索引最新版，只说明未跟到最新，不是停更。索引时间 2026-11-26 11:30:43（香港时间）。

| 对象 | 本仓锁定 | 索引最新 | 证据 |
|---|---|---|---|
| `next` | `16.2.9`（lock） | `16.4.6`（2026-11-10 04:33 香港时间发布） | `https://registry.npmjs.org/next` |
| `react` / `react-dom` | `^19.2.7` 经 lock 钉死 | `19.3.1`（2026-10-15 00:47 香港时间） | 同上 registry |
| `husky` | `^9.1.7` | 仍是 `9.1.7`（2024-11-18 发布；registry `modified` 2026-09-03 03:31 香港时间） | `https://registry.npmjs.org/husky` |
| `puppeteer-core` | `^23.11.1` 经 lock | `25.19.0`（2026-11-26 01:50 香港时间） | registry |
| `chu0119/Norma` | `v0.4.4`（`go.mod:85`） | 模块 `v0.4.5`（2026-11-25 08:36 香港时间） | `https://proxy.golang.org/github.com/chu0119/norma/@latest`；仓库 `archived=false`，`pushed_at` 2026-11-25 08:52 香港时间 |
| `Autumn-27/Norma` | 被 replace 换掉 | 仓库仍在，`archived=false`，`pushed_at` 2026-11-24 18:19 香港时间 | GitHub API `repos/Autumn-27/Norma`。`go.mod:83-84` 注释写原仓曾删除，与这次元数据不一致 |
| `golang.org/x/crypto`、`x/net`、`pgx/v5`、`go-mitmproxy`、`klauspost/compress`、`modernc.org/sqlite` | 见 `go.mod`／`go.sum` | 均有 2026-11 的更新版本（如 crypto `v0.59.0`） | `https://proxy.golang.org/<module>/@latest` |

未抽查的传递依赖、Docker Hub／GHCR 标签维护者、以及未进锁的 GitHub `latest` 工具：**未核实**。
