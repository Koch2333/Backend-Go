# _template

复制本文件夹为 `<你的模块名>/`（与后端 `internal/<name>` 同名）。然后：

1. `core.ts`：改 `name` 与 `apiPrefix`。
2. `module.ts`：改 `title` / `description` / `nav` / 路由指向的页面。
3. `api.ts`：加你需要的接口函数。
4. `views/`：额外的页面。

Shell 会自动发现 `module.ts`、拼路由、入侧边栏。

详见 `web/CONTRIBUTING-MODULE.md`。
