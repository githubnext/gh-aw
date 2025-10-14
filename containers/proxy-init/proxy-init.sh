#!/bin/bash
set -e

echo "================================================"
echo "GitHub Agentic Workflows - Proxy Init Container"
echo "Setting up transparent proxy with iptables..."
echo "================================================"
echo ""

# Wait a moment for network stack to be ready
sleep 1

echo "[1/4] Setting up HTTP traffic redirection (port 80 -> 3128)..."
# HTTP traffic -> Squid port 3128 (REDIRECT)
# This captures all outgoing HTTP traffic and redirects it to Squid
iptables -t nat -A OUTPUT -p tcp --dport 80 -j REDIRECT --to-port 3128
echo "✓ HTTP REDIRECT configured"
echo ""

echo "[2/4] Setting up policy routing for HTTPS traffic..."
# HTTPS traffic -> Squid port 3129 (TPROXY with policy routing)
# Create routing table 100 for TPROXY marked packets
ip rule add fwmark 1 lookup 100 2>/dev/null || echo "  (rule already exists)"
ip route add local 0.0.0.0/0 dev lo table 100 2>/dev/null || echo "  (route already exists)"
echo "✓ Policy routing configured (table 100)"
echo ""

echo "[3/4] Setting up TPROXY for HTTPS traffic (port 443 -> 3129)..."
# TPROXY rule for HTTPS traffic (PREROUTING chain)
# This preserves the original destination IP, allowing Squid to see where the connection is going
iptables -t mangle -A PREROUTING -p tcp --dport 443 \
  -j TPROXY --tproxy-mark 0x1/0x1 --on-port 3129

# Also handle OUTPUT chain for locally-generated HTTPS traffic
# Mark the packets so they get routed through table 100
iptables -t mangle -A OUTPUT -p tcp --dport 443 \
  -j MARK --set-mark 1
echo "✓ TPROXY configured for HTTPS (with mark 0x1)"
echo ""

echo "[4/4] Verifying iptables configuration..."
echo ""
echo "--- NAT rules (HTTP REDIRECT) ---"
iptables -t nat -L OUTPUT -v -n | grep -E "REDIRECT|tcp dpt:80" || echo "  (no matching rules)"
echo ""
echo "--- Mangle rules (HTTPS TPROXY) ---"
iptables -t mangle -L PREROUTING -v -n | grep -E "TPROXY|tcp dpt:443" || echo "  (no matching rules)"
iptables -t mangle -L OUTPUT -v -n | grep -E "MARK|tcp dpt:443" || echo "  (no matching rules)"
echo ""
echo "--- Policy routing rules ---"
ip rule list | grep "fwmark 0x1 lookup 100" || echo "  (no matching rules)"
echo ""
echo "--- Routing table 100 ---"
ip route show table 100 || echo "  (empty table)"
echo ""

echo "================================================"
echo "✓ Proxy initialization complete!"
echo "================================================"
echo ""
echo "Summary:"
echo "  HTTP  (port 80)  -> REDIRECT to Squid port 3128"
echo "  HTTPS (port 443) -> TPROXY to Squid port 3129 (preserves destination)"
echo ""
echo "Container will now exit. iptables rules persist in shared network namespace."
echo ""
