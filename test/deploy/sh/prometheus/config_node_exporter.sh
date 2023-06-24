# Make it as idempotent as possible, it can be called over and over

# Prometheus node exporter
# https://www.digitalocean.com/community/tutorials/how-to-install-prometheus-on-ubuntu-16-04

PROMETHEUS_NODE_EXPORTER_SERVICE_FILE=/etc/systemd/system/node_exporter.service

sudo rm -f $PROMETHEUS_NODE_EXPORTER_SERVICE_FILE

sudo tee $PROMETHEUS_NODE_EXPORTER_SERVICE_FILE <<EOF
[Unit]
Description=Prometheus Node Exporter
Wants=network-online.target
After=network-online.target
[Service]
User=node_exporter
Group=node_exporter
Type=simple
ExecStart=/usr/local/bin/node_exporter
[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload

sudo systemctl start node_exporter
sudo systemctl status node_exporter
curl http://localhost:9100/metrics > /dev/null
if [ "$?" -ne "0" ]; then
    echo localhost:9100/metrics
    exit $?
fi
