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
- **Strategy**: Enhanced Territory Strategy (see `2025/Summer/STRATEGY.md`)
- **Focus**: Territory control with intelligent combat tactics

### Key Features
- Real-time territory evaluation system
- Emergency response for threats and elimination opportunities
- Multi-factor target selection (elimination potential, territory impact, cover effectiveness)
- Strategic movement for optimal positioning
- Smart resource management for splash bombs and shooting

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
The Enhanced Territory Strategy balances aggressive combat with strategic positioning to maintain territorial advantage. It prioritizes elimination opportunities while ensuring optimal map control for sustained point generation.
