# dezetrein website image. Build from the web folder:
#   docker build -t dezetrein-web .
# BACKEND_URL (default http://backend:8080) is where nginx forwards /api.

FROM node:26-alpine AS build
RUN apk upgrade --no-cache
WORKDIR /app
COPY package.json package-lock.json ./
# --include=optional: platform-specific native binaries (rolldown, tailwind oxide) for this image's arch.
RUN npm ci --include=optional
COPY . .
RUN npm run build

FROM nginx:alpine
RUN apk upgrade --no-cache
COPY nginx.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /app/dist /usr/share/nginx/html
# Run as the image's non-root nginx user: it needs to write the rendered config, cache and pid.
RUN rm -f /etc/nginx/conf.d/default.conf \
 && chown -R nginx:nginx /usr/share/nginx/html /etc/nginx/conf.d /var/cache/nginx \
 && touch /var/run/nginx.pid && chown nginx:nginx /var/run/nginx.pid
ENV BACKEND_URL=http://backend:8080
USER nginx
EXPOSE 8080
