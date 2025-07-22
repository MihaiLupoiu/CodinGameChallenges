#!/bin/bash

echo "💧 Real Water Fight Testing System 💧"
echo "======================================"
echo ""

# Build real water fight tester
echo "📦 Building real water fight tester..."
cd simple_tester
go build -o ../bin/real_water_fight_tester real_game_tester.go
cd ..
echo "✅ Real water fight tester ready: bin/real_water_fight_tester"

# Build versioned bot
echo "📦 Building versioned bot..."
./build_bot_version.sh

echo ""
echo "🎯 READY TO TEST!"
echo ""
echo "🛠️  Available Commands:"
echo "  ./build_bot_version.sh                  # Build new timestamped version + update new_bot"
echo "  ./build_bot_version.sh current          # Build as current_bot (promote new changes)"
echo "  ./bin/real_water_fight_tester bot1 bot2 scenario.txt  # Real water fight battle"
echo ""
echo "📋 Available scenarios:"
ls -1 *scenario*.txt 2>/dev/null | sed 's/^/  /'
echo ""
echo "💡 RECOMMENDED WORKFLOW:"
echo "  1. Make changes to main.go"
echo "  2. Run ./build_bot_version.sh to create new timestamped version"
echo "  3. Test: ./bin/real_water_fight_tester ./current_bot ./new_bot ./scenarios/scenario1.txt"
echo "  4. If satisfied, run ./build_bot_version.sh current to promote to current_bot"
echo ""
echo "🚀 FEATURES:"
echo "  💥 Real shooting mechanics (soaking power, range, cover)"
echo "  🧨 Splash bombs (30 damage, 4 tile range, 3x3 splash)"
echo "  🛡️  Cover system (50% low, 75% high protection)"
echo "  🏆 Territory control scoring"
echo "  🎯 Real win conditions (600 point lead, elimination, 100 turns)" 