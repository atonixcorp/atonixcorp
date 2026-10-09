# ============================
# 1 — Build Stage
# ============================
FROM node:22-alpine AS builder
WORKDIR /app

COPY package*.json ./
RUN npm install --legacy-peer-deps

COPY . .
RUN npm run build


# ============================
# 2 — Production Runner
# ============================
FROM node:22-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production

# Install only production dependencies
COPY package*.json ./
RUN npm install --omit=dev --legacy-peer-deps

# Copy build output
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public

# Copy config files
COPY --from=builder /app/next.config.js ./next.config.js

# Expose port
EXPOSE 3000

# Start Next.js
CMD ["npm", "start"]
