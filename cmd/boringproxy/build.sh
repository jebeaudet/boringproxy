go build
sudo systemctl stop boringproxy.service
sudo setcap cap_net_bind_service=+ep boringproxy
sudo systemctl start boringproxy.service
