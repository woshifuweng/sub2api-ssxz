# SSXZ 项目 — 每个会话开工前必读

这个文件会被自动读进每个新会话。它只写"会让你判断错的事实"，不写别的。

## 1. 真源码不在 `backend/`

主目录 `backend/` 是空壳（0 个 `.go` 文件）。往那里 grep 会**静默返回空**，读起来像"这功能不存在"——这是本项目最容易上的当。

```
真源码：.codex-work/fix-client-brand-announcements2/
```

## 2. "生产在跑什么"去读 `DEPLOYED.md`

不要靠 grep 源码树推断。仓库里同时存在**多条平行代码线**（差异达 4720 个文件），每条都是"真的代码"，但只有一条是活的。`DEPLOYED.md` 是唯一仲裁者。

一条命令直接问生产，不需要凭证：

```bash
curl -s https://api.ssxzapi.com/api/v1/settings/public | grep -o '"version":"[^"]*"'
```

## 3. 三个会骗你的信号

| 信号 | 为什么不可信 |
|---|---|
| `VERSION` 文件 | 写的是**我们 fork 自己的**发布计数器（0.1.x），跟上游 Wei-Shaw 的计数器（0.1.17x）是两套编号撞在同一个字段里。`0.1.3` 不代表"落后 168 个版本"。 |
| `git merge-base` / `--is-ancestor` | 本仓库大量使用 squash-merge，**祖先关系已被销毁**。"不是祖先"≠"没合过这份代码"。 |
| `accounts.updated_at` | 运行时会写这一行，它不等于"配置改动时间"。 |

## 4. 判断代码血脉的正确方法：文件指纹

tag 祖先关系不可信，但**文件内容不会被 squash 抹掉**。要判断某条线有没有包含上游某版本：

```bash
# 列出上游两个 tag 之间新增的文件，再看它们在目标分支里是否存在
git diff --name-status v0.1.136 v0.1.165 | awk '$1=="A"{print $2}'
git cat-file -e <branch>:<那个文件> && echo 有 || echo 无
```

全部"无" = 那条线不含该版本。这个方法**能区分**「165 底座」和「更早底座」，而 tag 测试不能。

## 5. 生产路由探测（判断线上到底部署了哪条线）

未鉴权 GET 一个 admin 路由：`401` = 路由存在，`404` = 路由不存在。用一条两条线都有的路由当控制组：

```bash
curl -s -o /dev/null -w '%{http_code}\n' https://api.ssxzapi.com/api/v1/admin/users   # 控制组，应为 401
```

## 6. 红线

- 入口 JS **禁止**加 `?v=` query（两个 URL = 两份模块 = 双 mount 黑屏，2026-08-07 黑屏根因）
- 编译生产二进制必须 `GOOS=linux GOARCH=amd64 CGO_ENABLED=0`，漏设立刻 crash 回滚
- 往 `channel_model_pricing` 加行会**直接改扣费**，那张表是计费覆盖层
- 不要手改 `VERSION` 文件，`release.yml` 会从 tag 覆写它
