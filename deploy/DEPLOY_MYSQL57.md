# MySQL 5.7 部署说明

本项目当前以 MySQL 为正式数据源，业务数据必须入库，不使用内存临时存储承载正式功能。

## 版本要求

- MySQL 5.7.8 或以上版本。系统使用 `JSON` 字段类型，低于 5.7.8 不支持。
- 字符集建议 `utf8mb4`，排序规则建议 `utf8mb4_unicode_ci`。
- SQL 已避免 `CAST(... AS JSON)`、`JSON_OBJECT()`、`utf8mb4_0900_*`、`CHECK`、不可见索引等 MySQL 8 专属写法。

## 新环境安装

空库安装使用 `install/init.sql`：

```bash
mysql -h <host> -u <user> -p -e "CREATE DATABASE reporter CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql -h <host> -u <user> -p reporter < install/init.sql
```

安装后修改 `config.yaml` 的数据库连接：

```yaml
database:
  driver: mysql
  dsn: "<user>:<password>@tcp(<host>:3306)/reporter?charset=utf8mb4&loc=Local&parseTime=true"
```

## 带数据迁移

如果要把当前库完整迁移到正式环境，使用完整 dump：

```bash
mysql -h <host> -u <user> -p -e "CREATE DATABASE reporter CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql -h <host> -u <user> -p reporter < deploy/report_full_20260515.sql
```

注意：完整 dump 包含 `DROP TABLE IF EXISTS`，只应导入新库或已备份的库。

## 迁移脚本

不要直接用 `mysql < migrations/*.sql` 导入原始迁移文件。原始迁移是 goose 格式，包含 `-- +goose Down` 回滚段，直接拼接导入会执行回滚 SQL。

没有 goose 的环境使用已生成的 Up-only 文件：

```bash
mysql -h <host> -u <user> -p reporter < deploy/mysql_migrations_up_20260515.sql
```

如新增迁移，需要重新生成 Up-only 文件：

```bash
awk 'BEGIN{in_down=0} /^-- \+goose Up/{in_down=0; next} /^-- \+goose Down/{in_down=1; next} in_down==0 {print}' migrations/*.sql > deploy/mysql_migrations_up_20260515.sql
```

## 部署前检查

```bash
rg -n "CAST\\(|JSON_OBJECT|utf8mb4_0900|CHECK \\(|INVISIBLE|GENERATED ALWAYS|WITH RECURSIVE|ROW_NUMBER\\(|->>" install/init.sql migrations deploy/mysql_migrations_up_20260515.sql
```

该命令不应输出 MySQL 5.7 不兼容项。
