#!/bin/bash
# verify-proxy.sh — runtime sanity checks for an already-running sing-helm daemon.
#
# Checks (each independent, skipped with a warning if its prerequisite is missing):
#   1. daemon status (via `sing-helm status`)
#   2. fake-ip in effect: DNS answers for a foreign domain land in 198.18.0.0/15 / fc00::/18
#   3. end-to-end dial works: foreign domain reachable over HTTPS
#   4. CN domain routes direct / foreign domain routes proxy (via clash-api /connections chains)
#   5. literal IPv6 (bypassing DNS) fails fast instead of hanging (ip_version:6 reject backstop)
#   6. no port-53 DNS packets leak out the physical interface (only meaningful in tun mode)
#   7. optional: a domain that resolves to a private/LAN IP still routes direct
#
# Usage:
#   scripts/verify-proxy.sh [--iface en0] [--foreign-domain www.gstatic.com] \
#       [--cn-domain www.baidu.com] [--lan-domain my-router.local] [--skip-leak]
#
# Requires: sing-helm on PATH, dig, curl. Optional: jq (nicer /connections parsing),
# tcpdump+sudo (leak check).

set -u

IFACE=""
FOREIGN_DOMAIN="www.gstatic.com"
CN_DOMAIN="www.baidu.com"
LAN_DOMAIN=""
SKIP_LEAK=0
DNS_PLACEHOLDER="8.8.8.8" # must match tunDNSPlaceholder in internal/app/daemon/daemon.go

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; FAILED=1; }
warn() { echo -e "${YELLOW}[SKIP]${NC} $1"; }
info() { echo -e "$1"; }

FAILED=0

while [ $# -gt 0 ]; do
    case "$1" in
        --iface) IFACE="$2"; shift 2 ;;
        --foreign-domain) FOREIGN_DOMAIN="$2"; shift 2 ;;
        --cn-domain) CN_DOMAIN="$2"; shift 2 ;;
        --lan-domain) LAN_DOMAIN="$2"; shift 2 ;;
        --skip-leak) SKIP_LEAK=1; shift ;;
        -h|--help) grep '^#' "$0" | grep -v '^#!' | sed 's/^# \{0,1\}//'; exit 0 ;;
        *) echo "unknown arg: $1"; exit 1 ;;
    esac
done

command -v sing-helm >/dev/null 2>&1 || { echo "sing-helm not found on PATH"; exit 1; }
command -v dig >/dev/null 2>&1 || { echo "dig not found"; exit 1; }
command -v curl >/dev/null 2>&1 || { echo "curl not found"; exit 1; }
HAS_JQ=0; command -v jq >/dev/null 2>&1 && HAS_JQ=1
HAS_TCPDUMP=0; command -v tcpdump >/dev/null 2>&1 && HAS_TCPDUMP=1

echo "=== 1. daemon status ==="
STATUS="$(sing-helm status 2>&1)"
echo "$STATUS"
if ! echo "$STATUS" | grep -q "Running: true"; then
    fail "daemon is not running (run 'sing-helm start' first)"
    exit 1
fi
PROXY_MODE="$(echo "$STATUS" | sed -n 's/^Proxy mode: //p')"
API_HOSTPORT="$(echo "$STATUS" | sed -n 's/^API: //p')"
if [ -z "$API_HOSTPORT" ]; then
    warn "could not read clash-api address from 'sing-helm status'; chain checks (4) will be skipped"
fi
info "proxy_mode=$PROXY_MODE api=$API_HOSTPORT"
echo

echo "=== 2. fake-ip in effect (DNS answers should be placeholder addresses, not real IPs) ==="
A_ANSWER="$(dig +short +time=3 +tries=1 @"$DNS_PLACEHOLDER" "$FOREIGN_DOMAIN" A | tail -1)"
if [[ "$A_ANSWER" =~ ^198\.(18|19)\. ]]; then
    pass "A answer for $FOREIGN_DOMAIN is fake-ip: $A_ANSWER"
