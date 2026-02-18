# using multi stage build to finish web-service build in less time
# Stage 1: deps
FROM node:24-alpine3.23 AS deps

WORKDIR /app

COPY web/package*.json ./

RUN npm ci --prefer-offline --no-audit --no-fund

# Stage 2: builder
FROM node:24-alpine3.23 AS builder

WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules

COPY web ./

RUN npm run build

# Stage 3: runtime
FROM node:24-alpine3.23 AS runner

WORKDIR /app

COPY --from=builder /app ./

EXPOSE 3000

CMD ["npm", "start"]
