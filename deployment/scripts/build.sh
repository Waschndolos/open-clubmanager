#!/bin/bash
set -e
cd "$(dirname "$0")"

echo "🔧 Building frontend..."
cd ../../frontend
rm -rf dist
npm install
npm run build

echo "🔧 Building backend..."
cd ../backend
npm install

# 🧹 Clean old dist
rm -rf dist

# 🧪 Create dummy DB if not present
if [ ! -f ./clubmanager.db ]; then
  echo "📄 Creating dummy DB for Prisma..."
  touch ./clubmanager.db
fi

# 🌱 Generate Prisma client
export DATABASE_URL="file:./clubmanager.db"
npx prisma generate

# 🛠 Build with tsc
npm run build

echo "📦 Creating binary with pkg..."
pkg dist/server.js --config package.pkg.json --out-path ../deployment/release

echo "✅ Done!"
