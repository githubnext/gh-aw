> [!WARNING]
> **⚠️ Firewall blocked {domain_count} {domain_word}**
>
> The following {domain_word} {verb} blocked by the firewall during workflow execution:
>
{domain_list}>
{gh_proxy_tip}> To allow these domains, add them to the `network.allowed` list in your workflow frontmatter:
>
> ```yaml
> network:
>   allowed:
>     - defaults
{yaml_network_list}> ```
>
> See [Network Configuration](https://github.github.com/gh-aw/reference/network/) for more information.
