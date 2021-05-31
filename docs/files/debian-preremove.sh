systemctl stop go-ssb-room
systemctl disable go-ssb-room
# TODO: we might want to have a proper config file so users dont need to tweak this file, then we can also remove and upgrade it properly
# rm /etc/systemd/system/go-ssb-room.service
# systemctl daemon-reload
systemctl reset-failed
