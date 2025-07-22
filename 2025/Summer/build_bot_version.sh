#!/bin/bash

# Build Bot Version Script
# Creates timestamped bot versions in bin/ directory

if [ ! -d "bin" ]; then
    mkdir bin
fi

# Generate timestamp
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BOT_NAME="bot_${TIMESTAMP}"
BOT_PATH="bin/${BOT_NAME}"

echo "🤖 Building Bot Version"
echo "======================"
echo "Timestamp: ${TIMESTAMP}"
echo "Output: ${BOT_PATH}"
echo ""

# Build the bot
echo "📦 Compiling bot..."
go build -o "${BOT_PATH}" .

if [ $? -eq 0 ]; then
    echo "✅ Bot built successfully: ${BOT_PATH}"
    chmod +x "${BOT_PATH}"
    
    # Update current_bot and new_bot links
    echo "🔗 Updating current_bot and new_bot..."
    cp "${BOT_PATH}" current_bot
    cp "${BOT_PATH}" new_bot
    
    echo ""
    echo "📋 Available bot versions in bin/:"
    ls -la bin/ | grep "bot_" | awk '{print "  " $9 " (" $5 " bytes, " $6 " " $7 " " $8 ")"}'
    
    echo ""
    echo "🎯 Usage examples:"
    echo "  # Test latest version against itself"
    echo "  ./bin/real_water_fight_tester ./current_bot ./new_bot ./scenario1.txt"
    echo ""
    echo "  # Test against older version"
    echo "  ./bin/real_water_fight_tester ./bin/bot_OLDER_TIMESTAMP ./current_bot ./scenario1.txt"
    
else
    echo "❌ Build failed!"
    exit 1
fi 