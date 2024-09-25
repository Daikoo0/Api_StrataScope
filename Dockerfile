# Usa una imagen de Go para construir la aplicación
FROM golang:1.20-alpine AS build

WORKDIR /app

# Copia el go.mod y go.sum para instalar las dependencias
COPY go.mod go.sum ./

RUN go mod download

# Copia el código fuente
COPY . .

# Compila la aplicación
RUN go build -o server .

# Segunda etapa: Imagen minimalista para producción
FROM alpine:latest

WORKDIR /root/

# Copia el binario compilado desde la etapa anterior
COPY --from=build /app/server .
COPY .env .env
# Exponer el puerto que usa el backend
EXPOSE 8080

# Comando para ejecutar el servidor Go
CMD ["./server"]
