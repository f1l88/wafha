#!/bin/bash

set -e

echo "🚀 Начало сборки RuiQi WAF..."

# Проверка окружения
echo "🔍 Проверка среды сборки..."

# Проверка версии Node.js
REQUIRED_NODE="23.10.0"
if command -v node &> /dev/null; then
    NODE_VERSION=$(node --version | sed 's/v//')
    echo "📦 Версия Node.js: $NODE_VERSION (требуется: $REQUIRED_NODE)"
    if [ "$NODE_VERSION" != "$REQUIRED_NODE" ]; then
        echo "⚠️  Предупреждение: Версия Node.js не соответствует, рекомендуется использовать v$REQUIRED_NODE"
    fi
else
    echo "❌ Ошибка: Node.js не найден, пожалуйста установите Node.js $REQUIRED_NODE"
    exit 1
fi

# Проверка версии pnpm
REQUIRED_PNPM="10.11.0"
if command -v pnpm &> /dev/null; then
    PNPM_VERSION=$(pnpm --version)
    echo "📦 Версия pnpm: $PNPM_VERSION (требуется: $REQUIRED_PNPM)"
    if [ "$PNPM_VERSION" != "$REQUIRED_PNPM" ]; then
        echo "⚠️  Предупреждение: Версия pnpm не соответствует, рекомендуется использовать $REQUIRED_PNPM"
    fi
else
    echo "❌ Ошибка: pnpm не найден, пожалуйста установите pnpm $REQUIRED_PNPM"
    echo "💡 Команда установки: npm install -g pnpm@$REQUIRED_PNPM"
    exit 1
fi

# Проверка версии Go
REQUIRED_GO="1.24.1"
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    echo "🔧 Версия Go: $GO_VERSION (требуется: $REQUIRED_GO)"
    if [ "$GO_VERSION" != "$REQUIRED_GO" ]; then
        echo "⚠️  Предупреждение: Версия Go не соответствует, рекомендуется использовать $REQUIRED_GO"
    fi
else
    echo "❌ Ошибка: Go не найден, пожалуйста установите Go $REQUIRED_GO"
    exit 1
fi

echo "✅ Проверка окружения завершена"
echo ""

# 1. Сборка фронтенда
echo "📦 Сборка фронтенд ресурсов..."
cd ../web
pnpm install
pnpm build
cd ../server

# 2. Копирование фронтенд ресурсов в директорию для встраивания
echo "📋 Копирование фронтенд ресурсов..."
mkdir -p public/dist
cp -r ../web/dist/* public/dist/

# 3. Сборка бэкенда
echo "🔧 Сборка серверной части..."
go mod tidy
go build -o ruiqi-waf .

echo "✅ Сборка завершена!"
echo "📍 Расположение исполняемого файла: server/ruiqi-waf"