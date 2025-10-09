echo 'Extracting access.log from squid-proxy-TOOLNAME container'
if docker ps -a --format '{{.Names}}' | grep -q '^squid-proxy-TOOLNAME$'; then
  docker cp squid-proxy-TOOLNAME:/var/log/squid/access.log /tmp/gh-aw/access-logs/access-TOOLNAME.log 2>/dev/null || echo 'No access.log found for TOOLNAME'
else
  echo 'Container squid-proxy-TOOLNAME not found'
fi
