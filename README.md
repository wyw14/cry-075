# 季节素材发布编排与失效控制平台

这是一个完全离线运行的季节素材运营系统。运营人员可以把素材组成专题包、安排推荐位与可见时间，经过审核后发布到前台资源面板；系统同时管理版权失效、紧急下线、自动替补、多环境快照、回滚、复盘和审计导出。

## 模块职责

- `internal/domain`：专题、素材包、时间窗、推荐位、人群、状态机、排期、快照、失效和审计规则。
- `internal/application`：创建与编排、审核发布、预览与影响分析、紧急兜底、版本回滚、运营记录和受控导出。
- `internal/repository`：pgx/PostgreSQL 持久化实现以及可确定验证事务边界的内存实现。
- `internal/service`：本地排程器、版权到期扫描与 CSV 素材导入。
- `internal/transport/httpapi`：Gin `/api/v1` API、稳定错误体和输入校验。
- `internal/middleware`：本地角色、request_id、超时、CORS 和安全响应头。
- `internal/platform`：本地附件、通知、回调和健康检查适配器。
- `web`：Vue 3、TypeScript、Vite、Pinia 中文运营台。

前端包含专题编排、发布时间轴、前台预览、审核中心、失效中心、版本与历史六个工作页。它只访问本地 `/api/v1`，不依赖 CDN、云存储、在线模型或真实消息服务。

## 本地启动

要求 Go 1.24+、Node.js 22+、PostgreSQL 17 和 `psql`。

```bash
cp .env.example .env
export DATABASE_URL='postgres://seasonal:seasonal@localhost:5432/seasonal?sslmode=disable'
make migrate
make seed
go run ./cmd/server
```

另一个终端启动前端：

```bash
npm --prefix web ci
npm --prefix web run dev
```

也可以运行 `docker compose up --build`。容器使用官方多架构基础镜像，在 amd64 与 arm64 上采用相同构建步骤；PostgreSQL 数据和本地附件分别保存在命名卷中。

## 配置

| 变量 | 说明 | 默认值 |
|---|---|---|
| `HTTP_ADDR` | HTTP 监听地址 | `:8080` |
| `DATABASE_URL` | PostgreSQL 连接串 | 必填 |
| `REQUEST_TIMEOUT` | 单请求最大时长 | `5s` |
| `ATTACHMENT_DIR` | 受控附件目录 | `./data/attachments` |
| `MAX_UPLOAD_BYTES` | 单附件大小上限 | `5242880` |
| `ALLOWED_ORIGINS` | 允许的本地前端 Origin，逗号分隔 | `http://localhost:5173` |
| `WEB_DIST_DIR` | 后端托管的前端生产构建目录 | `./web/dist` |

API 使用 `X-Actor-ID` 和 `X-Actor-Role` 表示离线演示身份。可用角色为 `operator`、`reviewer`、`publisher`、`rights_admin`、`auditor`、`scheduler`。这不是面向公网的认证替代品，而是可验证的本地权限适配器。

## 迁移与演示数据

`migrations/001_initial.sql` 创建专题、通用实体、事件、幂等键和表现计数表，并用 PostgreSQL 排斥约束保证同一推荐位和环境的已发布时间窗不重叠。`002_indexes.sql` 创建列表和审计索引。`003_seed.sql` 用 `ON CONFLICT DO NOTHING` 写入首页推荐位和全部访客人群，不会覆盖已有数据。

PowerShell 可执行：

```powershell
$env:DATABASE_URL='postgres://seasonal:seasonal@localhost:5432/seasonal?sslmode=disable'
.\scripts\migrate.ps1
```

## 状态与一致性规则

专题状态为 `draft -> pending_review -> published -> paused/expired`。审核驳回回到草稿；暂停后的专题可以重新提交审核；过期状态不可恢复。发布前同时执行应用层冲突检查和数据库排斥约束。所有更新携带版本号，陈旧版本返回冲突。

创建专题和创建排期都使用幂等键。相同键和相同输入返回第一次创建的资源；相同键对应不同输入会被拒绝。创建、状态转换、紧急下线和回滚在事务中同时写业务对象与审计事件。

紧急下线先确定每个受影响专题的可用兜底包，全部可替换后才原子提交；任何专题没有兜底时整次操作回滚。前台预览只返回时间窗、人群和版权都有效的素材，主素材失效时优先使用条目替代素材，没有活动专题时再使用推荐位兜底包。

快照绑定环境、专题版本、素材包和校验和。回滚只能发生在相同环境中，并生成序号更高的新发布版本，不改写历史版本。

## 接口示例

创建专题：

```bash
curl -X POST http://localhost:8080/api/v1/campaigns \
  -H 'Content-Type: application/json' \
  -H 'X-Actor-ID: demo_operator' \
  -H 'X-Actor-Role: operator' \
  -H 'Idempotency-Key: summer-2026-create' \
  -d '{"name":"处暑清凉季","season":"夏末","environment":"production","starts_at":"2026-08-22T08:00:00Z","ends_at":"2026-08-29T08:00:00Z"}'
```

错误响应固定包含业务码、可读消息、字段错误和 request_id：

```json
{
  "code": "VALIDATION_FAILED",
  "message": "请求字段不合法",
  "field_errors": [{"field": "Idempotency-Key", "reason": "required"}],
  "request_id": "7a0e0b8d76dd7f281129b002"
}
```

完整接口见 `api/openapi/openapi.yaml`。

## 测试

```bash
go test ./...
go test -race ./...
go vet ./...
npm --prefix web test
npm --prefix web run build
```

仓储集成测试在提供 `DATABASE_URL` 时连接 PostgreSQL；领域和 HTTP 测试默认使用具备事务回滚语义的本地仓储。当前基线已实际通过普通 Go 测试、race 测试、vet、前端单测和生产构建。依赖安装后 `npm audit` 报告 0 个漏洞。

## 安全与运行数据

附件名称经过归一化，类型和大小使用白名单，内容以 SHA-256 键保存。审计元数据对 token、secret 和 password 字段脱敏，CSV 导出只允许显式白名单字段且要求审计角色和导出原因。日志为 zap 结构化日志；Gin 提供 panic 恢复，服务支持 readiness 关闭和十秒优雅停机。

不要提交 `.env`、密钥、`data`、`node_modules`、`dist`、日志或数据库卷。开发和测试期间产生的文件均在 `.gitignore` 中排除。
