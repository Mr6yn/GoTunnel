#!/bin/bash
# GoTunnel — نصب تعاملی
# اجرا: sudo bash install.sh

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

INSTALL_DIR="/opt/gotunnel"
BINARY="/usr/local/bin/gotunnel"
CMD_WRAPPER="/usr/local/bin/gt"

print_banner() {
    clear
    echo -e "${CYAN}${BOLD}"
    echo "  ____      _____                       _ "
    echo " / ___| ___|_   _|   _ _ __  _ __   ___| |"
    echo "| |  _ / _ \\ | || | | | '_ \\| '_ \\ / _ \\ |"
    echo "| |_| | (_) || || |_| | | | | | | |  __/ |"
    echo " \\____|\\___ / |_| \\__,_|_| |_|_| |_|\\___|_|"
    echo -e "${NC}"
    echo -e "${BOLD}     High Performance TCP Tunnel  v1.0.0${NC}"
    echo -e "     ─────────────────────────────────────"
    echo ""
}

step() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}  $1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

ok()   { echo -e "  ${GREEN}✓ $1${NC}"; }
info() { echo -e "  ${YELLOW}→ $1${NC}"; }
err()  { echo -e "  ${RED}✗ $1${NC}"; exit 1; }

ask() {
    echo -ne "  ${CYAN}▶ $1${NC} "
    read -r REPLY
    echo "$REPLY"
}

# ── بررسی root ────────────────────────────────────────────────
[ "$EUID" -ne 0 ] && err "با دسترسی root اجرا کن: sudo bash install.sh"
[ ! -f /etc/debian_version ] && err "فقط Ubuntu/Debian پشتیبانی میشه"

print_banner

# ══════════════════════════════════════════════
# مرحله ۱ — نوع سرور
# ══════════════════════════════════════════════
step "مرحله ۱ از ۵ — نوع سرور"

echo -e "  ${BOLD}این سرور کدام طرف تانل است؟${NC}"
echo ""
echo -e "  ${YELLOW}1)${NC}  سرور ایران   ← کاربران به این وصل میشن"
echo -e "  ${YELLOW}2)${NC}  سرور خارج   ← پنل و xray اینجاست"
echo ""
echo -ne "  ${CYAN}▶ انتخاب [1/2]:${NC} "
read -r SERVER_TYPE

case "$SERVER_TYPE" in
    1) MODE="server"; ok "سرور ایران انتخاب شد" ;;
    2) MODE="client"; ok "سرور خارج انتخاب شد" ;;
    *) err "فقط 1 یا 2 وارد کن" ;;
esac

# ══════════════════════════════════════════════
# مرحله ۲ — آدرس و پورت‌ها
# ══════════════════════════════════════════════
step "مرحله ۲ از ۵ — آدرس و پورت‌ها"

if [ "$MODE" == "server" ]; then

    echo -e "  ${BOLD}پورت کنترل تانل:${NC}"
    info "سرور خارج از این پورت به سرور ایران وصل میشه"
    echo -ne "  ${CYAN}▶ پورت کنترل [پیشفرض: 8443]:${NC} "
    read -r CTRL_PORT
    CTRL_PORT=${CTRL_PORT:-8443}
    ok "پورت کنترل: $CTRL_PORT"

    echo ""
    echo -e "  ${BOLD}پورت(های) کاربری:${NC}"
    info "کاربران از این پورت‌ها به سرور ایران وصل میشن"
    info "چند پورت رو با ویرگول جدا کن — مثال: 443,80,8080"
    echo -ne "  ${CYAN}▶ پورت(های) کاربری [پیشفرض: 443]:${NC} "
    read -r USER_PORTS_RAW
    USER_PORTS_RAW=${USER_PORTS_RAW:-443}
    PORTS_JSON="[$(echo "$USER_PORTS_RAW" | sed 's/ //g' | sed 's/,/, /g')]"
    ok "پورت‌های کاربری: $USER_PORTS_RAW"

