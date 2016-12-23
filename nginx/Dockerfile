FROM nginx:1-alpine
COPY nginx.vh.default.conf /etc/nginx/conf.d/default.conf
RUN rm /usr/share/nginx/html/*
COPY styles.css /usr/share/nginx/html
