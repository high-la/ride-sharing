# -------- Build Stage --------
FROM node:20-alpine AS builder

WORKDIR /app

COPY web/package*.json ./
RUN npm ci

COPY web .
RUN npm run build


# -------- Production Stage --------
FROM node:20-alpine

WORKDIR /app

ENV NODE_ENV=production

COPY --from=builder /app/package*.json ./
RUN npm ci --omit=dev

COPY --from=builder /app/.next ./.next
# COPY --from=builder /app/public ./public
COPY --from=builder /app/next.config.ts ./

EXPOSE 3000

# CMD ["npm","start"]
# production stage
CMD ["npx", "next", "start", "-p", "3000", "-H", "0.0.0.0"]