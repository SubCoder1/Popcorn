FROM nginx:stable-alpine3.17 as builder

RUN rm -rf /etc/nginx/conf.d/default.conf

COPY ./nginx.conf /etc/nginx/conf.d/default.conf

# Remove default nginx index page
RUN rm -rf /usr/share/nginx/html/*

CMD ["nginx", "-g", "daemon off;"]