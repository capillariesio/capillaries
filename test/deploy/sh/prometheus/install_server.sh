if [ "$PROMETHEUS_VERSION" = "" ]; then
  echo Error, missing: PROMETHEUS_VERSION=2.41.0
 exit 1
fi

# Create users
sudo useradd --no-create-home --shell /bin/false prometheus

# Before we download the Prometheus binaries, create the necessary directories for storing Prometheus’ files and data. Following standard Linux conventions, we’ll create a directory in /etc for Prometheus’ configuration files and a directory in /var/lib for its data.
sudo mkdir /etc/prometheus
sudo mkdir /var/lib/prometheus

# Now, set the user and group ownership on the new directories to the prometheus user.
sudo chown prometheus:prometheus /etc/prometheus
sudo chown prometheus:prometheus /var/lib/prometheus

if [ "$(uname -p)" == "x86_64" ]; then
ARCH=amd64
else
ARCH=arm64
fi

# Downloading Prometheus
PROMETHEUS_DL_FILE=prometheus-$PROMETHEUS_VERSION.linux-$ARCH
cd ~
sudo rm -f $PROMETHEUS_DL_FILE.gz
echo Downloading https://github.com/prometheus/prometheus/releases/download/v$PROMETHEUS_VERSION/$PROMETHEUS_DL_FILE.tar.gz
curl -LO https://github.com/prometheus/prometheus/releases/download/v$PROMETHEUS_VERSION/$PROMETHEUS_DL_FILE.tar.gz
if [ "$?" -ne "0" ]; then
    echo Cannot download, exiting
    exit $?
fi
tar xvf $PROMETHEUS_DL_FILE.tar.gz

# Copy the two binaries to the /usr/local/bin directory.

sudo cp $PROMETHEUS_DL_FILE/prometheus /usr/local/bin/
sudo cp $PROMETHEUS_DL_FILE/promtool /usr/local/bin/

# Set the user and group ownership on the binaries to the prometheus user created in Step 1.
sudo chown prometheus:prometheus /usr/local/bin/prometheus
sudo chown prometheus:prometheus /usr/local/bin/promtool

# Copy the consoles and console_libraries directories to /etc/prometheus.
sudo cp -r $PROMETHEUS_DL_FILE/consoles /etc/prometheus
sudo cp -r $PROMETHEUS_DL_FILE/console_libraries /etc/prometheus

# Set the user and group ownership on the directories to the prometheus user. Using the -R flag will ensure that ownership is set on the files inside the directory as well.
sudo chown -R prometheus:prometheus /etc/prometheus/consoles
sudo chown -R prometheus:prometheus /etc/prometheus/console_libraries

# Lastly, remove the leftover files from your home directory as they are no longer needed.
rm -rf $PROMETHEUS_DL_FILE.tar.gz $PROMETHEUS_DL_FILE

