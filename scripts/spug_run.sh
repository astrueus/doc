#!/bin/bash

# uploads目录
if [ ! -e /data/repos/doc.itopcms.com/resource/uploads ]; then
  mkdir -p /data/repos/doc.itopcms.com/resource/uploads
  #chown -R www:www /data/repos/doc.itopcms.com/resource/uploads
fi
ln -s /data/repos/doc.itopcms.com/resource/uploads /data/wwwroot/doc.itopcms.com/uploads

# 检查运行时目录
if [ ! -e /data/repos/doc.itopcms.com/resource/runtime ]; then
  mkdir -p /data/repos/doc.itopcms.com/resource/runtime
  #chown -R www:www /data/repos/doc.itopcms.com/resource/runtime
fi
ln -s /data/repos/doc.itopcms.com/resource/runtime /data/wwwroot/doc.itopcms.com/runtime

# 检查配置是否存在
if [ ! -e /data/repos/doc.itopcms.com/resource/app.conf ]; then
  mv /data/wwwroot/doc.itopcms.com/conf/app.conf.example /data/repos/doc.itopcms.com/resource/app.conf
  rm -rf /data/wwwroot/doc.itopcms.com/conf/app.conf.example
fi
cp -rf /data/repos/doc.itopcms.com/resource/app.conf /data/wwwroot/doc.itopcms.com/conf/app.conf

# 目录所属用户和组，以及权限
#chown -R www:www /data/wwwroot/doc.itopcms.com/
chmod 744 /data/wwwroot/doc.itopcms.com/doc

# 重启应用
if systemctl list-unit-files --type=service | grep -qw "^doc.service"; then
  SERVICE_PATH=$(systemctl show -p FragmentPath "doc.service" | awk -F= '{print $2}')
  echo $SERVICE_PATH
  LOADED_PATH=$(readlink -f "$SERVICE_PATH")
  echo $LOADED_PATH
  if [ "$LOADED_PATH" == "/data/repos/doc.itopcms.com/resource/scripts/doc.service" ]; then
    systemctl restart doc
  else
    echo "已存在同名[doc.service]服务，请重新修改服务名称，或检查。"
    exit 1
  fi
else
  if [ ! -e /data/repos/doc.itopcms.com/resource/scripts ]; then
    mkdir -p /data/repos/doc.itopcms.com/resource/scripts
    cp -rf /data/wwwroot/doc.itopcms.com/scripts /data/repos/doc.itopcms.com/resource/
  fi
  if [ ! -e /data/repos/doc.itopcms.com/resource/scripts/doc.service ]; then
    echo "[doc.service]服务定义文件不存在，请先创建服务文件提交后，再发布。"
    exit 1
  fi
  ln -sfn "/data/repos/doc.itopcms.com/resource/scripts/doc.service" "/etc/systemd/system/doc.service"
  systemctl daemon-reload
  systemctl start doc
fi