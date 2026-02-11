#!/bin/bash

# Nombre del archivo de salida
OUTPUT="transwarp_bundle.txt"

# Colores para la terminal
BLUE='\033[0;34m'
GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${BLUE}Generando paquete de código en $OUTPUT...${NC}"

# Limpiar archivo anterior si existe
> "$OUTPUT"

# Lista de extensiones a incluir
# Puedes añadir más si usas .sql, .yaml, etc.
EXTENSIONS=("*.go" "*.mod" "*.sh" "*.md")

# Función para añadir archivos
bundle_files() {
    for ext in "${EXTENSIONS[@]}"; do
        # Buscamos archivos ignorando carpetas pesadas o irrelevantes
        find . -name "$ext" \
            -not -path "*/.*" \
            -not -path "*/vendor/*" \
            -not -path "*/bin/*" \
            -type f | while read -r file; do
            
            echo "Añadiendo: $file"
            echo "==============================================================================" >> "$OUTPUT"
            echo "FILE: $file" >> "$OUTPUT"
            echo "==============================================================================" >> "$OUTPUT"
            cat "$file" >> "$OUTPUT"
            echo -e "\n\n" >> "$OUTPUT"
        done
    done
}

bundle_files

echo -e "${GREEN}¡Listo! Todo el código ha sido volcado en $OUTPUT${NC}"