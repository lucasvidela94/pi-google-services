#!/bin/bash
# install.sh — Instala pi-google-services en Pi
set -euo pipefail

BIN_DIR="${HOME}/.local/bin"
PI_MCP="${HOME}/.pi/agent/mcp.json"
BINARY="pi-google-services"
VERSION="0.1.0"

echo "=== pi-google-services v${VERSION} ==="
echo ""

# 1. Instalar binario (tiene las credenciales de OAuth embebidas)
mkdir -p "${BIN_DIR}"
cp "./${BINARY}" "${BIN_DIR}/${BINARY}"
chmod +x "${BIN_DIR}/${BINARY}"
echo "✅ Binario instalado en ${BIN_DIR}/${BINARY}"

# 2. Configurar en Pi MCP
if [ -f "${PI_MCP}" ]; then
	if grep -q "google-services" "${PI_MCP}" 2>/dev/null; then
		echo "✅ google-services ya está configurado en Pi MCP"
	else
		python3 -c "
import json
with open('${PI_MCP}') as f:
    cfg = json.load(f)
cfg.setdefault('mcpServers', {})['google-services'] = {
    'command': '${BIN_DIR}/${BINARY}',
    'args': ['serve']
}
with open('${PI_MCP}', 'w') as f:
    json.dump(cfg, f, indent=2)
print('✅ Configurado en Pi MCP')
" 2>&1
	fi
else
	echo "⚠ No se encontró ${PI_MCP}"
	echo "   Agregá este server MCP manualmente:"
fi

echo ""
echo "=== Instalación completa ==="
echo ""
echo "Primer uso:"
echo "  1. Corré:  ${BINARY} login"
echo "  2. Se abre el navegador → autorizás con Google → listo"
echo "  3. Reiniciá la sesión de Pi"
echo "  4. ¡A gestionar tu calendario! 🚀"
echo ""
echo "Para distribuir a un compa:"
echo "  Solo necesita el binario y correr 'login' — sin Google Cloud Console"
