#!/bin/bash

###############################################################################
# Transwarp Module Patcher
#
# DESCRIPCIÓN:
#   Gestiona las directivas 'replace' en los archivos go.mod de los submódulos
#   (adaptadores) y de los ejemplos dentro del monorepo Transwarp.
#
#   Es vital para el flujo de trabajo:
#   - Modo 'off': Comenta el 'replace' para que golangci-lint sea feliz.
#   - Modo 'on' : Descomenta el 'replace' para poder ejecutar los ejemplos y 
#     desarrollar los adaptadores localmente.
#
# ÁREAS DE IMPACTO:
#   - ./adapter/**
#   - ./examples/**
###############################################################################

# Colores para la salida
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

TARGET="replace github.com/iaconlabs/transwarp"

# Función para ejecutar go mod tidy en cada módulo encontrado
run_tidy() {
    echo -e "${CYAN}Sincronizando dependencias (go mod tidy)...${NC}"
    # Buscamos en adapter y examples
    find ./adapter ./examples -name "go.mod" -printf "%h\n" 2>/dev/null | while read -r dir; do
        echo -e "  -> Procesando: $dir"
        (cd "$dir" && go mod tidy)
    done
}

if [[ "$1" == "off" ]]; then
    echo -e "${YELLOW}Desactivando reemplazos locales (Modo Linting/Release)...${NC}"
    # Comenta la línea en adapter y examples
    find ./adapter ./examples -name "go.mod" -exec sed -i "s|^replace github.com/iaconlabs/transwarp|// replace github.com/iaconlabs/transwarp|g" {} + 2>/dev/null
    run_tidy
    echo -e "${GREEN}¡Operación completada! Reemplazos comentados.${NC}"

elif [[ "$1" == "on" ]]; then
    echo -e "${CYAN}Activando reemplazos locales (Modo Desarrollo)...${NC}"
    # Descomenta la línea en adapter y examples
	find ./adapter ./examples -name "go.mod" -exec sed -i "s|^// replace github.com/iaconlabs/transwarp|replace github.com/iaconlabs/transwarp|g" {} + 2>/dev/null
    run_tidy
    echo -e "${GREEN}¡Operación completada! Reemplazos activos.${NC}"

else
    echo -e "${YELLOW}Uso: $0 [on|off]${NC}"
    echo "  on  : Habilita el desarrollo local (unifica el monorepo)."
    echo "  off : Prepara para el linter (comenta dependencias locales)."
    exit 1
fi