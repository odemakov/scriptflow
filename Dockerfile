# Build Frontend
FROM node:alpine AS builder-frontend
WORKDIR /app

COPY frontend/package.json frontend/package-lock.json ./
RUN npm install

COPY frontend ./
RUN npm run build

# Build Backend
FROM golang:1.23-alpine AS builder-backend
WORKDIR /app

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/*.go ./
COPY backend/migrations/*.go ./migrations/
COPY --from=builder-frontend /app/dist ./dist
RUN go build -tags production -o scriptflow

# Production App
FROM alpine:latest AS app
WORKDIR /app

COPY --from=builder-backend /app/scriptflow .
RUN chmod +x /app/scriptflow

EXPOSE 8090
CMD ["/app/scriptflow", "serve", "--http=0.0.0.0:8090"]

# Development test VMs
FROM alpine:latest AS dev-vm

RUN apk add --no-cache openssh && \
    mkdir -p /root/.ssh && \
    chmod 700 /root/.ssh && \
    echo "PermitRootLogin yes" >> /etc/ssh/sshd_config && \
    echo "PasswordAuthentication no" >> /etc/ssh/sshd_config && \
    echo "PubkeyAuthentication yes" >> /etc/ssh/sshd_config && \
    echo "HostKey /etc/ssh/ssh_host_rsa_key" >> /etc/ssh/sshd_config

# Generate SSH host key pair
RUN ssh-keygen -t rsa -f /etc/ssh/ssh_host_rsa_key -q -N ""

# Generate SSH key pair for ssh to test VMs
RUN mkdir -p /root/.ssh && \
    ssh-keygen -t rsa -b 2048 -f /root/.ssh/id_rsa -q -N "" && \
    chmod 600 /root/.ssh/id_rsa && \
    chmod 644 /root/.ssh/id_rsa.pub

RUN cp /root/.ssh/id_rsa.pub /root/.ssh/authorized_keys

EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]

# Development Backend
FROM golang:1.23-alpine AS dev-backend
WORKDIR /app

RUN apk add --no-cache openssh-client
 
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend ./
RUN go install github.com/air-verse/air@latest

COPY --from=dev-vm /root/.ssh/id_rsa.pub /root/.ssh/id_rsa.pub
COPY --from=dev-vm /root/.ssh/id_rsa /root/.ssh/id_rsa

EXPOSE 8090
CMD ["air", "--build.cmd", "go build -o scriptflow", "--build.bin", "./scriptflow serve --http 0.0.0.0:8090 --dev", "--build.exclude_dir", "pb_data,sf_logs"]

# Development Frontend
FROM node:alpine AS dev-frontend
WORKDIR /app

COPY frontend/package.json frontend/package-lock.json ./
RUN npm install

COPY frontend ./
CMD ["npm", "run", "dev"]
