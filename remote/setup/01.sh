#!/bin/bash
set -eu

# set timezone
TIMEZONE=Africa/Addis_Ababa

# set new user
USERNAME=greenlight

read -p "Enter password for greenlight DB user: " DB_PASSWORD

export LC_ALL=en_US.UTF-8

# ==================================================================================== #
# SCRIPT LOGIC
# ==================================================================================== #

# enable the universe repository
add-apt-repository --yes universe

# update all software packages
apt update

# Set the system timezone and insall all locales
timedatectl set-timezone "$TIMEZONE"
apt --yes install locales-all

# add the new user
useradd --create-home --shell /bin/bash --groups sudo "$USERNAME"

# force a password to be set for the new user the first time they log in
passwd "$USERNAME"

# configure the firewall
ufw allow 22
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable

# install fail2ban
apt --yes install fail2ban

# install the migrate CLI tool
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.14.1/migrate.linux-amd64.tar.gz | tar xvz
mv migrate.linux-amd64 /usr/local/bin/migrate

# Install Docker Engine + the Docker Compose plugin.
apt --yes install ca-certificates curl gnupg
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg

echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo $VERSION_CODENAME) stable" > /etc/apt/sources.list.d/docker.list
apt update
apt --yes install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
systemctl enable --now docker

# Ensure group exists
getent group docker >/dev/null || groupadd docker

# Add user to docker group
usermod -aG docker "$USERNAME"

# Note: appending is simple but not idempotent; re-running the script will duplicate lines.
echo "DB_USER=greenlight" >> /etc/environment
echo "DB_NAME=greenlight" >> /etc/environment
echo "DB_PASSWORD=${DB_PASSWORD}" >> /etc/environment
echo "GREENLIGHT_DB_DSN=postgres://greenlight:${DB_PASSWORD}@localhost:5432/greenlight?sslmode=disable" >> /etc/environment

# Install Caddy (see https://caddyserver.com/docs/install#debian-ubuntu-raspbian).
apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
apt update
apt --yes install caddy

# Upgrade all packages. Using the --force-confnew flag means that configuration
# files will be replaced if newer ones are available.
apt --yes -o Dpkg::Options::="--force-confnew" upgrade
echo "Script complete! Rebooting..."

reboot
