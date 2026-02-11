#!/bin/bash

###############################################################################
# Transwarp Module Patcher
#
# DESCRIPCIÓN:
#   Este script gestiona las directivas 'replace' en los archivos go.mod de los
#   submódulos (adaptadores) del monorepo Transwarp. 
#
#   Es necesario porque 'golangci-lint' prohíbe reemplazos locales, pero el 
#   desarrollo activo requiere que los adaptadores apunten al Core localmente.
#
# FUNCIONAMIENTO:
#   - Modo 'off': Comenta el 'replace' y ejecuta 'go mod tidy'. Útil para CI/CD 
#     y procesos de linting.
#   - Modo 'on' : Descomenta el 'replace' y ejecuta 'go mod tidy'. Útil para 
#     desarrollo local y pruebas de integración.
#
# REQUISITOS:
#   - Ejecutar desde la raíz del proyecto.
#   - Tener instalado 'go' y 'sed' (estándar en Linux/macOS).
#
# AUTOR: profe-ajedrez
# PROYECTO: github.com/profe-ajedrez/transwarp
###############################################################################

# Colores para la salida
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

TARGET="replace github.com/profe-ajedrez/transwarp => ../../"

run_tidy() {
    echo -e "${CYAN}Ejecutando go mod tidy en los módulos afectados...${NC}"
    # Buscamos todos los directorios que contienen un go.mod dentro de /adapter
    find ./adapter -name "go.mod" -printf "%h\n" | while read -r dir; do
        echo -e "  -> Tidy en $dir"
        (cd "$dir" && go mod tidy)
    done
}

if [[ "$1" == "off" ]]; then
    echo -e "${YELLOW}Comentando directivas replace (Modo Linting/Release)...${NC}"
    find ./adapter -name "go.mod" -exec sed -i "s|^${TARGET}|// ${TARGET}|g" {} +
    run_tidy
    echo -e "${GREEN}¡Hecho! Reemplazos desactivados y módulos limpios.${NC}"

elif [[ "$1" == "on" ]]; then
    echo -e "${CYAN}Activando directivas replace (Modo Desarrollo Local)...${NC}"
    find ./adapter -name "go.mod" -exec sed -i "s|^// ${TARGET}|${TARGET}|g" {} +
    run_tidy
    echo -e "${GREEN}¡Hecho! Reemplazos activados y módulos vinculados.${NC}"

else
    echo -e "${CYAN}Uso: $0 [on|off]${NC}"
    exit 1
fi