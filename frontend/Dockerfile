# Dev image: nuxt dev with HMR. Source is bind-mounted at runtime; npm install
# at container start re-syncs deps when package.json changes (no rebuild needed).
FROM node:20-alpine

WORKDIR /app

EXPOSE 3000

ENV HOST=0.0.0.0
ENV PORT=3000
ENV NITRO_HOST=0.0.0.0

# `npm install` is idempotent against the lockfile; Nuxt picks up source changes via HMR.
CMD ["sh", "-c", "npm install --no-audit --no-fund --silent && npm run dev -- --host 0.0.0.0"]
