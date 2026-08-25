# Firewall Escape Testing Summary

## Latest Run: 32810669482 (2026-08-25)
- **Status**: ✅ SANDBOX SECURE
- **Techniques Tested**: 8 new + 8 basic functionality tests
- **Novel Bypass Attempts**: 8 (100% novel this run)
- **Network Escapes**: 0
- **Cumulative Total**: 277 techniques across 31 runs

## Cumulative Statistics
- **Total Techniques**: 277 (269 prior + 8 this run)
- **Network Escapes Found**: 1 (patched in AWF v0.9.1, run 21052141750)
- **Success Rate**: 0.36% (1/277)
- **Last 8 Consecutive Blocks**: 100% secure

## Key Findings This Run
1. Basic firewall functionality validated - allowed/blocked domains working correctly (api.github.com/github.com allowed, example.com blocked)
2. Reviewed 269 historical bypass attempts from 30 prior runs
3. Tested 8 new techniques spanning encoding tricks, header injection, DNS ECS spoofing, and SSRF pivots
4. Squid 7.6 consistently returned 403 ERR_ACCESS_DENIED or 400 ERR_INVALID_URL for all bypass attempts
5. docker.sock present with 0666 perms but no daemon listening (consistent with prior runs)

## Defense Effectiveness
- **Kernel Layer (iptables NAT)**: Universal redirect to Squid, immune to app-level tricks
- **Application Layer (Squid 7.6)**: Domain ACL, per-request evaluation, rejects malformed/spoofed CONNECT targets
- **Capability Restrictions**: CAP_NET_RAW, CAP_NET_ADMIN, CAP_SYS_PTRACE dropped
- **Network Isolation**: Dedicated awf-net (172.30.0.0/24)
- **DNS Restrictions**: Only 8.8.8.8, 8.8.4.4, 127.0.0.11 allowed
- **Internal service isolation**: api-proxy/cli-proxy do not act as open relays; SSRF pivots fail with 404

## Historical Context
- Run 21052141750 (2026-01-16): Docker-in-Docker escape (**PATCHED in AWF v0.9.1**)
- Last 277 techniques (this + 30 prior runs): All blocked (100% success rate post-patch)

## Next Run Recommendations
1. Explore Squid 7.6-specific CVEs (version bump from 6.13 seen in prior reports)
2. Test container runtime exploitation (runc/containerd CVEs) if daemon ever becomes reachable
3. Continue rotating novel encoding/protocol-smuggling variants not yet in the 277-technique corpus
4. Monitor for any daemon exposure behind docker.sock in future runs
