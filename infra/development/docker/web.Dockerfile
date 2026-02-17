# FROM node:20-alpine
FROM node:24-alpine3.23


WORKDIR /app

COPY web/package*.json ./

RUN npm config set fetch-retry-maxtimeout 600000 && \
    npm config set fetch-retries 5 && \
    npm config set fetch-timeout 600000 && \
    npm install --no-audit --no-fund

COPY web ./

RUN npm run build

EXPOSE 3000

CMD ["npm", "start"]