FROM node:20-alpine AS build
WORKDIR /app
ARG VITE_API_BASE=http://localhost:8080
ARG VITE_SITE_ID=1
ENV VITE_API_BASE=$VITE_API_BASE
ENV VITE_SITE_ID=$VITE_SITE_ID
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
