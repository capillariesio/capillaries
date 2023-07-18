# Disable it before changing DNS server, otherwise it may start updating
sudo systemctl stop unattended-upgrades

# We are about to remove DNS server 127.0.0.53 that knows this host. Just save it in /etc/hosts
echo 127.0.0.1 $(hostname) | sudo tee -a /etc/hosts

# Replace DNS server, default 127.0.0.53 knows nothing
sudo sed -i "s/nameserver[ ]*[0-9.]*/nameserver 8.8.8.8/" /etc/resolv.conf 

sudo resolvectl flush-caches

sudo apt-get -y update

# Utilities for checking cloud performance, feel free to comment this out
sudo apt-get install -y iperf
sudo apt-get install -y sysbench