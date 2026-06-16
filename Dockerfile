#docker 17.05+版本支持
#sudo docker build -t  gopub .
#sudo docker run --name gopub -p 8192:8192  --restart always  -d   gopub:latest 
FROM golang:1.25-alpine AS golang
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories && \
    apk update && \
    apk add  bash  && \ 
    rm -rf /var/cache/apk/*   /tmp/*     
ADD src/ /data/gopub/src/
ADD go.mod go.sum /data/gopub/
ADD control /data/gopub/control
WORKDIR /data/gopub/
RUN ./control build

FROM node:22-alpine AS node
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk update --no-cache && \
    rm -rf /var/cache/apk/*   /tmp/* 
ADD frontend/package.json frontend/package-lock.json /data/gopub/frontend/
WORKDIR /data/gopub/frontend
RUN npm ci
ADD frontend/ /data/gopub/frontend/
ADD src/views /data/gopub/src/views
ADD src/static /data/gopub/src/static
RUN npm run build

FROM alpine:3.22
MAINTAINER Linc "13579443@qq.com"
ENV TZ='Asia/Shanghai' 
RUN TERM=linux && export TERM
USER root 
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories && \
    apk update && \
    apk add ca-certificates bash tzdata sudo curl wget openssh git && \ 
    echo "Asia/Shanghai" > /etc/timezone && \
    cp -r -f /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    rm -rf /var/cache/apk/*   /tmp/*  && \ 
    mkdir -p /data/htdocs && \
    mkdir -p /data/logs && \
    ssh-keygen -q -N "" -f /root/.ssh/id_rsa && \
    #输出的key需要加入发布目标机的 ~/.ssh/authorized_keys
    cat ~/.ssh/id_rsa.pub  
WORKDIR /data/gopub
ADD control /data/gopub/control
COPY --from=golang /data/gopub/src/gopub /data/gopub/src/gopub
COPY --from=golang /data/gopub/src/conf /data/gopub/src/conf
COPY --from=golang /data/gopub/src/logs /data/gopub/src/logs
COPY --from=golang /data/gopub/src/agent /data/gopub/src/agent
COPY --from=node /data/gopub/src/views /data/gopub/src/views
COPY --from=node /data/gopub/src/static /data/gopub/src/static
CMD ["./control","rundocker"]
