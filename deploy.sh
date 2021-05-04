#!/bin/bash

pm2 stop tiebarankgo
mv /home/ubuntu/application/app/tiebarankgo /home/ubuntu/application/app/tiebarank
pm2 start tiebarankgo