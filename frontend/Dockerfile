# 1) Build stage
FROM node:18-alpine AS build

# set working dir
WORKDIR /app

# install dependencies
COPY package*.json ./
RUN npm ci

# copy source and build
COPY . .
RUN npm run build


FROM nginx:stable-alpine
RUN rm -rf /usr/share/nginx/html/*
COPY --from=build /app/build /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
