#!/bin/bash

# Build Bot Version Script
# Creates timestamped bot versions in bin/ directory
# Usage: 
#   ./build_bot_version.sh          # Build timestamped version + update new_bot
#   ./build_bot_version.sh current  # Build current version (overwrites current_bot only)

if [ ! -d "bin" ]; then
    mkdir bin
fi

# Check if "current" parameter was passed
if [ "$1" = "current" ]; then
    # Build current version (no timestamp)
    BOT_NAME="current_bot"
    BOT_PATH="current_bot"
    
    echo "🤖 Building Current Bot Version"
    echo "==============================="
    echo "Output: ${BOT_PATH}"
    echo ""
    
    # Build the bot
    echo "📦 Compiling current bot..."
    go build -o "${BOT_PATH}" .
    
    if [ $? -eq 0 ]; then
        echo "✅ Current bot built successfully: ${BOT_PATH}"
        chmod +x "${BOT_PATH}"
        
        echo ""
        echo "🎯 Usage examples:"
        echo "  # Test current bot against new_bot"
        echo "  ./bin/real_water_fight_tester ./current_bot ./new_bot ./scenarios/scenario1.txt"
        echo ""
        echo "  # Test current bot against older version"
        echo "  ./bin/real_water_fight_tester ./current_bot ./bin/bot_OLDER_TIMESTAMP ./scenarios/scenario1.txt"
        
    else
        echo "❌ Current bot build failed!"
        exit 1
    fi
    
else
    # Build timestamped version (default behavior)
    # Generate timestamp
    TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
    BOT_NAME="bot_${TIMESTAMP}"
    BOT_PATH="bin/${BOT_NAME}"
    
    echo "🤖 Building Timestamped Bot Version"
    echo "===================================="
    echo "Timestamp: ${TIMESTAMP}"
    echo "Output: ${BOT_PATH}"
    echo ""
    
    # Build the bot
    echo "📦 Compiling bot..."
    go build -o "${BOT_PATH}" .
    
    if [ $? -eq 0 ]; then
        echo "✅ Bot built successfully: ${BOT_PATH}"
        chmod +x "${BOT_PATH}"
        
        # Update new_bot (but keep current_bot unchanged)
        echo "🔗 Updating new_bot..."
        cp "${BOT_PATH}" new_bot
        echo "✅ new_bot updated with latest version"
        
        echo ""
        echo "📋 Available bot versions in bin/:"
        ls -la bin/ | grep "bot_" | awk '{print "  " $9 " (" $5 " bytes, " $6 " " $7 " " $8 ")"}'
        
        echo ""
        echo "🎯 Usage examples:"
        echo "  # Test latest version against current_bot"
        echo "  ./bin/real_water_fight_tester ./current_bot ./new_bot ./scenarios/scenario1.txt"
        echo ""
        echo "  # Test against older version"
        echo "  ./bin/real_water_fight_tester ./bin/bot_OLDER_TIMESTAMP ./new_bot ./scenarios/scenario1.txt"
        echo ""
        echo "💡 Next steps:"
        echo "  1. Test the new_bot version"
        echo "  2. If satisfied, run './build_bot_version.sh current' to promote to current_bot"
        
    else
        echo "❌ Build failed!"
        exit 1
    fi
fi 