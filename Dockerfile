# Многоэтапная сборка
# Этап 1: Сборка Node.js фронтенда
# Для production используем фиксированную версию

FROM node:23.10.0-alpine AS frontend-builder
# Установка pnpm
RUN npm install -g pnpm@10.11.0
# Установка рабочей директории
WORKDIR /app
# Копирование файлов фронтенд-проекта
COPY web/ ./
# Установка зависимостей и сборка фронтенда
RUN pnpm install
RUN pnpm build

# Этап 2: Сборка Go бэкенда
FROM golang:1.24.1-alpine AS backend-builder
# Установка переменных окружения Go
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
# Установка рабочей директории
WORKDIR /build
# Копирование всей структуры проекта
COPY coraza-spoa/ ./coraza-spoa/
COPY pkg/ ./pkg/
COPY server/ ./server/
COPY go.work ./
COPY geo-ip/ ./geo-ip/
# Копирование собранных фронтенд-файлов в правильное место
COPY --from=frontend-builder /app/dist ./server/public/dist
# Использование функциональности рабочих пространств Go для сборки
RUN go work use ./coraza-spoa ./pkg ./server
RUN cd server && go build -o ../ruiqi-waf main.go

# Этап 3: Финальный образ - использование официального образа HAProxy 3.0.10
FROM haproxy:3.0.10

# Обеспечиваем выполнение начальных настроек от root
USER root

# Установка инструментов управления Linux capabilities
RUN apt-get update && apt-get install -y libcap2-bin && \
    rm -rf /var/lib/apt/lists/*

# Создание пользователя и группы ruiqi
RUN groupadd --gid 1000 ruiqi && \
    useradd --uid 1000 --gid ruiqi --home-dir /home/ruiqi --create-home --shell /bin/bash ruiqi

# Добавление пользователя ruiqi в группу haproxy для прав на выполнение операций с haproxy
RUN usermod -a -G haproxy ruiqi

# Создание директории приложения и настройка прав
WORKDIR /app
RUN chown ruiqi:ruiqi /app

# Копирование Go-бинарника из сборщика
COPY --from=backend-builder /build/ruiqi-waf .

# Копирование файлов документации Swagger
COPY --from=backend-builder /build/server/docs/ ./docs/

# Установка прав на файлы приложения
RUN chown -R ruiqi:ruiqi /app && chmod +x /app/ruiqi-waf

# Создание директории ruiqi-waf в домашней директории пользователя ruiqi и копирование папки geo-ip
RUN mkdir -p /home/ruiqi/ruiqi-waf
COPY --from=backend-builder /build/geo-ip/ /home/ruiqi/ruiqi-waf/geo-ip/
RUN chown -R ruiqi:ruiqi /home/ruiqi/ruiqi-waf

# 🔑 Ключевой шаг: Добавление возможности привязки к привилегированным портам для HAProxy и приложения
RUN setcap 'cap_net_bind_service=+ep' /usr/local/sbin/haproxy && \
    setcap 'cap_net_bind_service=+ep' /app/ruiqi-waf

# Проверка установленных capabilities (опционально, для отладки)
RUN getcap /usr/local/sbin/haproxy /app/ruiqi-waf

# Теперь можно безопасно переключиться на пользователя ruiqi
USER ruiqi

# Установка переменных окружения
ENV GIN_MODE=release

# Сброс ENTRYPOINT (переопределяем docker-entrypoint.sh базового образа)
ENTRYPOINT []

# Открытие портов: 2333 (приложение)
EXPOSE 2333

# Запуск приложения
CMD ["/app/ruiqi-waf"]