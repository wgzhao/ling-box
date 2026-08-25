# lingbox

玲珑盒，一款实用的开发者工具集。

> [English Documentation](README.md)

## 安装

### Homebrew（macOS / Linux）

```bash
brew tap wgzhao/tap
brew install lingbox
```

### 脚本安装（macOS / Linux）

自动检测系统与架构，下载最新预编译二进制（无需 Go）：

```bash
curl -fsSL https://raw.githubusercontent.com/wgzhao/ling-box/master/install.sh | bash
```

可用 `--install-dir <dir>`（或环境变量 `LINGBOX_INSTALL_DIR`）指定安装目录，
或用 `LINGBOX_VERSION`（如 `LINGBOX_VERSION=0.5.0`）固定版本。下载后按
release 摘要校验 SHA-256。

### 源码安装

需要 [Go](https://go.dev/dl/) 1.26 或更高版本，参见下方 [构建](#构建)。

## 功能特性

- **URL 编码/解码**: 对 URL 字符串进行编码和解码
- **Base64 编码/解码**: 对 Base64 字符串进行编码和解码（支持 URL-safe 模式）
- **BCrypt 密码哈希**: 生成和验证 bcrypt 密码哈希
- **二维码生成**: 生成 PNG、JPG 或 GIF 格式的二维码图片
- **密码生成**: 生成可定制选项的安全随机密码
- **UUID 生成**: 批量生成 UUID（支持 v1、v4、v7）
- **格式转换**: JSON、YAML、CSV、Markdown 四格式互转（pandoc 风格 `-i`/`-o`/`-t`）
- **JSON 格式化与校验**: 重新缩进 JSON、校验 JSON 语法（`json format`、`json verify`）
- **Unicode 编解码**: 文本与 \uXXXX 转义序列互转
- **颜色转换**: 在 Hex、RGB、HSL 和颜色名称之间互转
- **BMI 计算**: 根据身高体重计算身体质量指数
- **进制转换**: 在二进制、八进制、十进制和十六进制之间转换
- **日期计算**: 日期加减天数或计算日期差值
- **终端图片显示**: 在终端中直接显示图片（支持 iTerm2/Kitty/half-block/ASCII）
- **终端 PDF 浏览**: 在终端中渲染和翻页浏览 PDF 文件
- **SSL 证书查看**: 解析并查看 X.509 证书详情（主题、签发者、密钥强度、有效期、扩展）
- **SSL 主机扫描**: 扫描主机支持的 TLS 协议版本与加密套件（含安全评级），并检测证书信任状态
- **车牌归属地查询**: 按省份查询中国车牌代码（`plate`）
- **IPv4 子网计算**: 计算网络、广播地址、通配掩码与主机范围，支持子网/超网划分、地址段去聚合与子网拆分（`ipcalc`）

## 环境要求

- Go 1.26 或更高版本

## 构建

```bash
go build -o lingbox .
```

## 运行

```bash
./lingbox <command> [options]
```

## 使用说明

### 查看帮助

```bash
./lingbox --help
```

### URL 编码/解码

```bash
# 编码 URL 字符串
./lingbox url -e 'hello world'
# 输出: hello+world

# 解码 URL 字符串
./lingbox url -d 'hello+world'
# 输出: hello world
```

### Base64 编码/解码

```bash
# 编码为 Base64
./lingbox base64 -e 'Hello World'
# 输出: SGVsbG8gV29ybGQ=

# 解码 Base64 字符串
./lingbox base64 -d 'SGVsbG8gV29ybGQ='
# 输出: Hello World

# URL-safe 模式（-u 需放在 -e 之前）
./lingbox base64 -u -e 'test+/string'
```

### BCrypt 密码哈希

```bash
# 生成 bcrypt 哈希
./lingbox bcrypt -g mypassword
# 输出: $2a$12$...

# 验证密码与哈希是否匹配
./lingbox bcrypt -v mypassword '$2a$12$...'
```

### 二维码生成

```bash
# 生成二维码（默认: qrcode.png, 300x300）
./lingbox qrcode 'https://example.com'

# 自定义输出文件和尺寸
./lingbox qrcode 'Hello World' -o mycode.png -s 500

# 不同图片格式
./lingbox qrcode 'Test' -o mycode.jpg -f JPG
```

### 密码生成

```bash
# 生成 16 位密码（默认）
./lingbox password

# 生成 24 位密码
./lingbox password -l 24

# 生成多个密码
./lingbox password -c 5

# 纯数字密码
./lingbox password -d

# 纯大写字母密码
./lingbox password -u

# 不含特殊字符的密码
./lingbox password -n
```

### UUID 生成

```bash
# 生成一个 v4 UUID（默认）
./lingbox uuid

# 生成 5 个 UUID
./lingbox uuid -n 5

# 生成 UUID v7（时间有序、可排序）
./lingbox uuid -t v7

# 生成 UUID v1（基于时间）
./lingbox uuid -t v1
```

### 格式转换 (convert)

在 JSON、YAML、CSV、Markdown 四种格式间互转，采用 pandoc 风格的
`-i`/`-o`/`-t` 参数。输入格式由 `-i` 文件后缀决定（stdin 时自动检测）；
目标格式由 `-t` 指定，省略时按 `-o` 文件后缀猜测。

输入格式：`json`、`yaml`/`yml`、`csv`。输出格式：`json`、`yaml`、`csv`、
`markdown`。

CSV 输出要求顶层为对象数组（首行表头）或标量数组；Markdown 输出还支持
标量数组（无序列表）和顶层对象（键值两列表格）。单元格中的嵌套对象/数组
会编码为紧凑 JSON 字符串。CSV 输入以首行为表头，单元格保持字符串（不推断
类型）。

```bash
# JSON 转 YAML（目标格式按 -o 后缀猜测）
./lingbox convert -i data.json -o data.yaml

# CSV 转 Markdown
./lingbox convert -i data.csv -t markdown

# JSON 转 CSV
./lingbox convert -i data.json -o data.csv

# 从标准输入读取（自动检测格式：JSON → YAML → CSV）
cat data.json | ./lingbox convert -t yaml
curl -s https://api.example.com/data | ./lingbox convert -o out.yaml

# 输出 Markdown 表格
./lingbox convert -i data.yaml -o table.md
```

### JSON 工具 (json)

```bash
# 格式化（重新缩进）JSON，保持 key 原始顺序
./lingbox json format data.json
cat data.json | ./lingbox json format --indent 4

# 按键排序和/或压缩为单行
cat data.json | ./lingbox json format --sort-keys --compact

# 校验 JSON 语法（失败时退出码为 1，并报告行列位置）
./lingbox json verify data.json
cat data.json | ./lingbox json verify -q   # 成功时静默
```

### Unicode 编解码

```bash
# 中文转 \uXXXX
./lingbox unicode -e '你好世界'
# 输出: 你好世界

# \uXXXX 转回中文
./lingbox unicode -d '你好世界'
# 输出: 你好世界

# 自动识别模式（纯文本→编码，含\u→解码）
./lingbox unicode '你好'
```

### 颜色转换

```bash
# 颜色名称
./lingbox color 'red'

# Hex 格式
./lingbox color '#FF0000'

# RGB 格式
./lingbox color 'rgb(255, 0, 0)'

# HSL 格式
./lingbox color 'hsl(0, 100%, 50%)'

# 多词颜色名称
./lingbox color 'dark gray'
./lingbox color 'light yellow'
```

### BMI 计算

```bash
# 计算 BMI（身高 cm，体重 kg）
./lingbox bmi 170 65
# 输出:
#   Height: 170.0 cm
#   Weight: 65.0 kg
#   BMI: 22.5 (Normal weight)
```

### 进制转换

```bash
# 十进制转其他进制
./lingbox base 255

# Hex 转其他进制
./lingbox base FF --from hex

# 带前缀自动识别
./lingbox base "0xFF"

# 二进制转其他进制
./lingbox base 1010 -f bin
```

### 日期计算

```bash
# 日期加减天数（负数用 -- 分隔）
./lingbox date add 2026-01-01 10
./lingbox date add 2026-01-01 -- -30

# 计算两个日期差值
./lingbox date diff 2026-01-01 2026-07-26

# 计算精确到时间的差值
./lingbox date diff "2026-01-01 12:00:00" "2026-01-02 14:30:00"
```

### 终端图片显示 (imgcat)

在终端中直接显示图片。默认使用 OSC 1337（iTerm2 协议），支持 iTerm2、WezTerm、
Warp、kaku、kitty（兼容模式）、VS Code 终端等，输出无损画质。

```bash
# 显示图片（自动选择最佳渲染方式）
./lingbox imgcat photo.jpg

# 指定输出宽度（字符列数）
./lingbox imgcat photo.png -w 60

# 手动指定渲染器
./lingbox imgcat photo.jpg -r halfblock   # ANSI ▀ 字符（通用兼容）
./lingbox imgcat photo.jpg -r iterm2      # OSC 1337 无损（默认）
./lingbox imgcat photo.jpg -r kitty       # Kitty 原生协议
./lingbox imgcat photo.jpg -r ascii       # 灰度字符画

# 从标准输入读取
cat photo.png | ./lingbox imgcat

# 交互式浏览多张图片（方向键/空格翻页，q 退出）
./lingbox imgcat photo1.jpg photo2.jpg photo3.png

# 自动展开通配符（适用于引号包裹的参数）
./lingbox imgcat "875*.png"
./lingbox imgcat photo1.jpg "screenshot-*.png"

```

### 终端 PDF 浏览 (pdf)

在终端中直接渲染并浏览 PDF 文件，渲染方式与 imgcat 相同。

```bash
# 交互式翻页浏览（方向键翻页，q 退出）
./lingbox pdf document.pdf

# 显示指定页
./lingbox pdf -p 1 document.pdf

# 指定渲染方式和宽度
./lingbox pdf -r ascii -w 80 document.pdf
./lingbox pdf -r halfblock document.pdf

# 更高 DPI 提升清晰度
./lingbox pdf --dpi 300 document.pdf

# 从标准输入读取（渲染第 1 页）
cat document.pdf | ./lingbox pdf
```

### SSL 工具 (ssl)

#### 证书查看 (ssl cert)

解析并展示 X.509 证书的详细信息：主题与签发者、签名算法、公钥及密钥
强度、有效期（含是否过期）、扩展信息（SAN、密钥用途、SKI/AKI、OCSP、
CRL、证书策略等）。

```bash
# 从文件读取（支持 PEM 或 DER 格式的 .crt 文件）
./lingbox ssl cert server.crt
./lingbox ssl cert -f fullchain.pem

# 直接输入 PEM 内容（单行、带 \n 转义也可）
./lingbox ssl cert '-----BEGIN CERTIFICATE-----...'

# 从标准输入读取
cat cert.pem | ./lingbox ssl cert
```

#### 主机扫描 (ssl host)

连接目标主机，报告其支持的 TLS 协议版本、各版本接受的加密套件（INSECURE
红色 / WEAK 黄色标注）、以及所出示证书的有效期、信任状态和主机名匹配情况。

```bash
# 默认使用 443 端口
./lingbox ssl host www.baidu.com

# 指定端口
./lingbox ssl host https://www.baidu.com:8443
./lingbox ssl host example.com:8443

# 自定义握手超时（默认 5 秒）
./lingbox ssl host www.baidu.com -t 10

# 静默模式：不显示进度（用于批处理/脚本）
./lingbox ssl host www.baidu.com -q
```

扫描进度默认在终端上显示，输出到 stderr（重定向 stdout 不会污染报告）；
stderr 不是终端时自动关闭进度。

说明：Go 的 TLS 客户端不支持 SSL 3.0，无法探测该协议；TLS 1.3 的加密
套件不可逐套件协商，按组列出。

### 车牌归属地查询 (plate)

按省份或直辖市查询车牌代码。参数支持简称（湘）、全称（湖南省）或不带
行政后缀的名称（湖南）。

```bash
# 查询单个省份
./lingbox plate 湘
./lingbox plate 湖南省
./lingbox plate 湖南

# 分页列出全国 31 个省级行政区（Enter 下一页，q 退出）
./lingbox plate

# 调整每页省份数量（交互模式）
./lingbox plate -n 10

# 非交互：管道或重定向输出时一次性全部打印
./lingbox plate | grep 湘
```

### IPv4 子网计算 (ipcalc)

移植自 [Krischan Jodies 的 ipcalc](http://jodies.de/ipcalc)（v0.41）。输入
IP 地址与掩码，计算网络地址、广播地址、Cisco 通配掩码与主机范围，并以易
读的二进制形式展示。指定第二个掩码可以划分子网或超网。

```bash
# 基本计算（省略掩码时默认 /24）
./lingbox ipcalc 192.168.0.1/24

# 支持点分十进制与通配（反）掩码
./lingbox ipcalc 192.168.0.1/255.255.128.0
./lingbox ipcalc 192.168.0.1 0.0.63.255

# 子网划分（第二个掩码大于第一个）
./lingbox ipcalc 192.168.0.1 255.255.255.0 255.255.255.128

# 超网合并（第二个掩码小于第一个）
./lingbox ipcalc 10.0.0.1 255.255.255.0 255.255.0.0

# 地址段去聚合为 CIDR 块
./lingbox ipcalc 192.168.1.10 - 192.168.2.5
./lingbox ipcalc 192.168.1.10 192.168.2.5 -r

# 按主机数拆分网络为子网
./lingbox ipcalc 10.0.0.0/24 -s 100 50 25

# 只打印地址所属类别的自然掩码
./lingbox ipcalc -c 192.168.0.1   # -> 24

# 关闭二进制显示 / 关闭颜色
./lingbox ipcalc 192.168.0.1/24 -b
./lingbox ipcalc 192.168.0.1/24 -n
```

颜色在终端下自动开启（遵循 `NO_COLOR`），可用 `-n` 关闭。原脚本残留的
两行调试输出（"WILDCARD" 与 "INVALID NETMASK"）有意未移植。

## Shell 自动补全

内置 shell 补全支持，按 Tab 即可自动补全命令和参数：

```bash
# Bash
source <(./lingbox completion bash)

# Zsh
source <(./lingbox completion zsh)

# Fish
mkdir -p ~/.config/fish/completions
./lingbox completion fish > ~/.config/fish/completions/lingbox.fish

# PowerShell
./lingbox completion powershell | Out-String | Invoke-Expression
```

启用后，输入 `./lingbox con<Tab>` 即可自动补全为 `convert`。

## 跨平台支持

本工具用 Go 编写，可编译为单个静态二进制文件，支持以下平台：

- Windows (x86-64)
- macOS (x86-64, ARM64)
- Linux (x86-64, ARM64)

## 发布流程

打标签并推送即可触发自动发布：

```bash
git tag v1.0.0
git push origin v1.0.0
```

GitHub Actions 会自动：
1. 构建各平台二进制（Linux x86-64/ARM64、Windows x86-64、macOS x86-64/ARM64）
2. 根据提交记录生成发布说明
3. 创建 GitHub Release 并上传所有二进制文件

## 命令速查

| 命令 | 用途 | 示例 |
|------|------|------|
| `url` | URL 编解码 | `url -e '你好'` |
| `base64` | Base64 编解码 | `base64 -e 'Hello'` |
| `bcrypt` | 密码哈希 | `bcrypt -g mypass` |
| `qrcode` | 二维码生成 | `qrcode 'text' -o qr.png` |
| `password` | 随机密码 | `password -l 24 -c 5` |
| `uuid` | UUID 生成 | `uuid -n 5 -t v7` |
| `convert` | 格式转换（JSON/YAML/CSV/Markdown） | `convert -i data.json -o data.yaml` |
| `json format` | JSON 格式化 | `json format data.json` |
| `json verify` | JSON 语法校验 | `json verify data.json` |
| `unicode` | Unicode 编解码 | `unicode -e '你好'` |
| `color` | 颜色转换 | `color '#FF0000'` |
| `bmi` | BMI 计算 | `bmi 170 65` |
| `base` | 进制转换 | `base FF -f hex` |
| `date` | 日期计算 | `date diff 2026-01-01 2026-07-26` |
| `imgcat` | 终端图片显示 | `imgcat photo.jpg` |
| `pdf` | 终端 PDF 浏览 | `pdf document.pdf` |
| `ssl` | SSL 证书查看 | `ssl cert server.crt` |
| `ssl host` | TLS 主机扫描 | `ssl host www.baidu.com` |
| `plate` | 车牌归属地查询 | `plate 湘` |
| `ipcalc` | IPv4 子网计算 | `ipcalc 192.168.0.1/24` |

完整帮助：`lingbox <command> --help`

## 开源协议

Apache License 2.0
