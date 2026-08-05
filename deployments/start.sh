#!/bin/bash
set -eux

# 数据库等初始化
/doc/doc install

# 导出同步检查
mkdir -p /doc-sync-host
if ! [ -f "/doc-sync-host/sync.sh" ]; then
    # 同步方向: docker->HOST 或 HOST -> docker
    # echo "export DOC_SYNC=" >> /doc-sync-host/sync.sh # 不同步
    echo "export DOC_SYNC=docker2host" >> /doc-sync-host/sync.sh # 默认 docker->HOST
    
    # 同步内容（Round 2 收尾：配置目录为 conf/；static/views 位于 web/ 下）
    # conf: 配置
    # database: sqlite方式数据库
    # runtime: 运行时数据(日志等)
    # web: 静态与模板（含 static/ views/）
    # uploads: 上传文件
    # echo "export SYNC_LIST='conf;database;runtime;web;uploads'" >> /doc-sync-host/sync.sh
    # echo "export SYNC_LIST=" >> /doc-sync-host/sync.sh
    echo "export SYNC_LIST='conf;database;uploads'" >> /doc-sync-host/sync.sh

    # 同步操作(sync/copy/sync --dry-run 等，具体参考rclone文档，host2docker务必谨慎操作)
    # echo "export SYNC_ACTION=sync --dry-run" >> /doc-sync-host/sync.sh
    echo "export SYNC_ACTION=sync" >> /doc-sync-host/sync.sh
    
    # 同步脚本
    echo "source /doc/sync_host.sh" >> /doc-sync-host/sync.sh
fi
# 同步操作
source /doc-sync-host/sync.sh

# 运行（默认子命令为 web）
/doc/doc
