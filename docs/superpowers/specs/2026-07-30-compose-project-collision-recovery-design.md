# Compose 项目名迁移冲突自愈设计

## 背景

历史本地部署脚本未显式传入 Compose 项目名，因此 Docker Compose 使用部署目录名
`local_full` 作为项目名。新版脚本使用 `docker compose -p <stable-name>`，避免不同
项目的 `local_full` 容器互相干扰。

升级后的第一次全量部署会遇到一个迁移问题：旧容器仍占用固定
`container_name`，但因为 Compose 项目标签不同，新 Compose 项目不会接管旧容器，
最终以同名冲突退出。聚合部署脚本启用了 `set -e`，所以一个子项目失败后，后续前端
不会执行。

## 目标

- 点击全量部署或随 VibeDeploy 启动时，自动迁移当前路线遗留的旧 Compose 容器。
- 只删除能够证明由当前 `docker-compose.yml` 创建的旧容器。
- 保持稳定的显式 Compose 项目名，不恢复共享的 `local_full` 项目名。
- 兼容启动和停止脚本，并保证重复物化、重复部署是幂等的。
- 将规范化后的脚本回写 `tb_project_script`，让数据库继续作为部署脚本事实来源。

## 非目标

- 不按容器名直接删除容器。
- 不删除标签或配置路径无法验证的同名容器。
- 不并行化聚合部署步骤。
- 不改变现有 Compose 网络规范化规则。

## 方案

扩展本地部署脚本物化流程。在处理 `start.sh` 或 `stop.sh` 时，仅当脚本同时满足以下
条件才注入兼容逻辑：

1. 包含 `docker compose` 命令；
2. 使用显式 `-p` Compose 项目名；
3. 引用了 `$SCRIPT_DIR/docker-compose.yml`；
4. 尚未包含 VibeDeploy 迁移标记。

注入的 shell 函数通过以下两个 Docker 标签查找历史容器：

- `com.docker.compose.project=$(basename "$SCRIPT_DIR")`
- `com.docker.compose.project.config_files=$SCRIPT_DIR/docker-compose.yml`

这两个条件分别证明容器属于旧的目录派生项目，并且由当前路线的 Compose 配置文件
创建。匹配到的容器在执行新版 Compose 命令前通过 `docker rm -f` 移除。无匹配时不
执行删除；无关同名容器仍由 Docker 返回清晰的冲突错误。

规范化使用固定注释标记保证幂等。现有物化事务会在发布文件前锁定数据库记录，并将
规范化内容原子回写 `tb_project_script`；因此手动全量部署和启动联动走同一条修复
路径。

## 数据流

1. 用户点击全量部署，或后端启动联动选择聚合全量路线。
2. VibeDeploy 加载聚合项目及子项目的数据库脚本。
3. 物化器规范化符合条件的 `start.sh`/`stop.sh`。
4. 数据库脚本和本地文件在同一事务边界内更新。
5. `start.sh` 删除当前配置文件对应的旧 `local_full` 容器。
6. 新的显式 Compose 项目执行 `up --build -d`。
7. 聚合脚本继续部署后续项目。

## 错误处理

- Docker 不可用：现有部署流程直接失败，不吞掉错误。
- 旧容器删除失败：`set -e` 终止部署并保留 Docker 原始错误。
- 同名容器标签不匹配：不删除，由 Docker 报冲突，避免误删。
- 数据库脚本并发修改：沿用现有乐观锁错误并回滚文件发布。
- 脚本不符合识别条件：保持原文，不做猜测性修改。

## 测试

- 符合条件的启动脚本被注入兼容逻辑。
- 符合条件的停止脚本被注入兼容逻辑。
- 注入内容同时包含旧项目标签和当前配置路径标签过滤。
- 重复规范化不产生第二份兼容逻辑。
- 未使用显式 `-p` 或未引用标准 Compose 文件的脚本保持不变。
- 物化后规范化内容回写数据库。
- 运行 `go test ./service/system` 验证现有部署、共享网络和物化测试无回归。

## 验收

清理当前历史冲突后重新执行英语抢词“全部项目全量部署”，确认以下端口可访问：

- Vue 主站：6011
- Java 后端：6012
- 完形填空前端：6014
- 管理后台后端：6015
- 管理后台前端：6016
- Python Agent：6017

再次执行全量部署，不再出现 `container name is already in use`。
