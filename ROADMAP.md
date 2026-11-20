# Roadmap

This roadmap is driven by one question: **what do people reach for a
"utils collection" to do that the operating system does not already do
well?**

## Product principles

1. **Do not wrap what the OS already does well.** `curl`, `dig`, `whois`,
   `ping`, `grep`, `diff`, `openssl` and friends are already best in class.
   A thin wrapper only loses.
2. **Offline-first, single binary.** No API keys, no accounts, no network
   dependency for a tool to work. Network-based tools belong in `ssl` and
   `ipgeo`, where the network *is* the tool.
3. **Launcher-first.** Every tool ships `--json` result envelopes and a
   launcher registry entry, including Chinese aliases. A tool that cannot be
   invoked from Alfred/uTools is only half shipped.
4. **Data tables are the moat.** Tools backed by embedded, maintained data
   (license plates, qqwry, ID-card regions, phone prefixes) are hard to
   replace and impossible to reproduce with a shell one-liner.

## P0 — Developer daily tools (v0.11)

High frequency, no OS equivalent (or a bad one), low implementation cost.
Each tool ships as its own PR.

| Tool | Scope | Why |
|------|-------|-----|
| `hash` | md5, sha1, sha256, sha512, sm3, crc32, HMAC; strings, files, stdin | macOS and Linux disagree on `md5`/`md5sum`; no SM3, HMAC or CRC in the base system |
| `jwt` | Decode header/payload, inspect claims, verify HS/RS/PS/ES signatures | No OS tool; everyday debugging need |
| `otp` | TOTP/HOTP codes, `otpauth://` URI parsing | No OS tool; 2FA debugging |
| `qrdecode` | Decode QR images from a file or stdin | Complements `qrcode`, which can only generate |
| `text` | camel/snake/kebab/constant/title/slug, HTML entities, JSON escapes, full/half width, text stats | Doing this with `sed`/`awk` is painful and error-prone |
| `date ts` | Unix seconds/milliseconds ↔ RFC3339 ↔ timezones | `date` flags are inconsistent across platforms and hostile |
| `cron` | Explain a cron expression, list the next N runs | No OS tool; avoids off-by-one scheduling mistakes |

## P1 — Chinese offline data (v0.12)

The differentiator: Chinese users, offline data, launcher aliases.

| Tool | Scope |
|------|-------|
| `idcard` | Checksum, issuing region, birth date, gender, 15→18 digit upgrade |
| `phone` | Phone prefix → province/city and carrier |
| `pinyin` | Hanzi → pinyin, initials, polyphonic characters |
| `rmb` | Amount → RMB uppercase (`壹仟贰佰叁拾肆元伍角陆分`) |
| `date lunar` | Gregorian ↔ lunar calendar, solar terms, sexagenary cycle, zodiac |
| `ref` | HTTP status codes, MIME types, user-agent parsing, well-known ports |

## P2 — Structured data and media

Worth scheduling, not urgent:

- `json` extensions: JSONPath queries, JSON→Go, JSON→TypeScript, JSON diff.
- `barcode`: Code128/EAN-13 generation and decoding.
- `unit`: length, weight, area, volume, temperature, data size, bandwidth.
- `regex`: test, replace preview, named groups.
- `image`: compress, convert, resize, EXIF.
- `uuid` family: nanoid, ULID, snowflake.
- `base64` family: base32, base58, base85, hex.
- `fake`: test data (names, phone numbers, addresses, ID cards).
- Formatters: SQL, XML, HTML; `.env` parsing; `.gitignore` templates.

## Non-goals

- **Network tools**: `curl`, `dig`, `whois`, `ping`, `traceroute`, port
  scanning. Already excellent, and they dilute the offline promise.
- **Generic text clones**: `diff`, `grep`, `sed`. `ripgrep` and `diff` win.
  Only a JSON-aware diff belongs here.
- **Heavy dependencies**: PDF merge/split, OCR, image upscaling. Binary size
  and licensing costs outweigh the value.
- **Network-dependent utilities**: currency, weather, speed tests, IP
  reputation. API keys and flaky endpoints break the single-binary promise.
- **System administration**: process/disk/Docker/Kubernetes wrappers.

## Definition of done

Every tool must:

1. Put pure logic in `internal/<tool>/` with tests, and keep `cmd/<tool>.go`
   to argument parsing and I/O.
2. Support `--json` and return the standard result envelope.
3. Be registered in `internal/launcher` with a summary, example, input mode
   and Chinese aliases where useful.
4. Be documented in `README.md`, `README_zh.md`, `CLAUDE.md` and this file's
   status.
5. Pass `go test ./...`, `go build -trimpath -o lingbox .`, and the launcher
   E2E suite.
