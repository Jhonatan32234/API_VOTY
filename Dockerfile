# Etapa de construcción (Builder)
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Instalar git (necesario a veces para descargar dependencias)
RUN apk add --no-cache git

# Copiar archivos de dependencias primero para aprovechar la caché de Docker
COPY go.mod go.sum ./
RUN go mod download

# Copiar el resto del código fuente
COPY . .

# Compilar el binario
# CGO_ENABLED=0 asegura un binario estático
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server/main.go

# Etapa final (Imagen ligera para producción)
FROM alpine:latest

WORKDIR /app

# Copiar el binario desde la etapa de construcción
COPY --from=builder /app/main .
COPY --from=builder /app/cmd/server/.env .
# Exponer el puerto que usa la aplicación
EXPOSE 8000

# Comando para ejecutar la aplicación
CMD ["./main"]
