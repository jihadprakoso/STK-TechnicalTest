FROM node:20-alpine AS base
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./

FROM base AS dependencies
RUN npm ci

FROM dependencies AS development
COPY frontend/ ./
EXPOSE 3000
CMD ["npm", "run", "dev", "--", "--hostname", "0.0.0.0"]

FROM dependencies AS build
ARG NEXT_PUBLIC_API_URL=http://localhost:8080/api
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}
COPY frontend/ ./
RUN npm run build

FROM node:20-alpine AS production
ENV NODE_ENV=production
WORKDIR /app
COPY --from=build /app/public ./public
COPY --from=build /app/.next/standalone ./
COPY --from=build /app/.next/static ./.next/static
EXPOSE 3000
CMD ["node", "server.js"]
