#!/bin/bash

set -x

echo "start deploy ${USER}"
GOOS=linux GOARCH=amd64 go build -o isuride_linux
for server in isu01; do
  ssh -t $server "sudo systemctl stop isuride-go.service"
  # for build on Linux
  # rsync -vau --exclude=app ./ $server:/home/isucon/private_isu/webapp/golang/
  # ssh -t $server "cd /home/isucon/private_isu/webapp/golang; PATH=/home/isucon/.local/go/bin/:/usr/local/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin make"

  # to remove old log
  # ssh -t $server "sudo truncate -s 0 /var/log/nginx/access.log; sudo truncate -s 0 /var/log/mysql/slow.log"

  scp ./isuride_linux $server:/home/isucon/webapp/go/isuride
  rsync -vau ../sql/ $server:/home/isucon/webapp/sql/
  ssh -t $server "sudo systemctl start isuride-go.service"
done

echo "finish deploy ${USER}"
