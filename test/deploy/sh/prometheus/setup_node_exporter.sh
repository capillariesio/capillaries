# Prometheus node exporter
# https://www.digitalocean.com/community/tutorials/how-to-install-prometheus-on-ubuntu-16-04

if [ "$PROMETHEUS_NODE_EXPORTER_VERSION" = "" ]; then
  echo Error, missing: PROMETHEUS_NODE_EXPORTER_VERSION=1.5.0
  exit
fi

sudo useradd --no-create-home --shell /bin/false node_exporter

# Download node exporter
EXPORTER_DL_FILE=node_exporter-$PROMETHEUS_NODE_EXPORTER_VERSION.linux-amd64
cd ~
sudo rm -f $EXPORTER_DL_FILE.tar.gz
echo Downloading https://github.com/prometheus/node_exporter/releases/download/v$PROMETHEUS_NODE_EXPORTER_VERSION/$EXPORTER_DL_FILE.tar.gz ...
curl -LO https://github.com/prometheus/node_exporter/releases/download/v$PROMETHEUS_NODE_EXPORTER_VERSION/$EXPORTER_DL_FILE.tar.gz
if [ "$?" -ne "0" ]; then
    echo Cannot download, exiting
    exit $?
fi
tar xvf $EXPORTER_DL_FILE.tar.gz

sudo cp $EXPORTER_DL_FILE/node_exporter /usr/local/bin
sudo chown node_exporter:node_exporter /usr/local/bin/node_exporter

rm -rf $EXPORTER_DL_FILE.tar.gz $EXPORTER_DL_FILE

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
