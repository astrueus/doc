#!/bin/bash
set -eux

# 数据库等初始化
/doc/doc_linux_amd64 install

# 导出同步检查
mkdir -p /doc-sync-host
if ! [ -f "/doc-sync-host/sync.sh" ]; then
    # 同步方向: docker->HOST 或 HOST -> docker
    # echo "export MINDOC_SYNC=" >> /doc-sync-host/sync.sh # 不同步
    echo "export MINDOC_SYNC=docker2host" >> /doc-sync-host/sync.sh # 默认 docker->HOST
    
    # 同步内容
    # configs: 配置
    # database: sqlite方式数据库
    # runtime: 运行时数据(日志等)
    # static: 静态文件
    # uploads: 上传文件
    # views: 页面视图
    # echo "export SYNC_LIST='configs;database;runtime;static;uploads;views'" >> /doc-sync-host/sync.sh # 同步所有内容
    # echo "export SYNC_LIST=" >> /doc-sync-host/sync.sh # 不同步任何内容
    echo "export SYNC_LIST='configs;database;uploads'" >> /doc-sync-host/sync.sh # 同步configs、database、uploads

    # 同步操作(sync/copy/sync --dry-run 等，具体参考rclone文档，host2docker务必谨慎操作)
    # echo "export SYNC_ACTION=sync --dry-run" >> /doc-sync-host/sync.sh # 无操作且仅显示同步文件信息(--dry-run)
    echo "export SYNC_ACTION=sync" >> /doc-sync-host/sync.sh # 默认同步
    
    # 同步脚本
    echo "source /doc/sync_host.sh" >> /doc-sync-host/sync.sh
fi
# 同步操作
source /doc-sync-host/sync.sh

# 运行
/doc/doc_linux_amd64
