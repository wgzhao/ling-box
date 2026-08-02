# lingbox

玲珑盒，一款实用的开发者工具集。

> [English Documentation](README.md)

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
```

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

完整帮助：`lingbox <command> --help`

## 开源协议

Apache License 2.0
