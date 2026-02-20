#!/bin/bash
set -e

# --- SSL certs ---
mkdir -p /etc/pki/tls/certs /etc/pki/tls/private
cp /var/app/staging/ssl/origin-cert.pem /etc/pki/tls/certs/origin-cert.pem
chmod 644 /etc/pki/tls/certs/origin-cert.pem
cp /var/app/staging/ssl/origin-key.pem /etc/pki/tls/private/origin-key.pem
chmod 600 /etc/pki/tls/private/origin-key.pem

# --- Nginx config: copy https.conf and proxy.conf ---
cp /var/app/staging/.platform/nginx/conf.d/https.conf /etc/nginx/conf.d/https.conf
cp /var/app/staging/.platform/nginx/conf.d/proxy.conf /etc/nginx/conf.d/proxy.conf

# --- Fix upstream: point to localhost:8080 instead of container IP:8000 ---
cat > /etc/nginx/conf.d/elasticbeanstalk-nginx-docker-upstream.conf <<'EOF'
upstream docker {
    server 127.0.0.1:8080;
    keepalive 256;
}
EOF

# --- Reload nginx ---
nginx -t && systemctl reload nginx || true
