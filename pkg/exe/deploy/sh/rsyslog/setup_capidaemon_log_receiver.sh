
sudo tee /etc/rsyslog.d/capidaemon.conf <<EOF
module(load="imudp")
ruleset(name="capidaemon"){action(type="omfile" dirOwner="ubuntu" fileOwner="ubuntu" DirCreateMode="0777" FileCreateMode="0644" dynaFile="/var/log/capidaemon/capidaemon.log")}
input(type="imudp" port="514" ruleset="capidaemon")
EOF

sudo systemctl restart rsyslog
