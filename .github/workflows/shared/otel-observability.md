---
env:
  OTEL_BACKEND_URL: ${{ secrets.OTLP_ENDPOINT }}
  OTEL_BACKEND_TOKEN: ${{ secrets.OTLP_TOKEN }}
observability:
  otlp:
    endpoint:
      url: ${{ secrets.OTLP_ENDPOINT }}
      headers:
        Authorization: ${{ secrets.OTLP_TOKEN }}
mcp-servers:
  otel:
    command: npx
    args: ["@your-org/otel-query-mcp"]
    env:
      OTEL_BACKEND_URL: ${{ env.OTEL_BACKEND_URL }}
      OTEL_BACKEND_TOKEN: ${{ env.OTEL_BACKEND_TOKEN }}
---
