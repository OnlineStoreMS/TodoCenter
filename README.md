# TodoCenter — OSMS 待办中心

统一记录电商、发货、售后、门店等业务待办事项；支持图片笔记与手机扫码上传。

| 项 | 值 |
|----|-----|
| Go module | `todocenter` |
| API | `:8102` |
| Web | `:5186` |
| Docker 镜像 | `todocenter-api`、`todocenter-web` |
| UserCore app | `todocenter`（`todo:read` / `todo:write`） |
| 存储 | 本地 `./data/uploads` 或平台 MinIO bucket `todocenter` |
| 端口约定 | [deploy/docs/PORTS.md](../deploy/docs/PORTS.md) |
| 平台编排 | `/home/asialeaf/projects/deploy` |

## 本地开发

```bash
cp configs/config.example.yaml configs/config.yaml
# 编辑 postgres_dsn / jwt_secret（与 UserCore 一致）

go run ./cmd/api -config configs/config.yaml

cd web && npm i && npm run dev
```

首次需在 PostgreSQL 创建库/用户（或走平台）：

```bash
cd ~/projects/deploy
./scripts/init-external-db.sh   # 含 todocenter
./scripts/init-external-minio.sh  # 含 todocenter bucket
```

## 部署注意（宝塔 PostgreSQL）

库/用户可用 `psql` 创建，但宝塔默认按「库+用户」写 `pg_hba`。Docker / `192.168.3.41` 访问前，请在宝塔「PostgreSQL → 数据库 → todocenter → 权限/远程」为用户 `todocenter` 放行（与 materialcore 相同做法）。本机开发可先用 `host=127.0.0.1`。

## 功能概览

- 待办 CRUD：分类、状态、优先级、截止时间
- **固定月待办**：设为每月循环后，每月自动生成一条待处理实例（可指定每月几号）
- 图片笔记：本地上传 + 手机扫码上传会话
- 分类管理：可增删改，种子分类电商 / 发货 / 售后 / 门店
