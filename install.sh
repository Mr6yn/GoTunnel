#!/bin/bash
# GoTunnel - Interactive Installer

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

INSTALL_DIR="/opt/gotunnel"

print_banner() {
    echo -e "${CYAN}"
    echo "  ____      _____                       _ "
    echo " / ___| ___|_   _|   _ _ __  _ __   ___| |"
    echo "| |  _ / _ \ | || | | | '_ \| '_ \ / _ \ |"
    echo "| |_| | (_) || || |_| | | | | | | |  __/ |"
    echo " \____|\___/ |_| \__,_|_| |_|_| |_|\___|_|"
    echo -e "${NC}"
    echo -e "${BOLD}        High Performance TCP Tunnel v1.0.0${NC}"
    echo ""
}

step() {
    echo -e "\n${BLUE}┌─────────────────────────────────────────┐${NC}"
    echo -e "${BLUE}│${NC} ${BOLD}$1${NC}"
    echo -e "${BLUE}└─────────────────────────────────────────┘${NC}"
}

ok() { echo -e "${GREEN}  ✓ $1${NC}"; }
info() { echo -e "${YELLOW}  → $1${NC}"; }
err() { echo -e "${RED}  ✗ $1${NC}"; exit 1; }

ask() {
    echo -ne "${CYAN}  ▶ $1: ${NC}"
    read -r REPLY
    echo "$REPLY"
}

# ─── Check root ───────────────────────────────────────────────
if [ "$EUID" -ne 0 ]; then
    err "لطفاً با دسترسی root اجرا کن: sudo bash install.sh"
fi

# ─── Check OS ─────────────────────────────────────────────────
if ! [ -f /etc/debian_version ]; then
    err "فقط Ubuntu/Debian پشتیبانی میشه"
fi

print_banner

# ══════════════════════════════════════════════════════════════
# STEP 1 — Server type
# ══════════════════════════════════════════════════════════════
step "مرحله ۱ از ۴ — نوع سرور"
echo ""
echo -e "  ${BOLD}این سرور کدام طرف تانل است؟${NC}"
echo ""
echo -e "  ${YELLOW}1)${NC} سرور ایران   (کاربران به این وصل میشن)"
echo -e "  ${YELLOW}2)${NC} سرور خارج   (پنل و xray/v2ray اینجاست)"
echo ""
echo -ne "${CYAN}  ▶ انتخاب [1/2]: ${NC}"
read -r SERVER_TYPE

case "$SERVER_TYPE" in
    1) MODE="server"; ok "سرور ایران انتخاب شد" ;;
    2) MODE="client"; ok "سرور خارج انتخاب شد" ;;
    *) err "انتخاب نامعتبر. فقط 1 یا 2 وارد کن" ;;
esac

# ══════════════════════════════════════════════════════════════
# STEP 2 — IP and Port
# ══════════════════════════════════════════════════════════════
step "مرحله ۲ از ۴ — آدرس و پورت"
echo ""

if [ "$MODE" == "server" ]; then
    # Iran server needs:
    # - control port (foreign client connects here)
    # - user ports (users connect here)

    echo -e "  ${BOLD}پورت کنترل تانل:${NC}"
    info "سرور خارج از این پورت به سرور ایران وصل میشه"
    info "پیشنهاد: 8443 (یا هر پورت دلخواه)"
    echo -ne "${CYAN}  ▶ پورت کنترل [پیشفرض 8443]: ${NC}"
    read -r CTRL_PORT
    CTRL_PORT=${CTRL_PORT:-8443}
    ok "پورت کنترل: $CTRL_PORT"

    echo ""
    echo -e "  ${BOLD}پورت(های) کاربری:${NC}"
    info "کاربران از این پورت‌ها به تانل وصل میشن"
    info "میتونی چند پورت وارد کنی — با ویرگول جدا کن"
    info "مثال: 443,80,8080"
    echo -ne "${CYAN}  ▶ پورت(های) کاربری: ${NC}"
    read -r USER_PORTS_RAW
    USER_PORTS_RAW=${USER_PORTS_RAW:-443}

    # Convert to JSON array
    PORTS_JSON="[$(echo "$USER_PORTS_RAW" | sed 's/ //g' | sed 's/,/, /g')]"
    ok "پورت‌های کاربری: $USER_PORTS_RAW"

else
    # Foreign client needs:
    # - Iran server IP
    # - Iran control port
    # - Local xray/v2ray port

    echo -e "  ${BOLD}آدرس IP سرور ایران:${NC}"
    info "آی‌پی سروری که طرف ایران نصب کردی"
    echo -ne "${CYAN}  ▶ IP سرور ایران: ${NC}"
    read -r IRAN_IP
    [ -z "$IRAN_IP" ] && err "IP نمیتونه خالی باشه"
    ok "IP ایران: $IRAN_IP"

    echo ""
    echo -e "  ${BOLD}پورت کنترل سرور ایران:${NC}"
    info "همون پورتی که روی سرور ایران وارد کردی"
    echo -ne "${CYAN}  ▶ پورت کنترل [پیشفرض 8443]: ${NC}"
    read -r CTRL_PORT
    CTRL_PORT=${CTRL_PORT:-8443}
    ok "پورت کنترل: $CTRL_PORT"

    echo ""
    echo -e "  ${BOLD}پورت سرویس محلی (xray/v2ray/پنل):${NC}"
    info "پورتی که پنل یا xray روی این سرور گوش میده"
    info "مثال: 3x-ui معمولاً روی 10000 یا 2053 هست"
    echo -ne "${CYAN}  ▶ پورت سرویس محلی [پیشفرض 10000]: ${NC}"
    read -r LOCAL_PORT
    LOCAL_PORT=${LOCAL_PORT:-10000}
    ok "پورت محلی: $LOCAL_PORT"
