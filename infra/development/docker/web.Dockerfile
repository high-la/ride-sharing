# ----------------------------------------
# Stage 1: Dependencies
# ----------------------------------------
FROM node:20-alpine AS deps

WORKDIR /app

# Increase npm network stability
RUN npm config set fetch-retries 5 \
 && npm config set fetch-retry-factor 2 \
 && npm config set fetch-retry-mintimeout 20000 \
 && npm config set fetch-retry-maxtimeout 120000 \
 && npm config set registry https://registry.npmjs.org/

# Copy only package files (better Docker caching)
COPY web/package.json web/package-lock.json ./

# Install dependencies with cache mount
RUN --mount=type=cache,target=/root/.npm \
    npm ci --no-audit --no-fund


# ----------------------------------------
# Stage 2: Builder
# ----------------------------------------
FROM node:20-alpine AS builder

WORKDIR /app

# Reuse installed node_modules
COPY --from=deps /app/node_modules ./node_modules

# Copy application source
COPY web ./

# Disable telemetry
ENV NEXT_TELEMETRY_DISABLED=1

# Build Next.js production bundle
RUN npm run build


# ----------------------------------------
# Stage 3: Production Runner
# ----------------------------------------
FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# Copy built output
COPY --from=builder /app ./

EXPOSE 3000

CMD ["npm", "start"]
