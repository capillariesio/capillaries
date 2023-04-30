# Default ssh connection limit from one client is 10, increase it
sudo sed -i -e "s~[# ]*MaxStartups[ ]*[0-9:]*~MaxStartups 100~g" /etc/ssh/sshd_config
sudo sed -i -e "s~[# ]*MaxSessions[ ]*[0-9]*~MaxSessions 100~g" /etc/ssh/sshd_config
sudo systemctl reload sshd