FROM nginx:alpine

# Copy file cấu hình có chứa biến môi trường
COPY nginx.conf /etc/nginx/nginx.conf

# Khi container start: thay biến env rồi chạy nginx
CMD envsubst < /etc/nginx/nginx.conf > /etc/nginx/nginx.tmp \
    && mv /etc/nginx/nginx.tmp /etc/nginx/nginx.conf \
    && nginx -g 'daemon off;'
