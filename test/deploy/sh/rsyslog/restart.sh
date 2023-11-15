# On AWS machines, this works when run after dasemon is started. Othereise, no capidaemon.log is sent.
# https://askubuntu.com/questions/318632/rsyslog-not-forwarding-specific-log-file-to-remote-server
# Don't ask me why.
sudo systemctl restart rsyslog