else
    fail "A answer for $FOREIGN_DOMAIN is NOT in fake-ip range 198.18.0.0/15: got '${A_ANSWER:-<empty>}'"
fi

AAAA_ANSWER="$(dig +short +time=3 +tries=1 @"$DNS_PLACEHOLDER" "$FOREIGN_DOMAIN" AAAA | tail -1)"
if [[ "$AAAA_ANSWER" =~ ^fc00: ]]; then
    pass "AAAA answer for $FOREIGN_DOMAIN is fake-ip: $AAAA_ANSWER"
elif [ -z "$AAAA_ANSWER" ]; then
    warn "no AAAA answer for $FOREIGN_DOMAIN (domain may not publish one upstream; not necessarily a bug)"
else
    fail "AAAA answer for $FOREIGN_DOMAIN is NOT in fake-ip range fc00::/18: got '$AAAA_ANSWER'"
fi
echo

echo "=== 3. end-to-end dial works (fake-ip -> resolve -> route -> real outbound) ==="
CODE="$(curl -s -o /dev/null -w '%{http_code}' --max-time 8 "https://$FOREIGN_DOMAIN/generate_204")"
if [ "$CODE" = "204" ] || [ "$CODE" = "200" ]; then
    pass "https://$FOREIGN_DOMAIN reachable (HTTP $CODE)"
else
    fail "https://$FOREIGN_DOMAIN NOT reachable (HTTP ${CODE:-<none>})"
fi
echo

echo "=== 4. routing chains via clash-api /connections ==="
if [ -z "$API_HOSTPORT" ]; then
    warn "no API address, skipping chain check"
elif [ "$HAS_JQ" = 0 ]; then
    warn "jq not installed, skipping chain check (install jq to enable)"
else
    # Hold both connections open with a slow endpoint while polling /connections,
    # so they're still active (not yet closed) when we read the chain.
    curl -s -o /dev/null --max-time 6 "https://$FOREIGN_DOMAIN/generate_204?slow=$RANDOM" &
    FOREIGN_CURL_PID=$!
    curl -s -o /dev/null --max-time 6 "https://$CN_DOMAIN/?slow=$RANDOM" &
    CN_CURL_PID=$!
    sleep 1
    curl -s --max-time 4 "http://$API_HOSTPORT/connections" > /tmp/verify-proxy-connections.json 2>/dev/null
    wait "$FOREIGN_CURL_PID" "$CN_CURL_PID" 2>/dev/null

    if [ -s /tmp/verify-proxy-connections.json ]; then
        FOREIGN_CHAIN="$(jq -r --arg h "$FOREIGN_DOMAIN" '.connections[] | select(.metadata.host==$h) | .chains[0]' /tmp/verify-proxy-connections.json | tail -1)"
        CN_CHAIN="$(jq -r --arg h "$CN_DOMAIN" '.connections[] | select(.metadata.host==$h) | .chains[0]' /tmp/verify-proxy-connections.json | tail -1)"

        if [ -n "$FOREIGN_CHAIN" ] && [ "$FOREIGN_CHAIN" != "direct" ]; then
            pass "$FOREIGN_DOMAIN routed via '$FOREIGN_CHAIN' (not direct, as expected)"
        elif [ -z "$FOREIGN_CHAIN" ]; then
            warn "$FOREIGN_DOMAIN not found in /connections (request may have completed too fast)"
        else
            fail "$FOREIGN_DOMAIN routed via 'direct', expected proxy"
        fi

        if [ "$CN_CHAIN" = "direct" ]; then
            pass "$CN_DOMAIN routed via 'direct', as expected"
        elif [ -z "$CN_CHAIN" ]; then
            warn "$CN_DOMAIN not found in /connections (request may have completed too fast)"
        else
            fail "$CN_DOMAIN routed via '$CN_CHAIN', expected direct"
        fi
    else
        warn "empty response from /connections"
    fi
    rm -f /tmp/verify-proxy-connections.json
fi
echo

