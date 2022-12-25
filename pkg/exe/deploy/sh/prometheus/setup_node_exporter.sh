# Prometheus node exporter
# https://www.digitalocean.com/community/tutorials/how-to-install-prometheus-on-ubuntu-16-04

# Expecting
# PROMETHEUS_NODE_EXPORTER_VERSION=1.5.0

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
sudo touch $PROMETHEUS_NODE_EXPORTER_SERVICE_FILE

echo "[Unit]" | sudo tee -a $PROMETHEUS_NODE_EXPORTER_SERVICE_FILE
echo "Description=Prometheus Node Exporter" | sudo tee -a $PROMETHEUS_NODE_EXPORTER_SERVICE_FILE
echo "Wants=network-online.target" | sudo tee -a $PROMETHEUS_NODE_EXPORTER_SERVICE_FILE
echo "After=network-online.target" | sudo tee -a $PROMETHEUS_NODE_EXPORTER_SERVICE_FILE
echo "" | sudo tee -a $PROMETHEUS_NODE_EXPORTER_SERVICE_FILE
echo "[Service]" | sudo tee -a $PROMETHEUS_NODE_EXPORTER_SERVICE_FILE
echo "User=node_exporter" | sudo tee -a $PROMETHEUS_NODE_EXPORTER_SERVICE_FILE
echo "Group=node_exporter" | sudo tee -a $PROMETHEUS_NODE_EXPORTER_SERVICE_FILE
echo "Type=simple" | sudo tee -a $PROMETHEUS_NODE_EXPORTER_SERVICE_FILE
echo "ExecStart=/usr/local/bin/node_exporter" | sudo tee -a $PROMETHEUS_NODE_EXPORTER_SERVICE_FILE
echo "" | sudo tee -a $PROMETHEUS_NODE_EXPORTER_SERVICE_FILE
echo "[Install]" | sudo tee -a $PROMETHEUS_NODE_EXPORTER_SERVICE_FILE
echo "WantedBy=multi-user.target" | sudo tee -a $PROMETHEUS_NODE_EXPORTER_SERVICE_FILE

sudo systemctl daemon-reload

sudo systemctl start node_exporter
sudo systemctl status node_exporter
curl http://localhost:9100/metrics > /dev/null
if [ "$?" -ne "0" ]; then
    echo localhost:9100/metrics
    exit $?
fi