fi

# ══════════════════════════════════════════════════════════════
# STEP 3 — Secret Token
# ══════════════════════════════════════════════════════════════
step "مرحله ۳ از ۴ — توکن امنیتی"
echo ""
echo -e "  ${BOLD}یک رمز مشترک انتخاب کن:${NC}"
info "این رمز باید روی هر دو سرور یکی باشه"
info "هر چیزی میتونه باشه — مثال: MySecret2024"
echo -ne "${CYAN}  ▶ توکن: ${NC}"
read -r TOKEN
[ -z "$TOKEN" ] && err "توکن نمیتونه خالی باشه"
ok "توکن ثبت شد"

# ══════════════════════════════════════════════════════════════
# STEP 4 — Install
# ══════════════════════════════════════════════════════════════
step "مرحله ۴ از ۴ — نصب و راه‌اندازی"
echo ""

# Install dependencies
info "در حال نصب Go و ابزارها..."
apt-get update -qq
apt-get install -y golang-go git curl > /dev/null 2>&1
ok "Go نصب شد"

# Create install dir and copy files
mkdir -p "$INSTALL_DIR"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cp "$SCRIPT_DIR"/*.go "$INSTALL_DIR/" 2>/dev/null || err "فایل‌های .go پیدا نشد — مطمئن شو همه فایل‌ها در یک پوشه‌ان"
cp "$SCRIPT_DIR/go.mod" "$INSTALL_DIR/"

# Write config
if [ "$MODE" == "server" ]; then
    cat > "$INSTALL_DIR/config.json" << EOF
{
    "token": "$TOKEN",
    "server_bind_addr": "0.0.0.0:$CTRL_PORT",
    "server_ports": $PORTS_JSON,
    "pool_size": 8,
    "heartbeat_sec": 10,
    "reconnect_delay_sec": 3
}
EOF
else
    cat > "$INSTALL_DIR/config.json" << EOF
{
    "token": "$TOKEN",
    "remote_addr": "$IRAN_IP:$CTRL_PORT",
    "local_service_addr": "127.0.0.1:$LOCAL_PORT",
    "pool_size": 8,
    "heartbeat_sec": 10,
    "reconnect_delay_sec": 3
}
EOF
fi
ok "فایل کانفیگ ساخته شد"

# Build
info "در حال کامپایل GoTunnel..."
cd "$INSTALL_DIR"
go build -o gotunnel . 2>&1 | while read line; do info "$line"; done
ok "کامپایل موفق"

# Open firewall ports
info "در حال باز کردن پورت‌ها در فایروال..."
if command -v ufw &> /dev/null; then
    ufw allow "$CTRL_PORT"/tcp > /dev/null 2>&1 || true
    if [ "$MODE" == "server" ]; then
        for port in $(echo "$USER_PORTS_RAW" | tr ',' ' '); do
            ufw allow "$port"/tcp > /dev/null 2>&1 || true
        done
    fi
    ok "پورت‌ها در ufw باز شدن"
fi

# Create systemd service
cat > /etc/systemd/system/gotunnel.service << EOF
[Unit]
Description=GoTunnel - High Performance TCP Tunnel
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/gotunnel -mode $MODE -config $INSTALL_DIR/config.json
Restart=always
RestartSec=3
LimitNOFILE=1000000

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable gotunnel > /dev/null 2>&1
systemctl start gotunnel
ok "سرویس systemd ثبت و اجرا شد"

# ─── Summary ──────────────────────────────────────────────────
echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║         GoTunnel نصب شد و در حال اجراست ✓   ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════╝${NC}"
echo ""

if [ "$MODE" == "server" ]; then
    echo -e "  ${BOLD}نوع:${NC}         سرور ایران"
    echo -e "  ${BOLD}پورت کنترل:${NC}  $CTRL_PORT"
    echo -e "  ${BOLD}پورت کاربری:${NC} $USER_PORTS_RAW"
    echo ""
    echo -e "  ${YELLOW}مرحله بعدی:${NC}"
    echo -e "  ← اسکریپت رو روی سرور خارج هم اجرا کن"
    echo -e "  ← بعد پنل (3x-ui یا Marzban) روی سرور خارج نصب کن"
else
    echo -e "  ${BOLD}نوع:${NC}         سرور خارج"
    echo -e "  ${BOLD}سرور ایران:${NC}  $IRAN_IP:$CTRL_PORT"
    echo -e "  ${BOLD}سرویس محلی:${NC} 127.0.0.1:$LOCAL_PORT"
    echo ""
    echo -e "  ${YELLOW}مرحله بعدی:${NC}"
    echo -e "  ← پنل (3x-ui یا Marzban) رو روی همین سرور نصب کن"
    echo -e "  ← پنل رو روی پورت $LOCAL_PORT تنظیم کن"
fi

echo ""
echo -e "  ${BOLD}دستورات مفید:${NC}"
echo -e "  systemctl status gotunnel     ← وضعیت سرویس"
echo -e "  journalctl -u gotunnel -f     ← لاگ زنده"
echo -e "  systemctl restart gotunnel    ← ریستارت"
echo ""