echo "=== 5. literal IPv6 (bypassing DNS) fails fast instead of hanging ==="
START=$(date +%s)
curl -6 -s -o /dev/null --max-time 5 "http://[2606:4700:4700::1111]" 2>/dev/null
END=$(date +%s)
ELAPSED=$((END - START))
if [ "$ELAPSED" -le 4 ]; then
    pass "literal IPv6 connect failed/returned within ${ELAPSED}s (reject backstop working, not hanging)"
else
    fail "literal IPv6 connect took ${ELAPSED}s (close to timeout; reject rule may not be catching this)"
fi
echo

if [ "$SKIP_LEAK" = 1 ]; then
    warn "6. DNS leak check skipped (--skip-leak)"
elif [ "$PROXY_MODE" != "tun" ]; then
    warn "6. DNS leak check skipped (only meaningful in tun mode, current mode: ${PROXY_MODE:-unknown})"
elif [ "$HAS_TCPDUMP" = 0 ]; then
    warn "6. DNS leak check skipped (tcpdump not found)"
elif [ -z "$IFACE" ]; then
    warn "6. DNS leak check skipped (pass --iface en0, the physical uplink, to enable)"
else
    echo "=== 6. DNS leak check on physical interface $IFACE ==="
    CAP_FILE="$(mktemp)"
    sudo timeout 6 tcpdump -i "$IFACE" -n "udp port 53" -c 20 -w "$CAP_FILE" >/tmp/verify-proxy-tcpdump.log 2>&1 &
    TCPDUMP_PID=$!
    sleep 1
    dig +time=2 +tries=1 @"$DNS_PLACEHOLDER" "leak-test-$RANDOM.example.com" A >/dev/null 2>&1
    wait "$TCPDUMP_PID" 2>/dev/null
    PACKET_COUNT="$(sudo tcpdump -r "$CAP_FILE" 2>/dev/null | wc -l | tr -d ' ')"
    rm -f "$CAP_FILE"
    if [ "${PACKET_COUNT:-0}" = "0" ]; then
        pass "no port-53 packets seen on physical interface $IFACE (no DNS leak)"
    else
        fail "$PACKET_COUNT port-53 packet(s) leaked out on $IFACE"
    fi
    echo
fi

if [ -n "$LAN_DOMAIN" ]; then
    echo "=== 7. LAN-private domain routes direct (validates the 'action:resolve' route rule) ==="
    LAN_ANSWER="$(dig +short +time=3 +tries=1 @"$DNS_PLACEHOLDER" "$LAN_DOMAIN" A | tail -1)"
    info "fake-ip answer for $LAN_DOMAIN: ${LAN_ANSWER:-<empty>}"
    curl -s -o /dev/null --max-time 6 "http://$LAN_DOMAIN" 2>/dev/null &
    LAN_CURL_PID=$!
    sleep 1
    if [ -n "$API_HOSTPORT" ] && [ "$HAS_JQ" = 1 ]; then
        LAN_CHAIN="$(curl -s --max-time 4 "http://$API_HOSTPORT/connections" 2>/dev/null | \
            jq -r --arg h "$LAN_DOMAIN" '.connections[] | select(.metadata.host==$h) | .chains[0]' | tail -1)"
        if [ "$LAN_CHAIN" = "direct" ]; then
            pass "$LAN_DOMAIN routed via 'direct', as expected"
        elif [ -z "$LAN_CHAIN" ]; then
            warn "$LAN_DOMAIN not found in /connections (request may have completed too fast)"
        else
            fail "$LAN_DOMAIN routed via '$LAN_CHAIN', expected direct"
        fi
    else
        warn "no API address or jq, cannot assert chain automatically"
    fi
    wait "$LAN_CURL_PID" 2>/dev/null
else
    warn "7. LAN-private domain check skipped (pass --lan-domain <domain-resolving-to-a-private-ip> to enable)"
fi

echo
if [ "$FAILED" = 1 ]; then
    echo -e "${RED}one or more checks FAILED${NC}"
    exit 1
fi
echo -e "${GREEN}all checks passed (see [SKIP] lines for anything not covered)${NC}"
