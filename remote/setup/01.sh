#!/bin/bash
set -eu

# Server timezone
TIMEZONE=Africa/Addis_Ababa

# Project settings
APP_NAME=greenlight
APP_USER=$APP_NAME
DB_USER=$APP_NAME
DB_NAME=$APP_NAME
DB_ENV_FILE="/etc/environment"

# Use DB_PASSWORD from env if provided; otherwise reuse the existing one in /etc/environment.
if [ -z "${DB_PASSWORD:-}" ] && [ -f "$DB_ENV_FILE" ]; then
    DB_PASSWORD=$(grep -E '^DB_PASSWORD=' "$DB_ENV_FILE" | head -n1 | cut -d= -f2- || true)
fi

# Prompt only when DB_PASSWORD is still empty.
if [ -z "${DB_PASSWORD:-}" ]; then
    read -s -p "Enter password for ${DB_USER} DB user: " DB_PASSWORD
    echo
fi

export LC_ALL=en_US.UTF-8

# ==================================================================================== #
# SCRIPT LOGIC
# ==================================================================================== #

# Enable Ubuntu universe repository
add-apt-repository --yes universe

# Refresh apt package index
apt update

# Set timezone and install locales
timedatectl set-timezone "$TIMEZONE"
apt --yes install locales-all

# Create app user if it does not already exist
if id -u "$APP_USER" >/dev/null 2>&1; then
    echo "User '$APP_USER' already exists, skipping creation."
else
    useradd --create-home --shell /bin/bash --groups sudo "$APP_USER"

    # Set login password for the new user.
    passwd "$APP_USER"
fi

# Configure UFW (SSH + HTTP + HTTPS)
ufw allow 22
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable

# Install fail2ban
apt --yes install fail2ban

# Install golang-migrate CLI
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.14.1/migrate.linux-amd64.tar.gz | tar xvz
mv migrate.linux-amd64 /usr/local/bin/migrate

# Install Docker Engine and Compose plugin
apt --yes install ca-certificates curl gnupg
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg

echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo $VERSION_CODENAME) stable" > /etc/apt/sources.list.d/docker.list
apt update
apt --yes install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
systemctl enable --now docker

# Ensure docker group exists
getent group docker >/dev/null || groupadd docker

# Allow app user to run Docker commands
usermod -aG docker "$APP_USER"

# Update app DB variables in /etc/environment without duplicating entries.
tmp_env_file=$(mktemp)
if [ -f "$DB_ENV_FILE" ]; then
    grep -Ev '^(DB_USER|DB_NAME|DB_PASSWORD|GREENLIGHT_DB_DSN)=' "$DB_ENV_FILE" > "$tmp_env_file" || true
fi
cat >> "$tmp_env_file" <<EOF
DB_USER=${DB_USER}
DB_NAME=${DB_NAME}
DB_PASSWORD=${DB_PASSWORD}
GREENLIGHT_DB_DSN=postgres://${DB_USER}:${DB_PASSWORD}@localhost:5432/${DB_NAME}?sslmode=disable
EOF
install -m 0644 "$tmp_env_file" "$DB_ENV_FILE"
rm -f "$tmp_env_file"

# Install Caddy from the official apt repository.
apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
apt update
apt --yes install caddy

# Upgrade installed packages.
# --force-confnew replaces local config files with package maintainer versions.
apt --yes -o Dpkg::Options::="--force-confnew" upgrade
echo "Script complete! Rebooting..."

reboot
