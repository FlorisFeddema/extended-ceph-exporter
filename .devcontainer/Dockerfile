FROM mcr.microsoft.com/devcontainers/base:bookworm

RUN apt-get update && apt-get install -y --no-install-recommends curl && rm -rf /var/lib/apt/lists/*

RUN if id -u vscode >/dev/null 2>&1; then \
      usermod -l dev vscode && \
      groupmod -n dev vscode && \
      usermod -d /home/dev -m dev; \
    fi

RUN mkdir -p /home/dev/.codex && chown -R dev:dev /home/dev/.codex
