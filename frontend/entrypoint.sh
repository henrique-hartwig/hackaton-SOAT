#!/bin/bash

echo "🚀 Iniciando Frontend..."

# Verificar se as variáveis de ambiente estão definidas
if [ -z "$API_BASE_URL" ]; then
    echo "⚠️  API_BASE_URL não definida, usando valor padrão"
    export API_BASE_URL="http://localhost:8000"
fi

if [ -z "$UPLOAD_BASE_URL" ]; then
    echo "⚠️  UPLOAD_BASE_URL não definida, usando valor padrão"
    export UPLOAD_BASE_URL="http://localhost:8081"
fi

echo "📡 API_BASE_URL: $API_BASE_URL"
echo "📤 UPLOAD_BASE_URL: $UPLOAD_BASE_URL"

# Substituir variáveis de ambiente no HTML
echo "🔧 Substituindo variáveis no index.html..."
sed -i "s|\${API_BASE_URL}|$API_BASE_URL|g" /usr/share/nginx/html/index.html
sed -i "s|\${UPLOAD_BASE_URL}|$UPLOAD_BASE_URL|g" /usr/share/nginx/html/index.html

echo "✅ Frontend configurado com sucesso!"
echo "🌐 Iniciando Nginx..."

# Iniciar nginx
exec nginx -g "daemon off;" 