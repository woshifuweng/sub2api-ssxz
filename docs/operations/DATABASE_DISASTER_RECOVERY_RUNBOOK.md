# SSXZ 数据库灾难恢复 RUNBOOK

适用场景：原业务数据库已不可用，后台“备份管理”也无法登录。目标是先把异地 `.sql.gz` 备份恢复到隔离 PostgreSQL 验证库；**验证通过前禁止覆盖生产库**。

## 1. 从数据库外取回恢复资料

从密码管理器或离线应急包取得以下内容，不要写入仓库、聊天、截图或日志：

- S3/R2 Endpoint、Region、Bucket、Prefix、Access Key ID、Secret Access Key
- 最近成功备份的 Object Key、对象大小、完成时间和 SHA-256（如有）
- 新数据库连接信息
- 应用恢复所需的加密密钥及运行环境密钥

若 Object Key 未记录，可在有权限的终端临时设置 `AWS_ACCESS_KEY_ID`、`AWS_SECRET_ACCESS_KEY`、`AWS_DEFAULT_REGION`，再执行：

```bash
aws --endpoint-url "$S3_ENDPOINT" s3 ls "s3://$S3_BUCKET/$S3_PREFIX" --recursive
```

选取最新且大小大于 0 的 `.sql.gz`，不要只凭文件名判断成功。

## 2. 下载并校验备份对象

```bash
mkdir -p /srv/ssxz-recovery && chmod 700 /srv/ssxz-recovery
cd /srv/ssxz-recovery
aws --endpoint-url "$S3_ENDPOINT" s3 cp "s3://$S3_BUCKET/$S3_OBJECT_KEY" ssxz.sql.gz
test -s ssxz.sql.gz
gzip -t ssxz.sql.gz
sha256sum ssxz.sql.gz
```

记录对象 Key、字节数、SHA-256 和下载时间；记录中不得包含任何凭据。

## 3. 恢复到隔离验证库

验证库必须是新建空库，禁止指向生产数据库：

```bash
export PGHOST=127.0.0.1 PGPORT=5432 PGUSER=postgres
export VERIFY_DB="ssxz_restore_verify_$(date +%Y%m%d_%H%M%S)"
createdb "$VERIFY_DB"
gzip -dc ssxz.sql.gz | psql --set ON_ERROR_STOP=on --single-transaction --dbname "$VERIFY_DB"
```

任何 SQL 错误都视为演练失败，不得继续切换生产流量。

## 4. 验证数据可用

```bash
psql --dbname "$VERIFY_DB" --set ON_ERROR_STOP=on <<'SQL'
SELECT current_database(), now();
SELECT count(*) AS public_tables
FROM pg_tables WHERE schemaname = 'public';
SELECT count(*) AS users FROM users;
SELECT count(*) AS api_keys FROM api_keys;
SELECT count(*) AS settings FROM settings;
SELECT count(*) AS usage_logs FROM usage_logs;
SQL
```

通过标准：恢复命令退出码为 0；核心表存在；管理员/用户、API Key、配置和用量记录可查询；抽查时间范围与备份时间一致。只记录计数和时间，不输出 Key、密码、Token、上游凭据或用户内容。

## 5. 演练收尾与正式恢复边界

演练完成后保存脱敏证据：备份 ID/Object Key 尾部、对象大小、SHA-256、恢复开始/结束时间、PostgreSQL 版本、核心表计数和结论。删除终端环境中的 S3/数据库凭据，并按需要删除验证库：

```bash
unset AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN PGPASSWORD
dropdb "$VERIFY_DB"
```

正式恢复必须另行审批：先停止写流量、再做故障库快照、恢复到新库、执行同样验证，最后切换数据库连接并保留回滚点。本 RUNBOOK 不授权直接覆盖生产库。

## 6. 当前 SSXZ 恢复资产（2026-07-12）

生产数据库之外已有一份 Windows DPAPI 加密恢复包：

```text
C:\Users\24091\Documents\中转站项目\backups\recovery-credentials\SSXZ_R2_BACKUP_CREDENTIALS.dpapi
```

配套检查工具：

```text
C:\Users\24091\Documents\中转站项目\backups\recovery-credentials\Inspect-SSXZ-R2-Credentials.ps1
```

该包包含 R2 取回凭据、桶/前缀和固定应用敏感配置加密密钥，不含明文。它只允许当前 Windows 用户在当前电脑解密，不能视为整机灾备；必须再复制到一个离机加密位置。若包不可用，可登录 Cloudflare 为现有私有桶重新签发仅限单桶、仅 Object Read & Write 的 Token，并撤销旧 Token。

最近一次完成隔离恢复验证的对象：

```text
database/2026/07/12/sub2api_20260712_012027.sql.gz
size: 2744514 bytes
sha256: 94b1340e67240959b8d219c1c025883919fac58798833feb7f671033bc1a9d32
```

恢复后核心计数：85 张 public 表，users=8、api_keys=50、settings=209、usage_logs=1633；四项业务计数与同时刻生产库一致。完整证据见：

```text
C:\Users\24091\Documents\中转站项目\docs\AI协作\P0_2_R2_RESTORE_DRILL.md
```

生产 `/opt/sub2api/config.yaml` 已固定 `totp.encryption_key`，其数据库外副本也在 DPAPI 包内。恢复后必须使用同一密钥启动应用，否则 TOTP、已加密的备份 Secret 和仍使用旧加密格式的敏感配置可能无法解密。数据库完全丢失但服务器仍在时，生产 `config.yaml` 继续提供数据库连接/JWT 等运行配置；若整台服务器同时丢失，还需从独立服务器配置备份恢复这些内容。
