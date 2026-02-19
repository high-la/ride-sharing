# ----------------------------------------
# Stage 1: Dependencies
# ----------------------------------------
FROM node:20-alpine AS deps

WORKDIR /app

# Prevent DNS / registry weirdness
RUN npm config set registry https://registry.npmjs.org/

# Copy only package files first (better cache)
COPY web/package.json web/package-lock.json* ./

# Install dependencies
RUN npm ci --no-audit --no-fund


# ----------------------------------------
# Stage 2: Builder
# ----------------------------------------
FROM node:20-alpine AS builder

WORKDIR /app

# Copy installed dependencies
COPY --from=deps /app/node_modules ./node_modules

# Copy rest of app
COPY web ./

# Build Next.js app
RUN npm run build


# ----------------------------------------
# Stage 3: Production Runner
# ----------------------------------------
FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production

# Copy built app
COPY --from=builder /app ./

EXPOSE 3000

CMD ["npm", "start"]
