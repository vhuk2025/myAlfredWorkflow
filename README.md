# myAlfredWorkflow

用 Go 编写 Alfred workflow。`cmd/<name>` 是一个 workflow，编译出的二进制和 `info.plist` 一起打包成 `.alfredworkflow`。

## 当前时间 `tt`

在 Alfred 里输入 `tt` 列出当前时间，回车复制到剪贴板。`tt` 后面也可以写时间，缺省的月和日按 1、分和秒按 0 处理，仍然列出全部格式。

- `tt`
- `tt 2025-01`
- `tt 2025-02-02 05`
- `tt 2025-02-02 05:04:05`
- `tt 1738454400`

## 格式化 JSON `json`

复制一段 JSON 后，在 Alfred 里输入 `json`。workflow 通过 Alfred 的 `{clipboard}` 读取剪贴板，在程序里格式化，再用编辑器打开。内容为空或不是合法 JSON 时会弹出警告。

默认优先使用已安装的 VSCodium、Cursor、VS Code、Zed、Sublime Text、GoLand 或 PhpStorm，否则用文本编辑。Workflow 配置里的「编辑器」可填应用名，例如 `GoLand`。

## 命令

```bash
make test
make package
make import
make import WORKFLOW=tt
make import WORKFLOW=json
```

`make import` 会用 Alfred 5 打开打包文件，确认导入即可。已存在相同 bundle id（`com.soul.alfred.tt`）时会提示更新。

## 新增 workflow

1. 新建 `cmd/<name>/main.go`。
2. 添加 `cmd/<name>/info.plist`，脚本里调用 `./<name>`。二进制名和工作目录名一致。
3. 可选：放一张 `cmd/<name>/icon.png`。
4. `make import WORKFLOW=<name>`。
