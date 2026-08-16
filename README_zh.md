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
- **YAML/JSON 互转**: YAML 与 JSON 格式双向转换
- **Unicode 编解码**: 文本与 \uXXXX 转义序列互转
- **颜色转换**: 在 Hex、RGB、HSL 和颜色名称之间互转
- **BMI 计算**: 根据身高体重计算身体质量指数
- **进制转换**: 在二进制、八进制、十进制和十六进制之间转换
- **终端图片显示**: 在终端中直接显示图片（支持 iTerm2/Kitty/half-block/ASCII）
- **SSL 证书查看**: 解析并查看 X.509 证书详情（主题、签发者、密钥强度、有效期、扩展）
- **SSL 主机扫描**: 扫描主机支持的 TLS 协议版本与加密套件（含安全评级），并检测证书信任状态

## 环境要求

- Go 1.21 或更高版本

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

### YAML/JSON 互转

```bash
# JSON 转 YAML（从文件）
./lingbox json2yaml data.json

# YAML 转 JSON（从文件）
./lingbox yaml2json config.yaml

# 通过管道传输
cat data.json | ./lingbox json2yaml
curl -s https://api.example.com/data | ./lingbox json2yaml
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

启用后，输入 `./lingbox yam<Tab>` 即可自动补全为 `yaml2json`。

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
| `json2yaml` | JSON → YAML | `json2yaml data.json` |
| `yaml2json` | YAML → JSON | `yaml2json config.yaml` |
| `unicode` | Unicode 编解码 | `unicode -e '你好'` |
| `color` | 颜色转换 | `color '#FF0000'` |
| `bmi` | BMI 计算 | `bmi 170 65` |
| `base` | 进制转换 | `base FF -f hex` |
| `date` | 日期计算 | `date diff 2026-01-01 2026-07-26` |
| `imgcat` | 终端图片显示 | `imgcat photo.jpg` |
| `ssl` | SSL 证书查看 | `ssl cert server.crt` |
| `ssl host` | TLS 主机扫描 | `ssl host www.baidu.com` |

完整帮助：`lingbox <command> --help`

## 开源协议

Apache License 2.0
