# 供应链（基准 `44a36480df29a2ea353c3ff5f4d3a7a1e759c046`）

结论：Go 与 npm 锁文件把已解析的模块钉在摘要上，OSV 未返回恶意包 ID；镜像构建、首次启动和若干安装脚本仍会以 root 拉取未固定版本且多数无校验和的第三方代码。未拉取的镜像层不能写成已确认安全。详情与建议见文末「依赖是否恶意」。

查询时间：2026-11-26（香港时间，UTC+8）。OSV 使用 `https://api.osv.dev/v1/querybatch`，本环境当次请求；响应无独立时区字段。

## 锁文件

- Go：`go.mod` 声明 `go 1.26.3`，模块 `github.com/chu0119/pojun`。`go.sum` 含 108 个模块版本坐标（含仅有 `/go.mod` 哈希、未必打进二进制的旧版本）。直接依赖见 `go.mod` 第 6–22 行。
- 唯一 `replace`：`github.com/Autumn-27/norma => github.com/chu0119/Norma v0.4.4`（`go.mod` 第 83–85 行）。`go.sum` 第 13–14 行有该版本的 `h1` 摘要。注释写明原上游仓库曾删除、改指维护者 fork。本单未打开该 fork 源码。
- npm：`web/package-lock.json` 766 个不重复包，`resolved` 主机均为 `registry.npmjs.org`。`web/package.json` 直接依赖为 Next 16 / React 19 等常见前端包，版本范围多为 `^`，锁文件再钉死。`skills/api-recon/scripts/package-lock.json` 另有 85 个包，直接依赖是 `puppeteer-core`。
- 无 Python/Rust 锁文件。镜像内 `pip install` 不写锁。

## Docker 与 compose

| 位置 | 镜像 | tag | digest |
|---|---|---|---|
| `Dockerfile` 第 16 行 | `golang` | `1.26-bookworm` | 无 |
| `Dockerfile` 第 25 行 | `python` | `3.11-slim-bookworm` | 无 |
| `docker-compose.yml` 第 9 行 | `postgres` | `16-alpine` | 无 |
| `docker-compose.yml` 第 26 行 | `${POJUN_IMAGE:-ghcr.io/chu0119/pojun}` | `${POJUN_TAG:-latest}` | 无 |

`.env.example` 第 2 行默认 `POJUN_TAG=latest`。镜像层未拉取，tag 可被上游移动。

## 子模块

仓库无 `.gitmodules`。未发现 git 子模块。

## curl / wget 与管道执行

已读到的浮动或管道安装（均未在本环境执行）：

- `Dockerfile` 第 53 行：`curl -fsSL "$NODESOURCE_URL" | bash -`，默认 `https://deb.nodesource.com/setup_20.x`，无脚本校验和。
- `Dockerfile` 第 21–23 行：`git clone --branch v2.4.1` `Ackites/KillWxapkg`，tag 非 commit，无独立哈希。
- `Dockerfile` 第 18–19 行：`go install` kerbrute `v1.0.3`、waybackurls `v0.1.0`（版本固定，构建机不使用本仓 `go.sum`）。
- `Dockerfile` 第 75–80 行：nuclei 取 GitHub `releases/latest`，下载 zip 后无校验和。
- `Dockerfile` 第 87 行：`npm install -g @playwright/mcp@latest @playwright/cli@latest playwright@latest`。
- `Dockerfile` 第 114 行：`pip install uv cryptography`，无版本。
- `docker/install-tools.sh` 第 63–97 行：一批 GitHub 工具取 `releases/latest`，解压后 `install` 到 `/usr/local/bin`，无校验和。第 108–109 行 pip 安装 `wafw00f`、`semgrep`、`pwntools`，无版本。同文件后部从 `impacket` 的 `master` 打包安装。
- `hosttools/hosttools.go`：目录内多处 `releases/latest`（例如第 296 行 winPEAS、第 346–347 行 kubectl 的 `stable.txt`）。容器首次启动由 `docker/docker-entrypoint.sh` 第 39–47 行后台执行 `pojun-hostd -install-all`（默认开启）。
- `install.sh` 第 682 行对**本项目发布 zip** 用同一次下载的 `SHA256SUMS` 做 `sha256sum -c`。第 900、1598 行安装 Homebrew 时执行 `curl …/Homebrew/install/HEAD/install.sh`，`HEAD` 未固定。
- `selfupdate/github.go` 第 20–35 行：更新源是构建期变量，默认 `chu0119/pojun`；重定向主机白名单为 GitHub 域。`selfupdate/stage.go` 用 Release 里的 `SHA256SUMS` 比对。校验和与制品同源，无单独签名。

mihomo 替代源（`mihomo/install_sources.go` 第 39–45 行）含 `gh-proxy.com`、`ghfast.top`、`ghproxy.net`。注释写明只在拿到 GitHub API 的官方 SHA-256 后启用，哈希不符则换源。自定义 URL 的 sha256 是可选的（同文件第 29 行）。

## 仿冒包名与安装钩子

