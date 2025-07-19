# Ultra-Aggressive Combat Strategy

## Overview

The Ultra-Aggressive Combat Strategy is a **cover-first, combat-focused approach** designed to win through overwhelming tactical aggression. This strategy prioritizes immediate combat opportunities while ensuring agents position themselves near cover for protection. All agents focus on shooting and bombing first, with movement optimized for tactical positioning.

## Key Strategic Principles

### 1. Role-Based Team Coordination
- **Attacker**: Focuses on eliminating enemies and aggressive positioning
- **Bomber**: Specializes in splash bomb opportunities and enemy clusters
- **Supporter**: Concentrates on territory control and team assistance
- **Defender**: Protects wounded allies and maintains defensive positions

### 2. Advanced Collision Avoidance
- **Priority-based movement**: Higher priority actions get first choice of positions
- **Alternative positioning**: Smart fallback positions when conflicts arise
- **Clustering prevention**: Penalties for being too close to teammates

### 3. Cover-First Tactics
- **Always seek cover**: All roles prioritize cover positions before attacking
- **Line of sight positioning**: Attackers find covered positions with enemy visibility
- **Tactical advancement**: Move from cover to cover, not in open areas

### 4. Smart Bombing Strategy
- **Center bombing**: Target the center of enemy clusters for maximum damage
- **Multi-hit priority**: Prefer bomb positions that hit 2+ enemies
- **Coordination**: Bombers position to support team objectives

### 5. Emergency Response System
- **Elimination priority**: Always prioritizes finishing off wounded enemies
- **Threat detection**: Identifies immediate dangers and responds appropriately
- **Team protection**: Defenders prioritize protecting wounded allies

## Key Improvements Over Previous Strategy

### ✅ Ultra-Aggressive Combat Priority
- **Combat before roles**: Shooting/bombing gets checked BEFORE role-specific movement
- **Cover-first shooting**: Only shoot when protected by cover (with unprotected fallback)
- **Universal targeting**: All agents target any enemies in range, no role restrictions

### ✅ Ultra-Strategic Bombing Intelligence  
- **Cluster bombing only**: Only bomb when hitting 2+ enemies simultaneously
- **Elimination bombing**: Bomb 70%+ wetness enemies behind cover that we can't easily reach
- **Zero waste**: Completely eliminates wasteful bombing - every bomb has clear tactical purpose

### ✅ Cover-First Positioning
- **Massive cover bonus**: 200x multiplier for positions adjacent to cover
- **Tactical positioning**: Movement prioritizes cover + shooting opportunities
- **Range optimization**: 150pt bonus for optimal range, 75pt for extended range

### ✅ Smart Bomb Targeting
- **Strategic conditions**: Only bomb when tactically advantageous (2+ enemies OR covered enemy)
- **Cluster detection**: Automatic detection of enemy clusters for multi-hit bombs  
- **Cover breaking**: Bomb enemies behind cover that are hard to reach with direct fire

### ✅ Simplified Role System
- **Roles affect movement only**: Combat decisions (shoot/bomb) are universal
- **Focused behaviors**: Each role has clear movement preferences without combat restrictions
- **Dynamic targeting**: Best target selection regardless of agent role

### ✅ Enhanced Target Selection
- **Priority scoring**: Higher wetness enemies + closer enemies get priority
- **Range flexibility**: Consider both optimal range (50pt bonus) and extended range
- **Elimination focus**: High wetness enemies are heavily prioritized

## Win Conditions Strategy

The strategy targets all three win conditions:

1. **600+ Point Lead**: Primary focus through territory control optimization
2. **Enemy Elimination**: Secondary focus through efficient targeting
3. **Most Points After 100 Turns**: Backup through sustained territorial advantage

## Technical Features

### Territory Evaluation System
- Calculates control for every passable tile on the map
- Accounts for wetness penalties (agents with 50+ wetness have doubled distance)
- Provides real-time territorial advantage/disadvantage metrics

### Safety Analysis
- Multi-threat assessment considering shooting and bomb ranges
- Cover effectiveness calculation
- Emergency escape route identification

### Combat Effectiveness Scoring
- Damage calculation with range and cover penalties
- Elimination probability assessment
- Resource efficiency optimization

## Usage

The strategy is automatically selected in `main.go`:

```go
game.CurrentStrategy = NewTeamCoordinationStrategy()
```

## Testing

Run the test suite to validate key functions:

```bash
go test -v
```

Tests cover:
- Role assignment logic
- Collision avoidance system  
- Territory calculation
- Bomb targeting accuracy
- Cover detection
- Agent clustering prevention

## Testing Recommendations

1. **Territorial Scenarios**: Test against opponents who focus on territory control
2. **Combat Scenarios**: Verify elimination priorities work against aggressive opponents  
3. **Mixed Scenarios**: Ensure balanced decision-making under various conditions
4. **Endgame Scenarios**: Test behavior when ahead/behind in territory control

## Future Enhancements

- **Predictive movement**: Anticipate enemy movements for better positioning
- **Team coordination**: Enhance multi-agent coordination for complex maneuvers
- **Advanced pathfinding**: Implement more sophisticated area denial tactics
- **Adaptive thresholds**: Tune decision parameters based on game state 