else

    echo -e "  ${BOLD}آی‌پی سرور ایران:${NC}"
    info "آی‌پی سروری که طرف ایران راه‌اندازی کردی"
    echo -ne "  ${CYAN}▶ IP سرور ایران:${NC} "
    read -r IRAN_IP
    [ -z "$IRAN_IP" ] && err "IP نمیتونه خالی باشه"
    ok "IP ایران: $IRAN_IP"

    echo ""
    echo -e "  ${BOLD}پورت کنترل سرور ایران:${NC}"
    info "همون پورتی که روی سرور ایران وارد کردی"
    echo -ne "  ${CYAN}▶ پورت کنترل [پیشفرض: 8443]:${NC} "
    read -r CTRL_PORT
    CTRL_PORT=${CTRL_PORT:-8443}
    ok "پورت کنترل: $CTRL_PORT"

    echo ""
    echo -e "  ${BOLD}پورت سرویس محلی (xray/v2ray/پنل):${NC}"
    info "پورتی که پنل یا xray روی این سرور گوش میده"
    info "3x-ui معمولاً روی 10000 هست"
    echo -ne "  ${CYAN}▶ پورت سرویس محلی [پیشفرض: 10000]:${NC} "
    read -r LOCAL_PORT
    LOCAL_PORT=${LOCAL_PORT:-10000}
    ok "پورت محلی: $LOCAL_PORT"

fi

# ══════════════════════════════════════════════
# مرحله ۳ — توکن امنیتی
# ══════════════════════════════════════════════
step "مرحله ۳ از ۵ — توکن امنیتی"

echo -e "  ${BOLD}یک رمز مشترک انتخاب کن:${NC}"
info "این رمز باید روی هر دو سرور دقیقاً یکی باشه"
info "هر چیزی میتونه باشه — مثال: GoTunnel@2024"
echo -ne "  ${CYAN}▶ توکن:${NC} "
read -r TOKEN
[ -z "$TOKEN" ] && err "توکن نمیتونه خالی باشه"
ok "توکن ثبت شد ✓"

# ══════════════════════════════════════════════
# مرحله ۴ — پورت Status API
# ══════════════════════════════════════════════
step "مرحله ۴ از ۵ — وضعیت‌سنج"

echo -e "  ${BOLD}پورت داخلی برای نمایش وضعیت:${NC}"
info "این پورت فقط روی localhost باز میشه — از بیرون در دسترس نیست"
echo -ne "  ${CYAN}▶ پورت وضعیت [پیشفرض: 9999]:${NC} "
read -r STATUS_PORT
STATUS_PORT=${STATUS_PORT:-9999}
ok "پورت وضعیت: $STATUS_PORT"

# ══════════════════════════════════════════════
# مرحله ۵ — نصب
# ══════════════════════════════════════════════
step "مرحله ۵ از ۵ — نصب و راه‌اندازی"

info "در حال نصب Go و ابزارها..."
apt-get update -qq
apt-get install -y golang-go curl > /dev/null 2>&1
ok "Go نصب شد"

mkdir -p "$INSTALL_DIR"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Copy Go source files
for f in main.go server.go client.go config.go go.mod; do
    [ -f "$SCRIPT_DIR/$f" ] && cp "$SCRIPT_DIR/$f" "$INSTALL_DIR/" || err "فایل $f پیدا نشد"
done
ok "فایل‌های سورس کپی شدن"

# Write config.json
if [ "$MODE" == "server" ]; then
    cat > "$INSTALL_DIR/config.json" << EOF
{
    "token": "$TOKEN",
    "server_bind_addr": "0.0.0.0:$CTRL_PORT",
    "server_ports": $PORTS_JSON,
    "status_port": $STATUS_PORT,
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
    "status_port": $STATUS_PORT,
    "pool_size": 8,
    "heartbeat_sec": 10,
    "reconnect_delay_sec": 3
}
EOF
fi
ok "کانفیگ ساخته شد"

# Build binary
info "در حال کامپایل..."
cd "$INSTALL_DIR"
go build -o gotunnel . 2>&1
cp "$INSTALL_DIR/gotunnel" "$BINARY"
ok "کامپایل موفق"

