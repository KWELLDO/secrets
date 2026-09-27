# secrets — 本地密钥工具（Go）

`~/.secrets` 是一个**单文件、纯文本、手工可编辑**的密钥表：一行一条 `name=value`。
这个项目是它的 **解析库 + `key` 命令行工具**，并额外支持**创建/修改时间**和**备注**。

```
~/.secrets                      ← 唯一的数据文件，权限 600
~/Projects/secrets/             ← 这个项目（模块 gitee.com/kwelldo/secrets）
```

格式规范：`~/Documents/secrets-format-spec.md`

---

## 安装

```sh
cd ~/Projects/secrets
./install.sh                         # 装到 ~/go/bin/key
./install.sh --prefix ~/.local/bin   # 或指定目录
./install.sh --uninstall             # 卸载
```

脚本会检查 Go 版本、构建、验证 PATH 里没有别的 `key` 遮蔽它、跑一次自检，并提醒
`~/.secrets` 权限不是 600 的情况。

装完直接：`key help`

---

## 30 秒上手

```sh
key list                          # 我有哪些密钥
key get openai_api_key            # 取一个值（脚本里最常用）
key set deepseek_api_key=sk-xxx   # 新增或更新
key info deepseek_api_key         # 看它的值 + 时间 + 备注
key rm deepseek_api_key           # 删掉
```

---

## 命令参考

| 命令 | 说明 | 例子 |
|---|---|---|
| `key path` | 打印密钥文件路径 | `key path` |
| `key list` | 列出键名（按名排序） | `key list` |
| `key list -l` | 附带创建/修改时间、备注名 | `key list -l` |
| `key get <名>` | 打印值，**只输出值**；不存在退出码 1 | `key get github_token` |
| `key copy <名>` | 把值复制到系统剪贴板（每平台各自的原生方式，见下） | `key copy github_token` |
| `key info <名>` | 值 + 创建/修改时间（含"32d ago"）+ 备注 | `key info dsh` |
| `key set <名>=<值>` | 新增或更新（**原地更新**，不产生重复行） | `key set foo=bar` |
| `key rm <名>` | 删除条目**及它的备注** | `key rm foo` |
| `key note <名>` | 列出该条的备注 | `key note dsh` |
| `key note <名> <标签>=<值>` | 加/改备注 | 见下 |
| `key unnote <名> <标签>` | 删单条备注 | `key unnote dsh account` |
| `key backfill [<时间>]` | 给缺时间的旧条目补日期（默认用文件 mtime） | `key backfill 2026-08-09` |
| `key help` | 帮助 | |

别名：`ls`=`list`、`del`/`delete`/`unset`=`rm`、`meta`/`show`=`info`、`cp`=`copy`。

**读操作用 `key get`，看全貌用 `key info`。**

`key copy` 复制**原始值**（不带尾部换行）；剪贴板方式**按平台分文件编译**（GOOS 决定，不会跨平台误调）：

| 平台 | 用的工具 |
|---|---|
| Linux / BSD | Wayland `wl-copy` → X11 `xclip` → `xsel`（按会话环境探测） |
| macOS | `pbcopy`（系统自带） |
| Windows | `clip.exe`（系统自带） |

任何时候都可用 `KEY_CLIPBOARD` 覆盖整套探测，指向自己的工具：

```sh
KEY_CLIPBOARD="xclip -selection clipboard" key copy foo
# Windows 上需要非 ASCII 精确时（clip.exe 走控制台代码页）
KEY_CLIPBOARD="powershell -NoProfile -Command Set-Clipboard" key copy foo
```

`install.sh` 会同时装 bash tab 补全（`~/.local/share/bash-completion/completions/key`），新 shell 里 `key <Tab>` 可补子命令、`key copy <Tab>` 补键名（补全是 bash 脚本，Windows 原生命令行不适用）。

---

## 备注怎么用

备注是**有标签的键值对**，一个条目可以挂任意多个：

```sh
key note deepseek_api_key base_url=https://api.deepseek.com
key note deepseek_api_key account=kwelldo
key note github_token note="读权限即可，2026-09 换过"
```

推荐标签：`base_url` / `account` / `url` / `email` / `note` / `source`。

- 标签不重复；**同名标签会原地覆盖**（不会新增一条）
- 值可以含空格和 `=`：`key note x note="hi there a=b"`
- 改备注会把「修改时间」刷新

`key info` 的输出长这样：

```
name:      dsh
value:     sk-4b42…
created:   2026-08-09 19:55:50 (32d ago)
modified:  2026-09-10 20:34:41 (3m ago)
notes:
  base_url  https://api.deepseek.com
```

---

## 典型场景

**① 新加一个 API key，并记下它的 base_url / 账号**

```sh
key set kling_access_key=AQf3xxxx
key note kling_access_key base_url=https://api.klingai.com
key note kling_access_key account=我的账号
```

**② 从别处的明文文件批量搬进来**

```sh
while IFS=: read -r name val; do key set "${name}=${val}"; done < 旧文件.txt
```

**③ 给一批老条目补日期**

```sh
key backfill "2026-08-09 19:55:50"     # 也可只写 2026-08-09
# 只补「时间未知」的条目，已有的不会被覆盖
```

**④ 月度检查：哪些密钥很久没动 / 都挂在什么地址上**

