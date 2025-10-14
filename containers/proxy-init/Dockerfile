FROM alpine:3.18

# Install iptables and iproute2 for network configuration
RUN apk add --no-cache \
    iptables \
    ip6tables \
    iproute2 \
    bash

# Copy init script
COPY proxy-init.sh /proxy-init.sh
RUN chmod +x /proxy-init.sh

# Run as init script
ENTRYPOINT ["/proxy-init.sh"]
