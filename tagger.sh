#!/bin/bash

# ==============================================================================
# TRANSWARP TAGGER - Documentación de Uso
# ==============================================================================
# Este script automatiza el etiquetado (tagging) de versiones para Transwarp.
# Al ser un MONOREPO con SUBMÓDULOS de Go, las etiquetas deben seguir una 
# estructura específica para que 'go get' funcione correctamente:
#
# 1. ESTRUCTURA DE LOS TAGS:
#    - Core: Se taguea directamente (ej: v1.0.0). Go lo asocia a la raíz.
#    - Adaptadores: Deben incluir la ruta relativa (ej: adapter/echoadapter/v1.0.0).
#
# 2. FLUJO DE PUBLICACIÓN RECOMENDADO:
#    A. Si cambias el Core:
#       1. Taguea el Core (ej: v1.1.0).
#       2. Sube el tag (git push origin v1.1.0).
#    B. Si un Adaptador depende de cambios en el Core:
#       1. Actualiza el go.mod del adaptador para requerir la nueva versión del Core.
#       2. Haz commit de ese cambio.
#       3. Taguea el Adaptador usando este script.
#
# 3. POR QUÉ ESTA ESTRUCTURA:
#    Esto permite que los usuarios importen solo lo que necesitan:
#    go get github.com/profe-ajedrez/transwarp/adapter/echoadapter@v1.1.0
#    Sin descargar las dependencias de Fiber, Gin o Chi.
# ==============================================================================

# Colores para la terminal
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# --- VALIDACIONES INICIALES ---

# 1. Verificar si git está instalado
if ! command -v git &> /dev/null; then
    echo -e "${RED}Error: El comando 'git' no está instalado o no se encuentra en el PATH.${NC}"
    exit 1
fi

# 2. Verificar si es un repositorio git válido
if ! git rev-parse --is-inside-work-tree &> /dev/null; then
    echo -e "${RED}Error: No se detectó un repositorio Git válido en este directorio.${NC}"
    echo -e "${YELLOW}Asegúrate de ejecutar este script desde la raíz de Transwarp.${NC}"
    exit 1
fi

if ! git rev-parse --verify HEAD &> /dev/null; then
    echo -e "${RED}Error: El repositorio no tiene commits.${NC}"
    echo -e "${YELLOW}Debes realizar al menos un commit (git commit) antes de crear un tag.${NC}"
    exit 1
fi

echo -e "${BLUE}=== Transwarp Tagger v2.2 ===${NC}"

# Función para obtener la última versión de un componente
get_last_version() {
    local prefix=$1
    local last_tag
    if [ -z "$prefix" ]; then
        last_tag=$(git tag -l "v*" | grep -v "/" | sort -V | tail -n 1)
    else
        last_tag=$(git tag -l "${prefix}v*" | sort -V | tail -n 1)
    fi
    
    [ -z "$last_tag" ] && echo "Ninguna" || echo "$last_tag"
}

# 1. Selección de Componente
echo "Selecciona el componente a taguear:"
echo "0) Core (Raíz)"
echo "1) Echo Adapter"
echo "2) Fiber Adapter"
echo "3) Gin Adapter"
echo "4) Chi Adapter"
echo "5) Mux Adapter"
read -p "Opción: " COMP

case $COMP in
    0) NAME="Core"; PREFIX="";;
    1) NAME="Echo"; PREFIX="adapter/echoadapter/";;
    2) NAME="Fiber"; PREFIX="adapter/fiberadapter2/";;
    3) NAME="Gin"; PREFIX="adapter/ginadapter/";;
    4) NAME="Chi"; PREFIX="adapter/chiadapter/";;
    5) NAME="Mux"; PREFIX="adapter/mux_adapter/";;
    *) echo -e "${RED}Opción inválida${NC}"; exit 1;;
esac

# 2. Mostrar versiones actuales
echo -e "\n${CYAN}--- Estado Actual en Git ---${NC}"
CORE_VER=$(get_last_version "")
COMP_VER=$(get_last_version "$PREFIX")

echo -e "Última versión del Core: ${YELLOW}$CORE_VER${NC}"
if [ "$COMP" -ne 0 ]; then
    echo -e "Última versión de $NAME: ${YELLOW}$COMP_VER${NC}"
fi
echo -e "${CYAN}----------------------------${NC}\n"

# 3. Solicitar nueva versión
read -p "Introduce la NUEVA versión (ej: v1.0.0): " VERSION

if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${YELLOW}Advertencia: La versión debería seguir el formato vX.Y.Z (SemVer)${NC}"
fi

FULL_TAG="${PREFIX}${VERSION}"

# 4. Verificación de duplicados
if git rev-parse "$FULL_TAG" >/dev/null 2>&1; then
    echo -e "${RED}Error: El tag $FULL_TAG ya existe localmente.${NC}"
    exit 1
fi

echo -e "\nComponente: ${BLUE}$NAME${NC}"
echo -e "Tag final:  ${GREEN}${FULL_TAG}${NC}"
read -p "¿Confirmar creación local? (s/n): " CONFIRM

if [[ $CONFIRM == "s" ]]; then
    git tag -a "$FULL_TAG" -m "Release $NAME $VERSION"
    echo -e "${GREEN}Tag creado localmente.${NC}"
    
    read -p "¿Subir a origin ahora? (s/n): " PUSH
    if [[ $PUSH == "s" ]]; then
        git push origin "$FULL_TAG"
        echo -e "${GREEN}Tag subido con éxito.${NC}"
    fi
else
    echo "Operación cancelada."
fi