# Create 'gt' shortcut command
cat > "$CMD_WRAPPER" << 'WRAPPER'
#!/bin/bash
# GoTunnel quick command: gt [status|start|stop|restart|logs]

CONFIG="/opt/gotunnel/config.json"
BINARY="/opt/gotunnel/gotunnel"

case "$1" in
    status|s)
        $BINARY -mode status -config $CONFIG
        ;;
    start)
        systemctl start gotunnel
        echo "✓ GoTunnel شروع شد"
        ;;
    stop)
        systemctl stop gotunnel
        echo "✓ GoTunnel متوقف شد"
        ;;
    restart|r)
        systemctl restart gotunnel
        echo "✓ GoTunnel ریستارت شد"
        ;;
    logs|l)
        journalctl -u gotunnel -f --no-hostname -o short
        ;;
    *)
        echo "GoTunnel — دستورات سریع:"
        echo "  gt status    ← وضعیت تانل"
        echo "  gt start     ← شروع"
        echo "  gt stop      ← توقف"
        echo "  gt restart   ← ریستارت"
        echo "  gt logs      ← لاگ زنده"
        ;;
esac
WRAPPER
chmod +x "$CMD_WRAPPER"
ok "دستور 'gt' نصب شد"

# Firewall
if command -v ufw &> /dev/null; then
    ufw allow "$CTRL_PORT"/tcp > /dev/null 2>&1 || true
    if [ "$MODE" == "server" ]; then
        for port in $(echo "$USER_PORTS_RAW" | tr ',' ' '); do
            ufw allow "$port"/tcp > /dev/null 2>&1 || true
        done
    fi
    ok "پورت‌ها در فایروال باز شدن"
fi

# Systemd service
cat > /etc/systemd/system/gotunnel.service << EOF
[Unit]
Description=GoTunnel TCP Tunnel
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$BINARY -mode $MODE -config $INSTALL_DIR/config.json
Restart=always
RestartSec=3
LimitNOFILE=1000000

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable gotunnel > /dev/null 2>&1
systemctl start gotunnel
ok "سرویس systemd فعال و اجرا شد"

# ── خلاصه نهایی ───────────────────────────────────────────────
echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║       GoTunnel نصب شد و در حال اجراست ✓     ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════╝${NC}"
echo ""

if [ "$MODE" == "server" ]; then
    echo -e "  نوع سرور:       سرور ایران"
    echo -e "  پورت کنترل:     $CTRL_PORT"
    echo -e "  پورت‌های کاربری: $USER_PORTS_RAW"
    echo ""
    echo -e "${YELLOW}  مرحله بعدی:${NC}"
    echo -e "  ① اسکریپت رو روی سرور خارج هم اجرا کن"
    echo -e "  ② بعد پنل (3x-ui یا Marzban) روی سرور خارج نصب کن"
else
    echo -e "  نوع سرور:     سرور خارج"
    echo -e "  سرور ایران:   $IRAN_IP:$CTRL_PORT"
    echo -e "  سرویس محلی:   127.0.0.1:$LOCAL_PORT"
    echo ""
    echo -e "${YELLOW}  مرحله بعدی:${NC}"
    echo -e "  ① پنل (3x-ui یا Marzban) روی همین سرور نصب کن"
    echo -e "  ② پنل رو روی پورت $LOCAL_PORT تنظیم کن"
fi

echo ""
echo -e "${CYAN}  دستورات سریع (با دستور 'gt'):${NC}"
echo -e "  gt status    ← وضعیت تانل + پینگ"
echo -e "  gt logs      ← لاگ زنده"
echo -e "  gt restart   ← ریستارت"
echo ""

# نمایش وضعیت اولیه بعد از 3 ثانیه
sleep 3
echo -e "${BOLD}  وضعیت فعلی:${NC}"
gt status 2>/dev/null || echo "  (چند ثانیه صبر کن و دوباره gt status بزن)"
echo ""
