#!/bin/bash -e
#
# example script to install tt running a broker
cp ttsrv.service /lib/systemd/system/
chmod 0664 /lib/systemd/system/ttsrv.service
systemctl daemon-reload
systemctl enable ttsrv.service
systemctl start ttsrv
systemctl status ttsrv
