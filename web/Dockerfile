# syntax=docker/dockerfile:1

# Production serves the build from S3 behind CloudFront. This image exists
# so docker compose can run the whole stack locally.

# Debian rather than Alpine: TypeScript 7, Vite and Tailwind ship native
# binaries, and glibc builds are the ones every release is tested on.
FROM node:26-slim AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY . .
RUN npm run build

FROM nginx:1.30-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