```sh
key list -l                             # 看时间列和备注列
key list -l | awk '$3 < "2026-06"'      # 按修改时间筛
```

---

## 在脚本 / 程序里用

**Shell** —— `get` 只输出值，直接当变量用：

```sh
TOKEN="$(key get github_token)"
curl -H "Authorization: Bearer $TOKEN" …
```

**Go 库**：

```go
import "gitee.com/kwelldo/secrets"

m, _ := secrets.Load("~/.secrets")        // 只要键值：map[string]string
f, _ := secrets.LoadFile("~/.secrets")    // 完整视图：值 + 时间 + 备注
e := f.Entry("openai_api_key")
e.Value; e.Created; e.Note("base_url")
```

**pi 的 LLM 工具** —— `get_secret` / `list_secrets` 读的是同一份文件，但**只认
`name=value`，看不到备注**（备注存在 `#@` 注释行里）。要备注就用 `key info`。

**环境变量** `SECRETS_FILE` 可以切换文件（测试/多份密钥时有用）：

```sh
SECRETS_FILE=/tmp/test.secrets key set a=1
```

---

## 文件长什么样

```
# migrated from ai-key.txt
deepseek_api_key=sk-xxx
#@meta deepseek_api_key created=2026-08-09T19:55:50+08:00 modified=2026-09-10T20:34:41+08:00
#@note deepseek_api_key base_url=https://api.deepseek.com
#@note deepseek_api_key account=kwelldo
```

解析规则（与旧版完全兼容）：按**第一个** `=` 切分、两侧空白去掉、空行与 `#` 开头的行
忽略、重名**后者生效（last wins）**、值可含 `=` 和 `#`、允许空值、兼容 CRLF。

**注释行里全是元数据，所以任何只认 `name=value` 的旧程序都不会被破坏。**

- `key` 写文件是**原子**的（临时文件 + rename），且**强制 600**
- 重写时**注释、空行、条目顺序原样保留**；手写的注释不会被吃掉
- 删除条目会连它的 `#@meta` / `#@note` 一起删；手写遗留的孤立元数据行按普通注释保留

---

## 安全须知（重要，都是踩过的坑）

这个方案的选择是：**明文 + 权限 600，不加密**。理由和边界：

- **0600 是唯一防线**。明文存储下，文件权限就是全部。
- **不要指望靠加密这个文件来变安全**。它必须被**无人值守**读取（脚本、LLM 工具），
  所以密钥必须能被"以你身份运行的任何进程"拿到——加密只挡得住"文件被拷走"，挡不住本机进程。
- **真正的风险不是这个文件，而是密钥被复制出去。** 实际踩到的坑：
  - ❌ **把密钥硬编码进源码，然后推到公开仓库** —— 曾有两个公开仓库因此泄露活密钥
  - ❌ 密钥被写进对话/工具日志、编辑器历史、`~/.bash_history`
  - ❌ 把密钥拷到 `~/Documents`、Obsidian 库等会被同步/备份的地方
- **轮换是唯一能让所有历史副本一次性失效的手段**。散落在 300+ 个日志里的旧值，
  改一处配置是清不掉的，但换个新 key，旧值全部作废。
- **永远不要 `git commit` 这个文件**。它的正确位置就是 `$HOME`，别放进 dotfiles 仓库或网盘。

**多台机器怎么办**（比如手机 Termux、小主机）：

| 你的情况 | 建议 |
|---|---|
| 各机器独立，只存自己需要的 | 各自一份 `~/.secrets` 即可，**推荐** |
| 手动拷同一份过去 | 能接受，但明文会同时存在于多台机器 |
| 用 git / 网盘同步 | **不要同步明文**，这时才需要口令加密（age/pass） |

原则：**每台机器只放它真正要用的那几个**，别把生产密钥发到不需要它的设备上。

---

## 项目结构

```
~/Projects/secrets/
├── secrets.go              库：解析 / 修改 / 保留注释地重写
├── secrets_test.go         格式解析测试
├── metadata_test.go        元数据、删除、备份快照测试
├── cmd/key/main.go         CLI
├── cmd/key/clipboard.go        剪贴板入口 + KEY_CLIPBOARD 覆盖 + exec 助手
├── cmd/key/clipboard_unix.go   Linux/BSD: wl-copy / xclip / xsel
├── cmd/key/clipboard_darwin.go macOS: pbcopy
├── cmd/key/clipboard_windows.go Windows: clip.exe
├── completions/key.bash    bash tab 补全
├── install.sh              安装脚本（构建 + 补全）
└── README.md
```

### 跨平台构建

剪贴板按 GOOS 分文件（`//go:build linux || ...` / `darwin` / `windows`），普通 `go build` 就是对的；交叉编译也不需要 cgo 或额外依赖：

```sh
GOOS=windows GOARCH=amd64 go build -o key.exe ./cmd/key
GOOS=darwin  GOARCH=arm64 go build -o key-mac ./cmd/key
GOOS=linux   GOARCH=arm64 go build -o key-arm64 ./cmd/key
```

```sh
go test ./...     # 跑测试
gofmt -l .        # 应为空
./install.sh      # 安装
```

---

## 许可证

[AGPL-3.0](LICENSE)（GNU Affero General Public License v3.0）。
