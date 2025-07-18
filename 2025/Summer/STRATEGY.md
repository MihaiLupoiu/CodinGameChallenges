# Enhanced Territory Strategy

## Overview

The Enhanced Territory Strategy is designed to win the water fight challenge by balancing **territory control** with **combat effectiveness**. Unlike the previous strategy that focused primarily on combat, this approach prioritizes the key win condition: controlling more territory to achieve a 600+ point advantage.

## Key Strategic Principles

### 1. Territory-First Approach
- **Real-time territory evaluation**: Constantly monitors which tiles are controlled by friendly vs enemy agents
- **Strategic positioning**: Moves agents to positions that maximize territory control
- **Territory impact bombing**: Considers how eliminating enemies affects overall map control

### 2. Emergency Response System
- **Elimination priority**: Always prioritizes actions that can eliminate enemies (splash bombs or shooting)
- **Survival instincts**: Detects immediate threats and moves agents to safety
- **Resource efficiency**: Uses splash bombs and shots optimally for maximum impact

### 3. Multi-layered Decision Making

The strategy evaluates actions in this priority order:

1. **Emergency Actions** (Highest Priority)
   - Elimination opportunities (bombs/shots that kill enemies)
   - Escape from immediate danger
   
2. **Strategic Bombing** 
   - High-value splash bomb targets
   - Territory impact consideration
   - Area denial tactics
   
3. **Strategic Shooting**
   - Target selection based on elimination potential
   - Territory gain from eliminating specific enemies
   - Cover and distance effectiveness
   
4. **Strategic Movement**
   - Territory control optimization
   - Combat positioning
   - Safety considerations
   
5. **Defensive Actions**
   - Hunker down when under threat or no better options

## Key Improvements Over Previous Strategy

### Better Target Selection
- **Elimination-focused**: Prioritizes targets that can be eliminated this turn
- **Territory-weighted**: Considers how eliminating each enemy affects map control
- **Multi-factor scoring**: Balances damage, distance, cover, and strategic value

### Enhanced Positioning
- **Territory value calculation**: Evaluates how much territory each position could control
- **Combat effectiveness**: Considers shooting/bombing opportunities from each position
- **Safety assessment**: Accounts for enemy threats and available cover

### Smart Resource Management
- **Splash bomb optimization**: Uses bombs for maximum territory impact, not just damage
- **Elimination priority**: Focuses resources on finishing off wounded enemies
- **Area denial**: Uses bombs to block enemy movement when beneficial

### Real-time Adaptation
- **Dynamic territory monitoring**: Adjusts strategy based on current map control
- **Threat assessment**: Responds to immediate dangers with emergency actions
- **Opportunity recognition**: Quickly identifies and acts on elimination chances

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
game.CurrentStrategy = &EnhancedTerritoryStrategy{}
```

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