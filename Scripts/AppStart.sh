#!/bin/sh

cd /home/testing_user/apps/goapp_news_demo/news-demo-starter-files || exit

export PATH=$HOME/opt/go/bin:$PATH

go build
./news-demo-starter-files