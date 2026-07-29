FROM --platform=linux/amd64 golang:1.25.0 AS build

ARG TAG=0.0.1

# 编译-环境变量
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct
ENV CGO_ENABLED=1
ENV GOARCH=amd64
ENV GOOS=linux

WORKDIR /app
COPY . .

# 编译
RUN go env
RUN go mod tidy -v
RUN go build -o doc -ldflags "-w -s -X 'git.itopcms.com/jackliu/doc/internal/config.VERSION=$TAG' -X 'git.itopcms.com/jackliu/doc/internal/config.BUILD_TIME=`date`' -X 'git.itopcms.com/jackliu/doc/internal/config.GO_VERSION=`go version`'" ./cmd/doc
RUN cp configs/app.conf.example configs/app.conf
# 清理源码与构建脚手架；保留运行所需：configs/ web/ uploads/ mail/ graphics/ doc start.sh sync_host.sh
RUN rm -f appveyor.yml .travis.yml .gitattributes .gitignore go.mod go.sum README.md
RUN rm -rf cache cmd internal pkg converter .git .github scripts docs
# 将部署脚本提升到镜像工作目录根，便于 ENTRYPOINT 与 sync 引用
RUN cp deployments/start.sh /app/start.sh \
 && cp deployments/sync_host.sh /app/sync_host.sh \
 && rm -rf deployments

# 测试编译的 doc 是否 ok
RUN ./doc version

# 必要的文件复制
ADD web/static/fonts/simsun.ttc /usr/share/fonts/win/


# Ubuntu 20.04
FROM ubuntu:focal

# 切换默认shell为bash
SHELL ["/bin/bash", "-c"]

COPY --from=build /usr/share/fonts/win/simsun.ttc /usr/share/fonts/win/
COPY --from=build /app /doc
WORKDIR /doc
RUN chmod a+r /usr/share/fonts/win/simsun.ttc

# 备份原有源
RUN mv /etc/apt/sources.list /etc/apt/sources.list-backup
# 最小化源，缩短apt update时间(ca-certificates必须先安装才支持换tsinghua源)
RUN echo 'deb http://archive.ubuntu.com/ubuntu/ focal main restricted' > /etc/apt/sources.list
RUN apt-get update
RUN apt install -y ca-certificates
# 更换tsinghua源(echo多行内容不能以#开头，会被docker误判为注释行，所以采用\n#开头)
RUN echo $'\
\n# from: https://mirrors.tuna.tsinghua.edu.cn/help/ubuntu/\n\
deb https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ focal main restricted universe multiverse\
\n# deb-src https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ focal main restricted universe multiverse\n\
deb https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ focal-updates main restricted universe multiverse\
\n# deb-src https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ focal-updates main restricted universe multiverse\n\
deb https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ focal-backports main restricted universe multiverse\
\n# deb-src https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ focal-backports main restricted universe multiverse\n\
deb https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ focal-security main restricted universe multiverse\
\n# deb-src https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ focal-security main restricted universe multiverse\
\n# deb https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ focal-proposed main restricted universe multiverse\
\n# deb-src https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ focal-proposed main restricted universe multiverse'\
> /etc/apt/sources.list

# 更新软件包信息
RUN apt-get update
# 安装必要的系统工具
RUN apt install -y apt-transport-https ca-certificates curl wget xz-utils

# 时区设置(如果不设置, calibre依赖的tzdata在安装过程中会要求选择时区)
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
# tzdata的前端类型默认为readline（Shell情况下）或dialog（支持GUI的情况下）
ARG DEBIAN_FRONTEND=noninteractive
# 安装时区信息
RUN apt install -y --no-install-recommends tzdata
# 重新配置tzdata软件包，使得时区设置生效
RUN dpkg-reconfigure --frontend noninteractive tzdata

# 安装 calibre 依赖的包
RUN apt install -y libgl-dev libnss3-dev libxcomposite-dev libxrandr-dev libxi-dev
# 安装文泉驿字体
RUN apt install -y fonts-wqy-microhei fonts-wqy-zenhei
# 安装中文语言包
RUN apt-get install -y locales language-pack-zh-hans language-pack-zh-hans-base
# 设置默认编码
RUN locale-gen "zh_CN.UTF-8"
RUN update-locale LANG=zh_CN.UTF-8
ENV LANG=zh_CN.UTF-8
ENV LANGUAGE=zh_CN:en
ENV LC_ALL=zh_CN.UTF-8
# 安装-calibre
# RUN apt-get install -y calibre # 此种方式安装省事，但会安装很多额外不需要的软件包，导致体积过大
RUN mkdir -p /tmp/calibre-cache
# 获取最新版本号
RUN curl -s http://code.calibre-ebook.com/latest>/tmp/calibre-cache/version
# 下载最新版本
# RUN wget -O /tmp/calibre-cache/calibre-x86_64.txz -c https://download.calibre-ebook.com/`cat /tmp/calibre-cache/version`/calibre-`cat /tmp/calibre-cache/version`-x86_64.txz
RUN wget -O /tmp/calibre-cache/calibre-x86_64.txz -c https://github.com/kovidgoyal/calibre/releases/download/v`cat /tmp/calibre-cache/version`/calibre-`cat /tmp/calibre-cache/version`-x86_64.txz
# 注: 调试阶段，下载alibre-5.22.1-x86_64.txz到本地(使用 python -m http.server)，加速构建
# RUN wget -O /tmp/calibre-cache/calibre-x86_64.txz -c http://10.96.8.252:8000/calibre-5.22.1-x86_64.txz
# 解压
RUN mkdir -p /opt/calibre
# RUN tar --extract --file=/tmp/calibre-cache/calibre-x86_64.txz --directory /opt/calibre
RUN tar xJof /tmp/calibre-cache/calibre-x86_64.txz -C /opt/calibre
ENV PATH=$PATH:/opt/calibre
# 设置calibre相关环境变量
ENV QTWEBENGINE_CHROMIUM_FLAGS="--no-sandbox"
ENV QT_QPA_PLATFORM='offscreen'
# 测试 calibre 可正常使用
RUN ebook-convert --version
# 清理calibre缓存
RUN rm -rf /tmp/calibre-cache

# refer: https://docs.docker.com/engine/reference/builder/#volume
# 数据同步目录
VOLUME /doc-sync-host

# refer: https://docs.docker.com/engine/reference/builder/#expose
EXPOSE 8181/tcp

# 如果配置文件不存在就复制
RUN cp --no-clobber /doc/configs/app.conf.example /doc/configs/app.conf

RUN chmod +x /doc/start.sh /doc/sync_host.sh /doc/doc

ENTRYPOINT ["/bin/bash", "/doc/start.sh"]

# https://docs.docker.com/engine/reference/commandline/build/#options
# docker build -f deployments/Dockerfile --progress plain --rm --build-arg TAG=0.0.1 --tag doc:latest .
# https://docs.docker.com/engine/reference/commandline/run/#options
# set DOC=//d/doc # windows
# export DOC=/home/ubuntu/doc-docker # linux
# docker run --rm -it -p 8181:8181 -v "%DOC%":"/doc-sync-host" --name doc -d doc:latest
# 或: docker compose -f deployments/docker-compose.yml up -d