- 直接 Go 模块与 npm 包名未见明显仿冒（无 `node-ipc`、`event-stream`、`flatmap-stream`、`peacenotwar` 等已知恶意名）。`colors` 的命中是 `ansi-colors` / `picocolors` / `yoctocolors`，不是 2022 年事件中的 `colors` 包。
- `web/package.json` 第 17 行 `"prepare": "husky"`。这是上游 husky 的安装钩子；本单未执行 `npm install`，未观察钩子实际行为。
- Go 模块无 install 脚本。

## 非官方源

- apt/pip 默认官方，Dockerfile 允许用构建参数改 Debian/PyPI 镜像。
- mihomo 字节搬运可走上述三个第三方前缀，完整性依赖 GitHub API 给出的 SHA-256。
- 运行时 `llmpresets/modelsdev.go` 第 31 行 `GET https://models.dev/api.json`（模型元数据，失败则沿用内嵌快照）。`server/hostenv_api.go` 第 332 行与 `server/proxypool_api.go` 的代理探测默认打 `https://api.ipify.org?format=json`，只在探测接口被调用时发出，不是进程启动即外联。

## OSV / GHSA

对 `go.sum` 108 个版本、`web/package-lock.json` 766 个包、`skills/api-recon/scripts/package-lock.json` 85 个包，共 959 个坐标查询 OSV。

- 恶意包 ID（`MAL-`）：0。
- 至少一条漏洞公告的坐标：41。唯一公告 ID 166 个。未做调用路径可达性验证，不把公告写成已确认可利用的产品漏洞。
- 选定进模块图、且本仓有直接 import 的例子（公告摘要取自 OSV，2026-11-26 当次）：
  - `golang.org/x/crypto@v0.55.0`：`GO-2026-6354` / `GO-2026-6355`（`ssh` 通道死锁 DoS）、`GO-2026-5932`（`openpgp` 子包被标为不再维护）。本仓 `egress/sshjump.go` 引用 `golang.org/x/crypto/ssh`，`server/auth.go` 引用 `bcrypt`。未看到本仓 import `openpgp`。
  - `golang.org/x/net@v0.58.0`：`GO-2026-6603` 等（HTTP/2 Trailer 内存消耗）。未验证本进程是否走到该路径。
  - `github.com/klauspost/compress@v1.18.6`：`GO-2026-5841`（s2 越界读）、`GO-2026-6865`（帧解析 panic）。本仓直接 import 的是 `zstd` 子包（`server/skill_zip.go`）。
  - `github.com/gorilla/websocket@v1.5.3`：`GO-2026-6278`（掩码密钥用弱 PRNG）。传递依赖。
  - `go.opentelemetry.io/otel@v1.35.0` 及 exporter：含 `GO-2026-6975`、`GHSA-8wmf-6v46-5gfg`（baggage 内存放大、exporter 日志可能记下端点 URL）。默认不启用 OTLP（见主报告）。
- `go.sum` 里只有 `/go.mod` 行、没有对应代码 `h1` 的旧版本（如 `golang.org/x/sys v0.6.0`、`gopkg.in/yaml.v3` 的 2020 伪版本）也会被查询命中。选定版本分别是 `x/sys v0.47.0` 与 `yaml.v3 v3.0.1`。
- npm 侧命中包括锁定的 `next@16.2.9`、`hono`、`undici`、`postcss` 等。未区分运行时与开发依赖，未验证前端是否暴露对应路径。

GitHub Actions（`.github/workflows/*.yml`）使用 `actions/*@v5`、`docker/*@v3`–`v6`、`softprops/action-gh-release@v2` 等 tag，未钉 commit SHA。影响 CI，不直接进入用户二进制。

## 依赖是否恶意

结论：在已查询的 959 个锁文件坐标上，没有 OSV 恶意包记录，也没有看到本仓第一方代码把凭据发往隐藏的第三方地址。这不等于整条供应链无恶意。

- 已排除到的范围：`go.sum` 与两份 npm lock 的包名+版本；第一方 Go/脚本里的 URL 主机抽查（GitHub、NVD、CISA、models.dev、ipify、OpenAI/Anthropic 官方 API、mihomo 镜像前缀）。遥测默认关闭，见 `telemetry/telemetry.go` 第 31–45 行。
- 不能下「无恶意」结论的部分：
  - 未拉取 `golang`、`python`、`postgres`、`ghcr.io/chu0119/pojun` 的镜像层。
  - Dockerfile / `install-tools.sh` / `hosttools` 在构建或首次启动时执行的 `latest`、`@latest`、无版本 pip、NodeSource 管道脚本、KillWxapkg tag 克隆，字节未取回、未对内容做恶意代码审查。这些步骤在镜像里以 root 运行。
  - `replace` 指向的 `github.com/chu0119/Norma v0.4.4` 只核对了 `go.sum` 摘要存在，未读源码。Agent 的 Bash 与 PTY 工具实现在该模块。
  - 第三方 gh 代理只验证「与 GitHub API 公布的 SHA-256 一致」。若发布资产与 API 摘要同时被替换，镜像搬运发现不了。
  - 未查询未进入锁文件的运行时下载，也未查询 OSV 以外的商业恶意包库。查不到的项记为未核实，不记为安全。
