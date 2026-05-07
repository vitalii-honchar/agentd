---
name: ai-product-research
enabled: false
schedule:
  type: manual
vendor:
  name: openai
  model: gpt-5-mini
tools:
  - name: product-research-python
    kind: local_tool
    command: uv
    args: ["run", "python", "-m", "ai_product_research.main"]
    env:
      - OPENAI_API_KEY
      - PRODUCT_HUNT_DEV_TOKEN
      - TELEGRAM_BOT_TOKEN
      - TELEGRAM_CHANNEL_ID
      - DEBUG
    read_paths:
      - /Users/vitaliihonchar/workspace/ai-product-research
    write_paths:
      - /Users/vitaliihonchar/workspace/ai-product-research/.agentd
    network_allow:
      - api.openai.com
      - api.producthunt.com
      - api.telegram.org
      - https://*
  - name: website-screenshot
    kind: local_tool
    command: uv
    args: ["run", "python", "-m", "playwright", "install", "chromium"]
    env: []
    read_paths:
      - /Users/vitaliihonchar/workspace/ai-product-research
    write_paths:
      - /Users/vitaliihonchar/workspace/ai-product-research/.agentd
    network_allow:
      - playwright.azureedge.net
mcp_servers: []
access:
  filesystem:
    read:
      - /Users/vitaliihonchar/workspace/ai-product-research
    write:
      - /Users/vitaliihonchar/workspace/ai-product-research/.agentd
  network:
    allow:
      - api.openai.com
      - api.producthunt.com
      - api.telegram.org
      - https://*
---
Run the AI product research workflow.

Analyze recent Product Hunt launches, open each product website with the declared Playwright-backed screenshot tool, capture screenshots, retrieve the primary customer, core job, main pain, and success metric from the screenshot, filter for AI products with meaningful revenue potential, and publish the final concise summary through the configured Telegram channel.

Use only the declared local tools, environment variable names, filesystem paths, and network destinations. Never include secret values in logs or output.
