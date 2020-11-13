#!/bin/bash
cp *.service *.timer /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now dc@gopheries.service
systemctl enable --now dc-reload@gopheries.timer
