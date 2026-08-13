FROM node:22-alpine AS build
WORKDIR /src
COPY package.json package-lock.json* ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:1.27.3-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY nginx-main.conf /etc/nginx/nginx.conf
COPY --from=build /src/dist /usr/share/nginx/html
RUN printf "window.__INCIDENTLAB_CONFIG__ = { authMode: 'disabled', clientId: '', tenantId: '', operatorGroupId: '', apiScope: '' };\n" > /usr/share/nginx/html/runtime-config.js
USER nginx
EXPOSE 8080
