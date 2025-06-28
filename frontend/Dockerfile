FROM nginx:alpine

# Instalar bash para o script
RUN apk add --no-cache bash

# Copiar arquivos do frontend
COPY index.html /usr/share/nginx/html/
COPY nginx.conf /etc/nginx/nginx.conf
COPY entrypoint.sh /entrypoint.sh

# Tornar o script executável
RUN chmod +x /entrypoint.sh

EXPOSE 80

CMD ["/entrypoint.sh"]
