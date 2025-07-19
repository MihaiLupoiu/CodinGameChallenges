# CodinGameChallenges
CodinGame Challenge Solutions

## 2025 Summer Challenge - Water Fight Arena

### Challenge Overview
A competitive multiplayer strategy game where players control teams of agents in a water fight arena. The objective is to control territory by positioning agents strategically while eliminating enemies.

**Win Conditions:**
- Achieve 600+ point lead through territory control
- Eliminate all enemy agents  
- Have the most points after 100 turns

### Current Implementation
- **Language**: Go 1.24
- **Strategy**: Team Coordination Strategy (see `2025/Summer/STRATEGY.md`)
- **Focus**: Role-based team coordination with tactical combat

### Key Features
- **Role-based coordination**: Attacker, Bomber, Supporter, Defender roles
- **Fixed collision avoidance**: No more agents converging on same positions
- **Cover-first tactics**: Always seek cover before engaging enemies
- **Smart bomb targeting**: Center bombing for maximum multi-enemy damage
- **Team objectives**: Shared goals and coordinated focus fire
- **Comprehensive testing**: Test suite covering core functions

### Files
- `2025/Summer/main.go` - Main implementation with Enhanced Territory Strategy
- `2025/Summer/task5-final.md` - Complete challenge rules and specifications
- `2025/Summer/STRATEGY.md` - Detailed strategy documentation

### Usage
```bash
cd 2025/Summer
go build -o challenge main.go
# Submit the resulting binary to CodinGame platform
```

### Strategy Philosophy
The Ultra-Aggressive Combat Strategy uses **cover-first, combat-before-movement tactics** with **ultra-strategic bombing**. All agents prioritize shooting from protected positions, and only bomb when hitting multiple enemies OR eliminating 70%+ wetness enemies behind cover. Movement is optimized for cover positioning and tactical advantages. The strategy completely eliminates wasteful bombing through precise targeting conditions.
