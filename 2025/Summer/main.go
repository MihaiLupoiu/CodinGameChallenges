package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Debug configuration
const DEBUG_PRINT_AGENTS = true // Set to false to disable agent location printing

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1000000), 1000000)

	// Initialize game state
	game := NewGame()

	// myId: Your player id (0 or 1)
	scanner.Scan()
	fmt.Sscan(scanner.Text(), &game.MyID)

	// agentCount: Total number of agents in the game
	var agentCount int
	scanner.Scan()
	fmt.Sscan(scanner.Text(), &agentCount)

	for i := 0; i < agentCount; i++ {
		// agentId: Unique identifier for this agent
		// player: Player id of this agent
		// shootCooldown: Number of turns between each of this agent's shots
		// optimalRange: Maximum manhattan distance for greatest damage output
		// soakingPower: Damage output within optimal conditions
		// splashBombs: Number of splash bombs this can throw this game
		var agentId, player, shootCooldown, optimalRange, soakingPower, splashBombs int
		scanner.Scan()
		fmt.Sscan(scanner.Text(), &agentId, &player, &shootCooldown, &optimalRange, &soakingPower, &splashBombs)

		// Store agent data
		agent := &Agent{
			ID:             agentId,
			Player:         player,
			ShootCooldown:  shootCooldown,
			OptimalRange:   optimalRange,
			SoakingPower:   soakingPower,
			MaxSplashBombs: splashBombs,
		}

		game.Agents[agentId] = agent
		if player == game.MyID {
			game.MyAgents = append(game.MyAgents, agent)
		}
	}

	// width: Width of the game map
	// height: Height of the game map
	scanner.Scan()
	fmt.Sscan(scanner.Text(), &game.Width, &game.Height)

	// Initialize and read grid
	game.Grid = make([][]Tile, game.Height)
	for i := 0; i < game.Height; i++ {
		game.Grid[i] = make([]Tile, game.Width)
	}

	for i := 0; i < game.Height; i++ {
		scanner.Scan()
		inputs := strings.Split(scanner.Text(), " ")
		for j := 0; j < game.Width; j++ {
			// x: X coordinate, 0 is left edge
			// y: Y coordinate, 0 is top edge
			x, _ := strconv.ParseInt(inputs[3*j], 10, 32)
			y, _ := strconv.ParseInt(inputs[3*j+1], 10, 32)
			tileType, _ := strconv.ParseInt(inputs[3*j+2], 10, 32)

			game.Grid[i][j] = Tile{
				X:    int(x),
				Y:    int(y),
				Type: int(tileType),
			}
		}
	}

	// Print the loaded map for easy context sharing
	game.PrintMap()

	game.CurrentStrategy = &EnhancedTerritoryStrategy{}
	fmt.Fprintln(os.Stderr, "Starting with strategy:", game.CurrentStrategy.Name())

	firstTurn := true // Flag to print agent locations on first turn

	for {
		scanner.Scan()
		fmt.Sscan(scanner.Text(), &agentCount)

		// Clear current agent list - only keep agents that exist this turn
		currentAgents := make(map[int]*Agent)
		game.MyAgents = make([]*Agent, 0)

		for i := 0; i < agentCount; i++ {
			// cooldown: Number of turns before this agent can shoot
			// wetness: Damage (0-100) this agent has taken
			var agentId, x, y, cooldown, splashBombs, wetness int
			scanner.Scan()
			fmt.Sscan(scanner.Text(), &agentId, &x, &y, &cooldown, &splashBombs, &wetness)

			// Get agent from previous turn (to keep static properties) or skip if not found
			if existingAgent, exists := game.Agents[agentId]; exists {
				// Update dynamic properties
				existingAgent.X = x
				existingAgent.Y = y
				existingAgent.Cooldown = cooldown
				existingAgent.SplashBombs = splashBombs
				existingAgent.Wetness = wetness

				// Add to current agents (only living agents appear in input)
				currentAgents[agentId] = existingAgent

				// Track our agents
				if existingAgent.Player == game.MyID {
					game.MyAgents = append(game.MyAgents, existingAgent)
				}
			}
		}

		// Replace agent list with current living agents only
		game.Agents = currentAgents

		// Clear target cache for new turn
		game.TargetCached = false
		game.CurrentTarget = nil

		fmt.Fprintln(os.Stderr, fmt.Sprintf("Turn update: %d total agents, %d mine",
			len(game.Agents), len(game.MyAgents)))

		// Print map with agents on first turn if debug enabled
		if firstTurn && DEBUG_PRINT_AGENTS {
			game.PrintMapAndAgents()
			firstTurn = false
		}

		// myAgentCount: Number of alive agents controlled by you
		var myAgentCount int
		scanner.Scan()
		fmt.Sscan(scanner.Text(), &myAgentCount)

		// Coordinate all agent actions using current strategy
		actions := game.CoordinateActions()

		for _, agent := range game.MyAgents {
			agentActions := actions[agent.ID]
			actionStr := game.FormatAction(agentActions)

			// Log all actions for this agent
			reasons := []string{}
			for _, action := range agentActions {
				if action.Reason != "" {
					reasons = append(reasons, action.Reason)
				}
			}
			reasonStr := strings.Join(reasons, ", ")

			log := fmt.Sprintf("Agent %d: %s (Reasons: %s)", agent.ID, actionStr, reasonStr)
			fmt.Fprintln(os.Stderr, log)

			// One line per agent: <agentId>;<action1;action2;...> actions are "MOVE x y | SHOOT id | THROW x y | HUNKER_DOWN | MESSAGE text"
			fmt.Printf("%d; %s\n", agent.ID, actionStr)
		}
	}
}

// Helper function
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Tile represents a single grid tile
type Tile struct {
	X, Y int
	Type int
}

// Agent represents an agent with all its properties
type Agent struct {
	ID          int
	Player      int
	X, Y        int
	Cooldown    int
	SplashBombs int
	Wetness     int
	// Static properties from initialization
	ShootCooldown  int
	OptimalRange   int
	SoakingPower   int
	MaxSplashBombs int
	// Target coordinates for this challenge
	TargetX, TargetY int
}

// Game holds the entire game state
type Game struct {
	MyID     int
	Grid     [][]Tile
	Width    int
	Height   int
	Agents   map[int]*Agent
	MyAgents []*Agent
	// Current strategy for coordinating actions
	CurrentStrategy Strategy
	// Target tracking for dynamic switching
	PreviousTargetID int
	// Current turn's target (cached to avoid multiple lookups)
	CurrentTarget *Agent
	TargetCached  bool
}

// NewGame creates a new game instance
func NewGame() *Game {
	return &Game{
		Agents:   make(map[int]*Agent),
		MyAgents: make([]*Agent, 0),
	}
}

// CoordinateActions coordinates the actions of all agents using the current strategy
func (g *Game) CoordinateActions() map[int][]AgentAction {
	allActions := make(map[int][]AgentAction)

	// Step 1: Each agent evaluates all actions they want to perform
	for _, agent := range g.MyAgents {
		actions := g.CurrentStrategy.EvaluateActions(agent, g)
		allActions[agent.ID] = actions
		log := fmt.Sprintf("Agent %d generated %d actions", agent.ID, len(actions))
		fmt.Fprintln(os.Stderr, log)
	}

	// Step 2: Sort actions by priority and resolve conflicts for movement
	finalActions := g.resolveActionConflicts(allActions)

	return finalActions
}

// FormatAction formats multiple actions into a string for output (separated by ;)
func (g *Game) FormatAction(actions []AgentAction) string {
	if len(actions) == 0 {
		return "HUNKER_DOWN"
	}

	// Sort actions by priority (highest first)
	sortedActions := make([]AgentAction, len(actions))
	copy(sortedActions, actions)
	for i := 0; i < len(sortedActions); i++ {
		for j := i + 1; j < len(sortedActions); j++ {
			if sortedActions[i].Priority < sortedActions[j].Priority {
				sortedActions[i], sortedActions[j] = sortedActions[j], sortedActions[i]
			}
		}
	}

	parts := []string{}
	for _, action := range sortedActions {
		switch action.Type {
		case ActionMove:
			parts = append(parts, fmt.Sprintf("MOVE %d %d", action.TargetX, action.TargetY))
		case ActionShoot:
			parts = append(parts, fmt.Sprintf("SHOOT %d", action.TargetAgentID))
		case ActionThrow:
			parts = append(parts, fmt.Sprintf("THROW %d %d", action.TargetX, action.TargetY))
		case ActionHunker:
			parts = append(parts, "HUNKER_DOWN")
		case ActionMessage:
			parts = append(parts, fmt.Sprintf("MESSAGE %s", action.Message))
		}
	}

	if len(parts) == 0 {
		return "HUNKER_DOWN"
	}

	return strings.Join(parts, "; ")
}

// Action types and priorities
type ActionType int

const (
	ActionMove ActionType = iota
	ActionShoot
	ActionThrow
	ActionHunker
	ActionMessage
)

type AgentAction struct {
	Type             ActionType
	TargetX, TargetY int    // For MOVE/THROW
	TargetAgentID    int    // For SHOOT
	Message          string // For MESSAGE
	Priority         int    // Higher = more important
	Reason           string // For debugging
}

// Decision priorities (higher number = higher priority)
const (
	PriorityEmergency = 100 // Avoid death
	PriorityCombat    = 50  // Shooting
	PriorityMovement  = 50  // Positioning
	PriorityDefault   = 10  // Hunker down
)

// Strategy interface for different AI behaviors
type Strategy interface {
	EvaluateActions(agent *Agent, game *Game) []AgentAction
	Name() string
}

// PositionStrategy - Phase 1: Get into tactical position near enemy clusters
type PositionStrategy struct{}

func (s *PositionStrategy) Name() string {
	return "Position"
}

func (s *PositionStrategy) EvaluateActions(agent *Agent, game *Game) []AgentAction {
	actions := []AgentAction{}

	// Find tactical position near largest enemy cluster
	targetX, targetY := game.FindTacticalPosition(agent)

	if targetX != agent.X || targetY != agent.Y {
		actions = append(actions, AgentAction{
			Type:     ActionMove,
			TargetX:  targetX,
			TargetY:  targetY,
			Priority: PriorityMovement,
			Reason:   fmt.Sprintf("Moving to tactical position (%d,%d) near enemy cluster", targetX, targetY),
		})
	}

	fmt.Fprintln(os.Stderr, fmt.Sprintf("Agent %d positioning: generated %d actions", agent.ID, len(actions)))
	return actions
}

// CombatStrategy - Phase 2: Deal maximum damage from good position
type CombatStrategy struct{}

func (s *CombatStrategy) Name() string {
	return "Combat"
}

func (s *CombatStrategy) EvaluateActions(agent *Agent, game *Game) []AgentAction {
	actions := []AgentAction{}

	// Priority 1: Use bombs if we have good cluster targets
	if agent.SplashBombs > 0 {
		bombX, bombY, expectedDamage := game.FindOptimalSplashBombTarget(agent)
		if expectedDamage >= 150.0 { // Higher threshold for combat phase
			actions = append(actions, AgentAction{
				Type:     ActionThrow,
				TargetX:  bombX,
				TargetY:  bombY,
				Priority: PriorityCombat,
				Reason:   fmt.Sprintf("Combat bombing at (%d,%d) - damage: %.0f", bombX, bombY, expectedDamage),
			})
		}
	}

	// Priority 2: Shoot highest wetness enemy if no good bomb targets
	if len(actions) == 0 && agent.Cooldown == 0 {
		target := game.FindHighestWetnessEnemyInRange(agent)
		if target != nil {
			actions = append(actions, AgentAction{
				Type:          ActionShoot,
				TargetAgentID: target.ID,
				Priority:      PriorityCombat,
				Reason:        fmt.Sprintf("Combat shooting highest wetness enemy %d (wetness: %d)", target.ID, target.Wetness),
			})
		}
	}

	fmt.Fprintln(os.Stderr, fmt.Sprintf("Agent %d combat: generated %d actions", agent.ID, len(actions)))
	return actions
}

// TakeCoverAndShootBomb strategy - now uses two-phase approach
type TakeCoverAndShootBombStrategy struct{}

func (s *TakeCoverAndShootBombStrategy) Name() string {
	return "TakeCoverAndShootBomb"
}

func (s *TakeCoverAndShootBombStrategy) EvaluateActions(agent *Agent, game *Game) []AgentAction {
	// Determine which phase we're in
	if game.ShouldSwitchToCombat(agent) {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Agent %d in COMBAT phase", agent.ID))
		combatStrategy := &CombatStrategy{}
		return combatStrategy.EvaluateActions(agent, game)
	} else {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Agent %d in POSITION phase", agent.ID))
		positionStrategy := &PositionStrategy{}
		return positionStrategy.EvaluateActions(agent, game)
	}
}

// Enhanced strategy that balances territory control with combat effectiveness
type EnhancedTerritoryStrategy struct{}

func (s *EnhancedTerritoryStrategy) Name() string {
	return "EnhancedTerritory"
}

func (s *EnhancedTerritoryStrategy) EvaluateActions(agent *Agent, game *Game) []AgentAction {
	actions := []AgentAction{}

	// Step 1: Evaluate current territorial situation
	territoryScore := game.EvaluateTerritoryControl()

	// Step 2: Check for immediate threats or elimination opportunities
	emergencyAction := game.EvaluateEmergencyActions(agent)
	if emergencyAction.Type != -1 {
		actions = append(actions, emergencyAction)
		return actions
	}

	// Step 3: Look for high-value splash bomb opportunities
	if agent.SplashBombs > 0 {
		bombAction := game.EvaluateStrategicBombing(agent, territoryScore)
		if bombAction.Type != -1 {
			actions = append(actions, bombAction)
			// Still evaluate movement for positioning
		}
	}

	// Step 4: Evaluate shooting opportunities
	if len(actions) == 0 && agent.Cooldown == 0 {
		shootAction := game.EvaluateStrategicShooting(agent, territoryScore)
		if shootAction.Type != -1 {
			actions = append(actions, shootAction)
		}
	}

	// Step 5: Evaluate movement for territory control and positioning
	moveAction := game.EvaluateStrategicMovement(agent, territoryScore)
	if moveAction.Type != -1 {
		actions = append(actions, moveAction)
	}

	// Step 6: Defensive actions if needed
	if len(actions) == 0 || game.IsUnderThreat(agent) {
		hunkerAction := AgentAction{
			Type:     ActionHunker,
			Priority: PriorityDefault,
			Reason:   "Defensive positioning or no better actions available",
		}
		actions = append(actions, hunkerAction)
	}

	fmt.Fprintln(os.Stderr, fmt.Sprintf("Agent %d enhanced strategy: generated %d actions", agent.ID, len(actions)))
	return actions
}

// FindOptimalSplashBombTarget finds the best position to throw a splash bomb for maximum damage
func (g *Game) FindOptimalSplashBombTarget(agent *Agent) (int, int, float64) {
	bestX, bestY := agent.X, agent.Y
	maxDamage := 0.0

	// Check all positions within splash bomb range (4 tiles)
	for targetY := 0; targetY < g.Height; targetY++ {
		for targetX := 0; targetX < g.Width; targetX++ {
			// Check if position is within throwing range
			distance := abs(agent.X-targetX) + abs(agent.Y-targetY)
			if distance > 4 {
				continue
			}

			// Check for friendly fire first
			if g.WouldHitFriendlyAgents(targetX, targetY) {
				continue
			}

			// Calculate total damage potential at this position
			totalDamage := g.CalculateSplashDamageScore(targetX, targetY)

			if totalDamage > maxDamage {
				bestX, bestY = targetX, targetY
				maxDamage = totalDamage
			}
		}
	}

	fmt.Fprintln(os.Stderr, fmt.Sprintf("Agent %d optimal splash bomb: (%d,%d) total damage score: %.1f",
		agent.ID, bestX, bestY, maxDamage))

	return bestX, bestY, maxDamage
}

// ShouldSwitchToCombat determines if agent should switch from positioning to combat phase
func (g *Game) ShouldSwitchToCombat(agent *Agent) bool {
	// Switch to combat if:
	// 1. Agent has cover AND enemies in bomb/shoot range, OR
	// 2. No better tactical position available, OR
	// 3. Agent is under threat (enemies very close), OR
	// 4. Agent has been positioning for too long

	currentCover := g.GetMaxAdjacentCover(agent.X, agent.Y)
	hasGoodPosition := currentCover > 0

	// Count enemies within bomb range (4 tiles)
	enemiesInBombRange := 0
	enemiesInShootRange := 0
	closestEnemyDistance := 999

	for _, enemy := range g.Agents {
		if enemy.Player != g.MyID && enemy.Wetness < 100 {
			distance := abs(agent.X-enemy.X) + abs(agent.Y-enemy.Y)
			if distance <= 4 {
				enemiesInBombRange++
			}
			if distance <= agent.OptimalRange {
				enemiesInShootRange++
			}
			if distance < closestEnemyDistance {
				closestEnemyDistance = distance
			}
		}
	}

	// Switch to combat if in good position with targets
	if hasGoodPosition && (enemiesInBombRange >= 2 || enemiesInShootRange >= 1) {
		return true
	}

	// Switch to combat if under immediate threat (enemy within 3 tiles)
	if closestEnemyDistance <= 3 {
		return true
	}

	// Switch to combat if we have ANY targets in range (don't wait forever)
	if enemiesInShootRange >= 1 {
		return true
	}

	// Otherwise stay in positioning phase
	return false
}

// FindTacticalPosition finds the best position near enemy clusters with cover
func (g *Game) FindTacticalPosition(agent *Agent) (int, int) {
	bestX, bestY := agent.X, agent.Y
	bestScore := -999.0

	// Find the best enemy cluster for this specific agent (for coordination)
	clusterX, clusterY, clusterSize := g.FindBestClusterForAgent(agent)

	if clusterSize == 0 {
		// No enemies found, just find any cover
		return g.FindBestCoverPosition(agent)
	}

	fmt.Fprintln(os.Stderr, fmt.Sprintf("Agent %d targeting cluster: (%d,%d) with %d enemies", agent.ID, clusterX, clusterY, clusterSize))

	// Try multiple approaches in order of preference

	// Approach 1: Look for cover positions within bomb range (4 tiles) of the cluster
	maxSearchDistance := 5 // Increased from 3
	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			// Skip impassable tiles
			if g.Grid[y][x].Type > 0 {
				continue
			}

			// Check if reachable
			distanceToAgent := abs(agent.X-x) + abs(agent.Y-y)
			if distanceToAgent > maxSearchDistance {
				continue
			}

			// Check if within bomb range of cluster
			distanceToCluster := abs(x-clusterX) + abs(y-clusterY)
			if distanceToCluster > 4 {
				continue
			}

			// Score this position
			score := 0.0

			// Bonus for cover
			coverLevel := g.GetMaxAdjacentCover(x, y)
			score += float64(coverLevel) * 20.0

			// Bonus for being closer to cluster (better bomb angles)
			score += (4.0 - float64(distanceToCluster)) * 10.0

			// Small penalty for distance from agent
			score -= float64(distanceToAgent) * 0.5

			if score > bestScore {
				bestX, bestY = x, y
				bestScore = score
			}
		}
	}

	// Approach 2: If no ideal position found, move toward cluster (even without perfect cover)
	if bestScore <= -999.0 {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("No ideal tactical position found, moving toward cluster"))

		// Find the best position to move toward the cluster
		for y := 0; y < g.Height; y++ {
			for x := 0; x < g.Width; x++ {
				// Skip impassable tiles
				if g.Grid[y][x].Type > 0 {
					continue
				}

				// Check if reachable in 1-2 moves
				distanceToAgent := abs(agent.X-x) + abs(agent.Y-y)
				if distanceToAgent > 2 {
					continue
				}

				// Score based on getting closer to cluster
				distanceToCluster := abs(x-clusterX) + abs(y-clusterY)
				currentDistanceToCluster := abs(agent.X-clusterX) + abs(agent.Y-clusterY)

				// Only consider positions that get us closer
				if distanceToCluster >= currentDistanceToCluster {
					continue
				}

				score := 0.0

				// Bonus for getting closer to cluster
				score += float64(currentDistanceToCluster-distanceToCluster) * 20.0

				// Bonus for any cover
				coverLevel := g.GetMaxAdjacentCover(x, y)
				score += float64(coverLevel) * 10.0

				// Small penalty for distance from agent
				score -= float64(distanceToAgent) * 1.0

				if score > bestScore {
					bestX, bestY = x, y
					bestScore = score
				}
			}
		}
	}

	// Approach 3: If still no good position, find any cover
	if bestScore <= -999.0 {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("No path to cluster found, finding any cover"))
		return g.FindBestCoverPosition(agent)
	}

	fmt.Fprintln(os.Stderr, fmt.Sprintf("Agent %d tactical position: (%d,%d) score: %.1f", agent.ID, bestX, bestY, bestScore))
	return bestX, bestY
}

// FindLargestEnemyCluster finds the center of the largest enemy cluster
func (g *Game) FindLargestEnemyCluster() (int, int, int) {
	bestX, bestY, maxEnemies := 0, 0, 0

	// Scan the map in 5x5 windows to find enemy clusters
	for centerY := 2; centerY < g.Height-2; centerY++ {
		for centerX := 2; centerX < g.Width-2; centerX++ {
			enemyCount := 0

			// Count enemies in 5x5 area around this center
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					checkX, checkY := centerX+dx, centerY+dy
					if !g.IsValidPosition(checkX, checkY) {
						continue
					}

					// Check if any enemy is at this position
					for _, enemy := range g.Agents {
						if enemy.Player != g.MyID && enemy.Wetness < 100 &&
							enemy.X == checkX && enemy.Y == checkY {
							enemyCount++
						}
					}
				}
			}

			if enemyCount > maxEnemies {
				bestX, bestY = centerX, centerY
				maxEnemies = enemyCount
			}
		}
	}

	return bestX, bestY, maxEnemies
}

// FindBestClusterForAgent finds the best enemy cluster for a specific agent (for coordination)
func (g *Game) FindBestClusterForAgent(agent *Agent) (int, int, int) {
	bestX, bestY, maxScore := 0, 0, 0

	// Scan the map in 5x5 windows to find enemy clusters
	for centerY := 2; centerY < g.Height-2; centerY++ {
		for centerX := 2; centerX < g.Width-2; centerX++ {
			enemyCount := 0

			// Count enemies in 5x5 area around this center
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					checkX, checkY := centerX+dx, centerY+dy
					if !g.IsValidPosition(checkX, checkY) {
						continue
					}

					// Check if any enemy is at this position
					for _, enemy := range g.Agents {
						if enemy.Player != g.MyID && enemy.Wetness < 100 &&
							enemy.X == checkX && enemy.Y == checkY {
							enemyCount++
						}
					}
				}
			}

			if enemyCount >= 2 { // Only consider clusters with 2+ enemies
				// Score based on enemy count and distance to agent
				distanceToAgent := abs(agent.X-centerX) + abs(agent.Y-centerY)
				score := enemyCount*10 - distanceToAgent // Prefer closer clusters

				if score > maxScore {
					bestX, bestY = centerX, centerY
					maxScore = score
				}
			}
		}
	}

	// If no good cluster found for this agent, fall back to largest cluster
	if maxScore == 0 {
		return g.FindLargestEnemyCluster()
	}

	return bestX, bestY, maxScore / 10 // Convert score back to enemy count approximation
}

// FindHighestWetnessEnemyInRange finds the enemy with highest wetness within agent's range
func (g *Game) FindHighestWetnessEnemyInRange(agent *Agent) *Agent {
	var bestTarget *Agent
	highestWetness := -1

	for _, enemy := range g.Agents {
		if enemy.Player != g.MyID && enemy.Wetness < 100 {
			distance := abs(agent.X-enemy.X) + abs(agent.Y-enemy.Y)

			// Only consider enemies within optimal range
			if distance > agent.OptimalRange {
				continue
			}

			// Prefer higher wetness (closer to elimination)
			if enemy.Wetness > highestWetness {
				bestTarget = enemy
				highestWetness = enemy.Wetness
			}
		}
	}

	if bestTarget != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Highest wetness target in range: Enemy %d (wetness: %d)", bestTarget.ID, bestTarget.Wetness))
	}

	return bestTarget
}

// CalculateSplashDamageScore calculates the total damage score for a splash bomb at given position
func (g *Game) CalculateSplashDamageScore(bombX, bombY int) float64 {
	totalScore := 0.0

	// Check the bomb tile and all 8 adjacent tiles (3x3 area)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			checkX, checkY := bombX+dx, bombY+dy

			if !g.IsValidPosition(checkX, checkY) {
				continue
			}

			// Check if any enemy agent is at this position
			for _, enemy := range g.Agents {
				if enemy.Player != g.MyID && enemy.Wetness < 100 && enemy.X == checkX && enemy.Y == checkY {
					// Base damage score
					damageScore := 30.0

					// Bonus for enemies with higher wetness (closer to elimination)
					wetnessBonus := float64(enemy.Wetness) * 0.5 // 0.5 point per wetness

					// Extra bonus if this would eliminate the enemy
					if enemy.Wetness+30 >= 100 {
						damageScore += 50.0 // Elimination bonus
					}

					totalScore += damageScore + wetnessBonus

					// Only log if we're actually calculating a potential bomb target
					// fmt.Fprintln(os.Stderr, fmt.Sprintf("Enemy %d at (%d,%d): wetness %d, score %.1f",
					//	enemy.ID, enemy.X, enemy.Y, enemy.Wetness, damageScore+wetnessBonus))
				}
			}
		}
	}

	return totalScore
}

// FindEnemyClusterCenters finds potential cluster centers and their coverage scores
func (g *Game) FindEnemyClusterCenters() [][]int {
	clusterCenters := [][]int{}

	// Get all enemy positions
	enemyPositions := [][]int{}
	for _, enemy := range g.Agents {
		if enemy.Player != g.MyID && enemy.Wetness < 100 {
			enemyPositions = append(enemyPositions, []int{enemy.X, enemy.Y})
		}
	}

	// For each potential center position, count nearby enemies
	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			nearbyEnemies := 0

			// Count enemies within 3x3 area around this position
			for _, pos := range enemyPositions {
				if abs(x-pos[0]) <= 1 && abs(y-pos[1]) <= 1 {
					nearbyEnemies++
				}
			}

			// If this position covers 2+ enemies, it's a potential cluster center
			if nearbyEnemies >= 2 {
				clusterCenters = append(clusterCenters, []int{x, y, nearbyEnemies})
				fmt.Fprintln(os.Stderr, fmt.Sprintf("Cluster center candidate: (%d,%d) covers %d enemies",
					x, y, nearbyEnemies))
			}
		}
	}

	return clusterCenters
}

// CountEnemyHitsAtPosition counts how many enemies would be hit by a splash bomb at given position
func (g *Game) CountEnemyHitsAtPosition(bombX, bombY int) int {
	hits := 0

	// Check the bomb tile and all 8 adjacent tiles (3x3 area)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			checkX, checkY := bombX+dx, bombY+dy

			if !g.IsValidPosition(checkX, checkY) {
				continue
			}

			// Check if any enemy agent is at this position
			for _, enemy := range g.Agents {
				if enemy.Player != g.MyID && enemy.Wetness < 100 && enemy.X == checkX && enemy.Y == checkY {
					hits++
					fmt.Fprintln(os.Stderr, fmt.Sprintf("Enemy %d at (%d,%d) would be hit by bomb at (%d,%d)",
						enemy.ID, enemy.X, enemy.Y, bombX, bombY))
				}
			}
		}
	}

	return hits
}

// WouldHitFriendlyAgents checks if a splash bomb would hit any of our agents
func (g *Game) WouldHitFriendlyAgents(bombX, bombY int) bool {
	// Check the bomb tile and all 8 adjacent tiles (3x3 area)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			checkX, checkY := bombX+dx, bombY+dy

			if !g.IsValidPosition(checkX, checkY) {
				continue
			}

			// Check if any of our agents is at this position
			for _, friendly := range g.MyAgents {
				if friendly.X == checkX && friendly.Y == checkY {
					return true // Would hit friendly agent
				}
			}
		}
	}

	return false // Safe to throw
}

// PrintMap prints just the map layout for context sharing
func (g *Game) PrintMap() {
	fmt.Fprintln(os.Stderr, "=== MAP LAYOUT ===")
	fmt.Fprintln(os.Stderr, fmt.Sprintf("Size: %d×%d", g.Width, g.Height))
	fmt.Fprintln(os.Stderr, "Tile types: 0=empty, 1=low cover, 2=high cover")
	fmt.Fprintln(os.Stderr, "")

	// Print column headers
	header := "   "
	for x := 0; x < g.Width; x++ {
		header += fmt.Sprintf("%2d", x)
	}
	fmt.Fprintln(os.Stderr, header)

	// Print each row
	for y := 0; y < g.Height; y++ {
		row := fmt.Sprintf("%2d ", y)
		for x := 0; x < g.Width; x++ {
			tileType := g.Grid[y][x].Type
			switch tileType {
			case 0:
				row += " ." // Empty tile
			case 1:
				row += " ▒" // Low cover
			case 2:
				row += " █" // High cover
			default:
				row += fmt.Sprintf(" %d", tileType)
			}
		}
		fmt.Fprintln(os.Stderr, row)
	}

	fmt.Fprintln(os.Stderr, "==================")
}

// PrintMapAndAgents prints the map with agent positions for game evolution tracking
func (g *Game) PrintMapAndAgents() {
	fmt.Fprintln(os.Stderr, "=== MAP + AGENTS ===")
	fmt.Fprintln(os.Stderr, fmt.Sprintf("Size: %d×%d", g.Width, g.Height))
	fmt.Fprintln(os.Stderr, "Legend: .=empty ▒=low cover █=high cover F=friend E=enemy")
	fmt.Fprintln(os.Stderr, "")

	// Print column headers
	header := "   "
	for x := 0; x < g.Width; x++ {
		header += fmt.Sprintf("%2d", x)
	}
	fmt.Fprintln(os.Stderr, header)

	// Print each row with agents overlaid
	for y := 0; y < g.Height; y++ {
		row := fmt.Sprintf("%2d ", y)
		for x := 0; x < g.Width; x++ {
			// Check if any agent is at this position
			agentHere := ""
			for _, agent := range g.Agents {
				if agent.X == x && agent.Y == y {
					if agent.Player == g.MyID {
						agentHere = fmt.Sprintf("F%d", agent.ID) // Friend
					} else {
						agentHere = fmt.Sprintf("E%d", agent.ID) // Enemy
					}
					break
				}
			}

			if agentHere != "" {
				row += fmt.Sprintf("%3s", agentHere) // Agent position
			} else {
				// Show tile type
				tileType := g.Grid[y][x].Type
				switch tileType {
				case 0:
					row += " . " // Empty tile
				case 1:
					row += " ▒ " // Low cover
				case 2:
					row += " █ " // High cover
				default:
					row += fmt.Sprintf(" %d ", tileType)
				}
			}
		}
		fmt.Fprintln(os.Stderr, row)
	}

	fmt.Fprintln(os.Stderr, "")
	g.PrintAgentLocations()
}

// FindBestCoverPosition finds the best cover position for an agent considering enemy positions
func (g *Game) FindBestCoverPosition(agent *Agent) (int, int) {
	bestX, bestY := agent.X, agent.Y
	bestScore := -1.0

	// Get all enemy positions for threat analysis
	enemies := g.GetEnemyPositions()

	// If agent is in open area with no cover, force them to move towards cover structures
	currentCover := g.GetMaxAdjacentCover(agent.X, agent.Y)
	forceMovement := currentCover == 0

	// Search for positions adjacent to cover tiles - increased search radius
	maxSearchDistance := 5 // Increased from 3 to ensure we can reach cover

	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			tile := g.Grid[y][x]

			// Skip if this tile has cover (impassable)
			if tile.Type > 0 {
				continue
			}

			// Check if position is reachable
			distance := abs(agent.X-x) + abs(agent.Y-y)
			if distance > maxSearchDistance {
				continue
			}

			// Calculate protection score considering enemy positions
			protectionScore := g.CalculatePositionProtection(x, y, enemies)

			// Strong bonus for positions that are actually adjacent to cover
			adjacentCover := g.GetMaxAdjacentCover(x, y)
			if adjacentCover > 0 {
				protectionScore += float64(adjacentCover) * 15.0 // Stronger bonus: 15 for low, 30 for high cover
			}

			// If we're forcing movement (agent in open), heavily penalize staying in current position
			if forceMovement && x == agent.X && y == agent.Y {
				protectionScore -= 50.0 // Heavy penalty for staying put when exposed
			}

			// Small penalty for distance to encourage closer positions when protection is equal
			distancePenalty := float64(distance) * 0.5
			finalScore := protectionScore - distancePenalty

			if finalScore > bestScore {
				bestX, bestY = x, y
				bestScore = finalScore
			}
		}
	}

	fmt.Fprintln(os.Stderr, fmt.Sprintf("Agent %d best cover: (%d,%d) with protection score %.2f (current cover: %d)",
		agent.ID, bestX, bestY, bestScore, currentCover))

	return bestX, bestY
}

// GetEnemyPositions returns all enemy agent positions
func (g *Game) GetEnemyPositions() [][]int {
	enemies := [][]int{}

	for _, enemy := range g.Agents {
		if enemy.Player != g.MyID && enemy.Wetness < 100 {
			enemies = append(enemies, []int{enemy.X, enemy.Y})
		}
	}

	return enemies
}

// CalculatePositionProtection calculates how well a position is protected from enemies
func (g *Game) CalculatePositionProtection(x, y int, enemies [][]int) float64 {
	if len(enemies) == 0 {
		return 0.0
	}

	totalProtection := 0.0

	// Check orthogonally adjacent tiles for cover
	directions := [][]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} // up, down, left, right

	for _, dir := range directions {
		coverX, coverY := x+dir[0], y+dir[1]

		if !g.IsValidPosition(coverX, coverY) {
			continue
		}

		coverType := g.Grid[coverY][coverX].Type
		if coverType == 0 {
			continue // No cover here
		}

		// Calculate protection from this cover tile against all enemies
		for _, enemy := range enemies {
			enemyX, enemyY := enemy[0], enemy[1]

			// Check if this cover blocks line of sight from enemy
			if g.DoesCoverBlockShot(x, y, coverX, coverY, enemyX, enemyY) {
				coverProtection := 0.0
				switch coverType {
				case 1:
					coverProtection = 0.5 // Low cover: 50%
				case 2:
					coverProtection = 0.75 // High cover: 75%
				}
				totalProtection += coverProtection
			}
		}
	}

	return totalProtection
}

// DoesCoverBlockShot checks if a cover tile blocks a shot from enemy to agent position
func (g *Game) DoesCoverBlockShot(agentX, agentY, coverX, coverY, enemyX, enemyY int) bool {
	// Cover blocks shot if it's between the enemy and agent position
	// Simple check: cover must be on the line between enemy and agent

	// Vector from enemy to agent
	dx := agentX - enemyX
	dy := agentY - enemyY

	// Cover blocks if it's roughly between enemy and agent
	// and agent is on opposite side of cover from enemy
	if abs(dx) >= abs(dy) {
		// Horizontal shot - cover blocks if it's between X coordinates
		return (enemyX < coverX && coverX < agentX) || (agentX < coverX && coverX < enemyX)
	} else {
		// Vertical shot - cover blocks if it's between Y coordinates
		return (enemyY < coverY && coverY < agentY) || (agentY < coverY && coverY < enemyY)
	}
}

// GetMaxAdjacentCover returns the highest cover value adjacent to a position
func (g *Game) GetMaxAdjacentCover(x, y int) int {
	maxCover := 0

	// Check orthogonally adjacent tiles (up, down, left, right)
	directions := [][]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

	for _, dir := range directions {
		adjX, adjY := x+dir[0], y+dir[1]
		if g.IsValidPosition(adjX, adjY) {
			coverType := g.Grid[adjY][adjX].Type
			if coverType > maxCover {
				maxCover = coverType
			}
		}
	}

	return maxCover
}

// FindLeastProtectedEnemy finds the best target balancing distance and protection
func (g *Game) FindLeastProtectedEnemy(agent *Agent) *Agent {
	var bestTarget *Agent
	bestScore := -999.0
	bestDistance := 999

	// Find the best target using a combined score of distance and protection
	for _, enemy := range g.Agents {
		if enemy.Player != g.MyID && enemy.Wetness < 100 {
			distance := abs(agent.X-enemy.X) + abs(agent.Y-enemy.Y)

			// Skip enemies outside optimal range
			if distance > agent.OptimalRange {
				continue
			}

			protection := g.CalculateCoverProtection(agent.X, agent.Y, enemy.X, enemy.Y)

			// Calculate combined score: prioritize unprotected enemies, but consider distance
			// Higher score is better
			distanceScore := float64(agent.OptimalRange) - float64(distance) // Closer = higher score
			protectionScore := (1.0 - protection) * 30.0                     // Unprotected = higher score
			combinedScore := distanceScore + protectionScore

			fmt.Fprintln(os.Stderr, fmt.Sprintf("Enemy %d: distance %d, protection %.1f%%, score %.1f",
				enemy.ID, distance, protection*100, combinedScore))

			// Better tie-breaking: prefer higher score, then closer distance, then lower ID
			if combinedScore > bestScore ||
				(combinedScore == bestScore && distance < bestDistance) ||
				(combinedScore == bestScore && distance == bestDistance && enemy.ID < bestTarget.ID) {
				bestTarget = enemy
				bestScore = combinedScore
				bestDistance = distance
			}
		}
	}

	if bestTarget != nil {
		distance := abs(agent.X-bestTarget.X) + abs(agent.Y-bestTarget.Y)
		protection := g.CalculateCoverProtection(agent.X, agent.Y, bestTarget.X, bestTarget.Y)
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Best target: Enemy %d (distance: %d, protection: %.1f%%)",
			bestTarget.ID, distance, protection*100))
	}

	return bestTarget
}

// CalculateCoverProtection calculates damage reduction from cover between shooter and target
func (g *Game) CalculateCoverProtection(shooterX, shooterY, targetX, targetY int) float64 {
	// For now, simplified: check if target is adjacent to cover
	// This should be enhanced with line-of-sight and cover direction logic

	maxCover := g.GetMaxAdjacentCover(targetX, targetY)

	switch maxCover {
	case 0:
		return 0.0 // No protection
	case 1:
		return 0.5 // Low cover: 50% protection
	case 2:
		return 0.75 // High cover: 75% protection
	default:
		return 0.0
	}
}

// resolveActionConflicts sorts actions by priority and handles movement conflicts
func (g *Game) resolveActionConflicts(allActions map[int][]AgentAction) map[int][]AgentAction {
	finalActions := make(map[int][]AgentAction)

	// Step 1: Extract movement actions and resolve collisions
	moveActions := make(map[int]AgentAction)
	nonMoveActions := make(map[int][]AgentAction)

	for agentID, actions := range allActions {
		nonMoveActions[agentID] = []AgentAction{}

		for _, action := range actions {
			if action.Type == ActionMove {
				moveActions[agentID] = action
			} else {
				nonMoveActions[agentID] = append(nonMoveActions[agentID], action)
			}
		}
	}

	// Step 2: Resolve movement collisions
	resolvedMoves := g.resolveMovementCollisions(moveActions)

	// Step 3: Combine resolved moves with other actions
	for agentID, nonMoves := range nonMoveActions {
		finalActions[agentID] = []AgentAction{}

		// Add resolved movement if exists
		if moveAction, hasMove := resolvedMoves[agentID]; hasMove {
			finalActions[agentID] = append(finalActions[agentID], moveAction)
		}

		// Add non-movement actions (sorted by priority)
		sortedNonMoves := make([]AgentAction, len(nonMoves))
		copy(sortedNonMoves, nonMoves)
		for i := 0; i < len(sortedNonMoves); i++ {
			for j := i + 1; j < len(sortedNonMoves); j++ {
				if sortedNonMoves[i].Priority < sortedNonMoves[j].Priority {
					sortedNonMoves[i], sortedNonMoves[j] = sortedNonMoves[j], sortedNonMoves[i]
				}
			}
		}

		finalActions[agentID] = append(finalActions[agentID], sortedNonMoves...)
	}

	return finalActions
}

// resolveMovementCollisions prevents agents from moving to the same tile
func (g *Game) resolveMovementCollisions(actions map[int]AgentAction) map[int]AgentAction {
	resolvedActions := make(map[int]AgentAction)

	// Sort agents by action priority first, then by agent ID for tie-breaking
	type agentPriority struct {
		agentID  int
		priority int
	}

	agentPriorities := make([]agentPriority, 0, len(actions))
	for agentID, action := range actions {
		agentPriorities = append(agentPriorities, agentPriority{agentID, action.Priority})
	}

	// Sort by priority (higher first), then by agent ID (lower first)
	for i := 0; i < len(agentPriorities); i++ {
		for j := i + 1; j < len(agentPriorities); j++ {
			if agentPriorities[i].priority < agentPriorities[j].priority ||
				(agentPriorities[i].priority == agentPriorities[j].priority && agentPriorities[i].agentID > agentPriorities[j].agentID) {
				agentPriorities[i], agentPriorities[j] = agentPriorities[j], agentPriorities[i]
			}
		}
	}

	occupiedPositions := make(map[string]bool)

	// Mark current agent positions as occupied to prevent staying in place conflicts
	for _, agent := range g.MyAgents {
		currentPosKey := fmt.Sprintf("%d,%d", agent.X, agent.Y)
		occupiedPositions[currentPosKey] = false // Mark as potentially available
	}

	// Process movement actions in priority order
	for _, ap := range agentPriorities {
		agentID := ap.agentID
		action := actions[agentID]
		agent := g.Agents[agentID]

		posKey := fmt.Sprintf("%d,%d", action.TargetX, action.TargetY)
		currentPosKey := fmt.Sprintf("%d,%d", agent.X, agent.Y)

		// Check if target position is available and valid
		if !occupiedPositions[posKey] && g.IsValidPosition(action.TargetX, action.TargetY) &&
			g.Grid[action.TargetY][action.TargetX].Type == 0 {
			// Position is free, take it
			resolvedActions[agentID] = action
			occupiedPositions[posKey] = true

			// Free up current position
			occupiedPositions[currentPosKey] = false

			fmt.Fprintln(os.Stderr, fmt.Sprintf("Agent %d moving to (%d,%d) [priority %d]", agentID, action.TargetX, action.TargetY, action.Priority))
		} else {
			// Position occupied, invalid, or blocked - find alternatives
			altX, altY, found := g.FindBestAlternativeMove(agent, action.TargetX, action.TargetY, occupiedPositions)

			if found {
				altKey := fmt.Sprintf("%d,%d", altX, altY)
				resolvedActions[agentID] = AgentAction{
					Type:     ActionMove,
					TargetX:  altX,
					TargetY:  altY,
					Priority: action.Priority,
					Reason:   fmt.Sprintf("Alternative move due to collision - %s", action.Reason),
				}
				occupiedPositions[altKey] = true
				// Free up current position
				occupiedPositions[currentPosKey] = false

				fmt.Fprintln(os.Stderr, fmt.Sprintf("Agent %d taking alternative move to (%d,%d) [wanted (%d,%d)]",
					agentID, altX, altY, action.TargetX, action.TargetY))
			} else {
				// No good alternative found, stay put
				resolvedActions[agentID] = AgentAction{
					Type:     ActionMove,
					TargetX:  agent.X,
					TargetY:  agent.Y,
					Priority: PriorityDefault,
					Reason:   "Staying put - no alternatives available",
				}
				// Keep current position occupied
				occupiedPositions[currentPosKey] = true

				fmt.Fprintln(os.Stderr, fmt.Sprintf("Agent %d staying put at (%d,%d) - no alternatives", agentID, agent.X, agent.Y))
			}
		}
	}

	return resolvedActions
}

// Action represents a single action for an agent
type Action struct {
	MoveX, MoveY   int
	ShootID        int
	ThrowX, ThrowY int
	HunkerDown     bool
	Message        string
	Reason         string
}

// CalculateMoveToward calculates the next move toward a specific target position
func (g *Game) CalculateMoveToward(agent *Agent, targetX, targetY int) (int, int) {
	dx := targetX - agent.X
	dy := targetY - agent.Y

	// If already at target, stay put
	if dx == 0 && dy == 0 {
		return agent.X, agent.Y
	}

	// Move in direction with largest remaining distance
	nextX, nextY := agent.X, agent.Y

	if abs(dx) >= abs(dy) && dx != 0 {
		// Prioritize X movement
		if dx > 0 {
			nextX++
		} else {
			nextX--
		}
	} else if dy != 0 {
		// Prioritize Y movement
		if dy > 0 {
			nextY++
		} else {
			nextY--
		}
	}

	return nextX, nextY
}

// GetCurrentTarget returns the current turn's target (cached)
func (g *Game) GetCurrentTarget() *Agent {
	if !g.TargetCached {
		g.CurrentTarget = g.FindHighestWetnessEnemy()
		g.TargetCached = true
	}
	return g.CurrentTarget
}

// FindHighestWetnessEnemy finds the alive enemy with the highest wetness
func (g *Game) FindHighestWetnessEnemy() *Agent {
	var bestTarget *Agent
	maxWetness := -1
	aliveEnemies := 0

	for _, enemy := range g.Agents {
		if enemy.Player != g.MyID {
			if enemy.Wetness >= 100 {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("Enemy Agent %d eliminated (wetness: %d)",
					enemy.ID, enemy.Wetness))
			} else {
				aliveEnemies++
				// Only consider alive enemies for targeting
				if enemy.Wetness > maxWetness {
					bestTarget = enemy
					maxWetness = enemy.Wetness
				}
			}
		}
	}

	fmt.Fprintln(os.Stderr, fmt.Sprintf("Alive enemies: %d", aliveEnemies))

	if bestTarget != nil {
		// Check if target changed
		if g.PreviousTargetID != bestTarget.ID {
			if g.PreviousTargetID != 0 {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("TARGET CHANGED: %d -> %d",
					g.PreviousTargetID, bestTarget.ID))
			}
			g.PreviousTargetID = bestTarget.ID
		}

		fmt.Fprintln(os.Stderr, fmt.Sprintf("Current target: Agent %d (wetness: %d)",
			bestTarget.ID, bestTarget.Wetness))
	} else {
		fmt.Fprintln(os.Stderr, "No valid enemy targets found - all enemies eliminated!")
		g.PreviousTargetID = 0
	}

	return bestTarget
}

// FindNearestEnemy finds the nearest enemy to the given agent
func (g *Game) FindNearestEnemy(agent *Agent) *Agent {
	var nearestEnemy *Agent
	minDistance := 9999

	for _, enemy := range g.Agents {
		if enemy.Player != g.MyID {
			distance := abs(agent.X-enemy.X) + abs(agent.Y-enemy.Y)
			if distance < minDistance {
				nearestEnemy = enemy
				minDistance = distance
			}
		}
	}

	return nearestEnemy
}

// CheckCollision returns true if two positions would collide
func (g *Game) CheckCollision(x1, y1, x2, y2 int) bool {
	return x1 == x2 && y1 == y2
}

// GetAlternateMove tries to find an alternate direction when collision occurs
func (g *Game) GetAlternateMove(agent *Agent, blockedX, blockedY int) (int, int) {
	dx := agent.TargetX - agent.X
	dy := agent.TargetY - agent.Y

	// Try the other direction
	nextX, nextY := agent.X, agent.Y

	// If we were moving in X, try Y
	if blockedX != agent.X && dy != 0 {
		if dy > 0 {
			nextY++
		} else {
			nextY--
		}
	} else if blockedY != agent.Y && dx != 0 {
		// If we were moving in Y, try X
		if dx > 0 {
			nextX++
		} else {
			nextX--
		}
	}

	return nextX, nextY
}

// IsValidPosition checks if a position is within grid bounds
func (g *Game) IsValidPosition(x, y int) bool {
	return x >= 0 && x < g.Width && y >= 0 && y < g.Height
}

// PrintAgentLocations prints friend and enemy positions for debugging
func (g *Game) PrintAgentLocations() {
	fmt.Fprintln(os.Stderr, "")

	// Print friend locations first
	fmt.Fprintln(os.Stderr, "=== FRIEND LOCATIONS ===")
	friendCount := 0
	for _, agent := range g.Agents {
		if agent.Player == g.MyID {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Agent %d: (%d,%d) - Wetness: %d, Bombs: %d, Cooldown: %d",
				agent.ID, agent.X, agent.Y, agent.Wetness, agent.SplashBombs, agent.Cooldown))
			friendCount++
		}
	}
	if friendCount == 0 {
		fmt.Fprintln(os.Stderr, "No friendly agents found")
	}

	fmt.Fprintln(os.Stderr, "")

	// Print enemy locations second
	fmt.Fprintln(os.Stderr, "=== ENEMY LOCATIONS ===")
	enemyCount := 0
	for _, agent := range g.Agents {
		if agent.Player != g.MyID {
			status := "Alive"
			if agent.Wetness >= 100 {
				status = "Eliminated"
			}
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Enemy %d: (%d,%d) - Wetness: %d, Status: %s",
				agent.ID, agent.X, agent.Y, agent.Wetness, status))
			enemyCount++
		}
	}
	if enemyCount == 0 {
		fmt.Fprintln(os.Stderr, "No enemy agents found")
	}

	fmt.Fprintln(os.Stderr, "========================")
}

// EvaluateTerritoryControl calculates current territory advantage/disadvantage
func (g *Game) EvaluateTerritoryControl() TerritoryScore {
	friendlyTiles := 0
	enemyTiles := 0
	contested := 0

	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			// Skip impassable tiles
			if g.Grid[y][x].Type > 0 {
				continue
			}

			closestFriendly := 999
			closestEnemy := 999

			// Find closest friendly agent
			for _, agent := range g.MyAgents {
				distance := abs(agent.X-x) + abs(agent.Y-y)
				// Double distance if agent has wetness >= 50
				if agent.Wetness >= 50 {
					distance *= 2
				}
				if distance < closestFriendly {
					closestFriendly = distance
				}
			}

			// Find closest enemy agent
			for _, agent := range g.Agents {
				if agent.Player != g.MyID && agent.Wetness < 100 {
					distance := abs(agent.X-x) + abs(agent.Y-y)
					// Double distance if agent has wetness >= 50
					if agent.Wetness >= 50 {
						distance *= 2
					}
					if distance < closestEnemy {
						closestEnemy = distance
					}
				}
			}

			// Determine control
			if closestFriendly < closestEnemy {
				friendlyTiles++
			} else if closestEnemy < closestFriendly {
				enemyTiles++
			} else {
				contested++
			}
		}
	}

	score := TerritoryScore{
		FriendlyTiles: friendlyTiles,
		EnemyTiles:    enemyTiles,
		Contested:     contested,
		Advantage:     friendlyTiles - enemyTiles,
	}

	fmt.Fprintln(os.Stderr, fmt.Sprintf("Territory: F:%d E:%d C:%d Advantage:%d",
		friendlyTiles, enemyTiles, contested, score.Advantage))

	return score
}

// TerritoryScore holds territory control evaluation
type TerritoryScore struct {
	FriendlyTiles int
	EnemyTiles    int
	Contested     int
	Advantage     int
}

// EvaluateEmergencyActions checks for immediate threats or elimination opportunities
func (g *Game) EvaluateEmergencyActions(agent *Agent) AgentAction {
	// Check if we can eliminate an enemy with a splash bomb
	if agent.SplashBombs > 0 {
		bombX, bombY, targets := g.FindEliminationBombTarget(agent)
		if len(targets) > 0 {
			return AgentAction{
				Type:     ActionThrow,
				TargetX:  bombX,
				TargetY:  bombY,
				Priority: PriorityEmergency,
				Reason:   fmt.Sprintf("Emergency elimination bomb - %d targets", len(targets)),
			}
		}
	}

	// Check if we can eliminate an enemy with shooting
	if agent.Cooldown == 0 {
		target := g.FindEliminationShootTarget(agent)
		if target != nil {
			return AgentAction{
				Type:          ActionShoot,
				TargetAgentID: target.ID,
				Priority:      PriorityEmergency,
				Reason:        fmt.Sprintf("Emergency elimination shot - Enemy %d", target.ID),
			}
		}
	}

	// Check if agent is in immediate danger
	if g.IsInImmediateDanger(agent) {
		// Try to move to safety
		safeX, safeY := g.FindSafetyPosition(agent)
		if safeX != agent.X || safeY != agent.Y {
			return AgentAction{
				Type:     ActionMove,
				TargetX:  safeX,
				TargetY:  safeY,
				Priority: PriorityEmergency,
				Reason:   "Emergency escape from danger",
			}
		}
	}

	// No emergency action needed
	return AgentAction{Type: -1}
}

// FindEliminationBombTarget finds bomb targets that would eliminate enemies
func (g *Game) FindEliminationBombTarget(agent *Agent) (int, int, []*Agent) {
	bestX, bestY := agent.X, agent.Y
	bestTargets := []*Agent{}

	// Check all positions within splash bomb range
	for targetY := 0; targetY < g.Height; targetY++ {
		for targetX := 0; targetX < g.Width; targetX++ {
			distance := abs(agent.X-targetX) + abs(agent.Y-targetY)
			if distance > 4 {
				continue
			}

			// Check for friendly fire
			if g.WouldHitFriendlyAgents(targetX, targetY) {
				continue
			}

			// Find enemies that would be eliminated
			eliminationTargets := []*Agent{}

			// Check 3x3 area around bomb
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					checkX, checkY := targetX+dx, targetY+dy
					if !g.IsValidPosition(checkX, checkY) {
						continue
					}

					for _, enemy := range g.Agents {
						if enemy.Player != g.MyID && enemy.Wetness < 100 &&
							enemy.X == checkX && enemy.Y == checkY {
							// Splash bombs deal 30 damage and ignore damage reduction
							if enemy.Wetness+30 >= 100 {
								eliminationTargets = append(eliminationTargets, enemy)
							}
						}
					}
				}
			}

			// Prefer targets with more eliminations
			if len(eliminationTargets) > len(bestTargets) {
				bestX, bestY = targetX, targetY
				bestTargets = eliminationTargets
			}
		}
	}

	return bestX, bestY, bestTargets
}

// FindEliminationShootTarget finds shooting targets that would eliminate enemies
func (g *Game) FindEliminationShootTarget(agent *Agent) *Agent {
	for _, enemy := range g.Agents {
		if enemy.Player != g.MyID && enemy.Wetness < 100 {
			distance := abs(agent.X-enemy.X) + abs(agent.Y-enemy.Y)
			if distance > agent.OptimalRange*2 {
				continue // Out of range
			}

			// Calculate damage after cover reduction
			damage := float64(agent.SoakingPower)
			if distance > agent.OptimalRange {
				damage *= 0.5 // Reduced damage beyond optimal range
			}

			// Apply cover reduction
			cover := g.CalculateCoverProtection(agent.X, agent.Y, enemy.X, enemy.Y)
			damage *= (1.0 - cover)

			// Check if this would eliminate the enemy
			if float64(enemy.Wetness)+damage >= 100 {
				return enemy
			}
		}
	}
	return nil
}

// IsInImmediateDanger checks if agent is under immediate threat
func (g *Game) IsInImmediateDanger(agent *Agent) bool {
	// Check if any enemy can eliminate us with their next attack
	for _, enemy := range g.Agents {
		if enemy.Player != g.MyID && enemy.Wetness < 100 {
			distance := abs(agent.X-enemy.X) + abs(agent.Y-enemy.Y)

			// Check shooting threat
			if distance <= enemy.OptimalRange && enemy.Cooldown == 0 {
				damage := float64(enemy.SoakingPower)
				cover := g.CalculateCoverProtection(enemy.X, enemy.Y, agent.X, agent.Y)
				damage *= (1.0 - cover)

				if float64(agent.Wetness)+damage >= 100 {
					return true
				}
			}

			// Check splash bomb threat (assume enemy has bombs)
			if distance <= 4 {
				// Splash bombs deal 30 damage and ignore cover
				if agent.Wetness+30 >= 100 {
					return true
				}
			}
		}
	}
	return false
}

// FindSafetyPosition finds the safest position for an agent in danger
func (g *Game) FindSafetyPosition(agent *Agent) (int, int) {
	bestX, bestY := agent.X, agent.Y
	bestSafety := g.CalculatePositionSafety(agent.X, agent.Y, agent)

	// Search nearby positions
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			newX, newY := agent.X+dx, agent.Y+dy
			if !g.IsValidPosition(newX, newY) || g.Grid[newY][newX].Type > 0 {
				continue
			}

			safety := g.CalculatePositionSafety(newX, newY, agent)
			if safety > bestSafety {
				bestX, bestY = newX, newY
				bestSafety = safety
			}
		}
	}

	return bestX, bestY
}

// CalculatePositionSafety calculates how safe a position is from enemy threats
func (g *Game) CalculatePositionSafety(x, y int, agent *Agent) float64 {
	safety := 100.0 // Base safety score

	// Check threats from all enemies
	for _, enemy := range g.Agents {
		if enemy.Player != g.MyID && enemy.Wetness < 100 {
			distance := abs(x-enemy.X) + abs(y-enemy.Y)

			// Reduce safety based on enemy proximity and threat level
			threat := 0.0
			if distance <= enemy.OptimalRange {
				threat += 30.0 // Shooting threat
			}
			if distance <= 4 {
				threat += 20.0 // Bomb threat
			}

			// Apply cover protection
			cover := g.CalculateCoverProtection(enemy.X, enemy.Y, x, y)
			threat *= (1.0 - cover)

			safety -= threat
		}
	}

	// Bonus for being near cover
	coverLevel := g.GetMaxAdjacentCover(x, y)
	safety += float64(coverLevel) * 10.0

	return safety
}

// EvaluateStrategicBombing evaluates splash bomb opportunities considering territory
func (g *Game) EvaluateStrategicBombing(agent *Agent, territoryScore TerritoryScore) AgentAction {
	bestX, bestY := agent.X, agent.Y
	bestScore := 0.0

	for targetY := 0; targetY < g.Height; targetY++ {
		for targetX := 0; targetX < g.Width; targetX++ {
			distance := abs(agent.X-targetX) + abs(agent.Y-targetY)
			if distance > 4 || g.WouldHitFriendlyAgents(targetX, targetY) {
				continue
			}

			score := 0.0

			// Damage score
			damageScore := g.CalculateSplashDamageScore(targetX, targetY)
			score += damageScore

			// Territory impact score
			territoryImpact := g.CalculateBombTerritoryImpact(targetX, targetY)
			score += territoryImpact * 2.0 // Weight territory impact highly

			// Area denial score (blocking enemy movement)
			denialScore := g.CalculateAreaDenialScore(targetX, targetY)
			score += denialScore

			if score > bestScore && score >= 100.0 { // Minimum threshold
				bestX, bestY = targetX, targetY
				bestScore = score
			}
		}
	}

	if bestScore >= 100.0 {
		return AgentAction{
			Type:     ActionThrow,
			TargetX:  bestX,
			TargetY:  bestY,
			Priority: PriorityCombat,
			Reason:   fmt.Sprintf("Strategic bombing - score: %.0f", bestScore),
		}
	}

	return AgentAction{Type: -1}
}

// CalculateBombTerritoryImpact calculates how a bomb would affect territory control
func (g *Game) CalculateBombTerritoryImpact(bombX, bombY int) float64 {
	impact := 0.0

	// Check 3x3 area around bomb
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			checkX, checkY := bombX+dx, bombY+dy
			if !g.IsValidPosition(checkX, checkY) {
				continue
			}

			// Check if eliminating enemies here would improve territory control
			for _, enemy := range g.Agents {
				if enemy.Player != g.MyID && enemy.Wetness < 100 &&
					enemy.X == checkX && enemy.Y == checkY {

					// Calculate territory gain from eliminating this enemy
					territoryGain := g.CalculateTerritoryGainFromElimination(enemy)
					impact += territoryGain
				}
			}
		}
	}

	return impact
}

// CalculateTerritoryGainFromElimination estimates territory gain from eliminating an enemy
func (g *Game) CalculateTerritoryGainFromElimination(enemy *Agent) float64 {
	gain := 0.0
	searchRadius := enemy.OptimalRange + 2

	// Check area around enemy's position
	for dy := -searchRadius; dy <= searchRadius; dy++ {
		for dx := -searchRadius; dx <= searchRadius; dx++ {
			checkX, checkY := enemy.X+dx, enemy.Y+dy
			if !g.IsValidPosition(checkX, checkY) || g.Grid[checkY][checkX].Type > 0 {
				continue
			}

			// Check if this tile would flip from enemy to friendly control
			currentDistance := abs(checkX-enemy.X) + abs(checkY-enemy.Y)
			if enemy.Wetness >= 50 {
				currentDistance *= 2
			}

			// Find closest friendly after enemy elimination
			closestFriendly := 999
			for _, friendly := range g.MyAgents {
				distance := abs(friendly.X-checkX) + abs(friendly.Y-checkY)
				if friendly.Wetness >= 50 {
					distance *= 2
				}
				if distance < closestFriendly {
					closestFriendly = distance
				}
			}

			// If friendly would now control this tile, add to gain
			if closestFriendly < currentDistance {
				gain += 1.0
			}
		}
	}

	return gain
}

// CalculateAreaDenialScore calculates strategic value of denying area to enemies
func (g *Game) CalculateAreaDenialScore(bombX, bombY int) float64 {
	// This is a simplified version - could be enhanced with pathfinding analysis
	denial := 0.0

	// Check if bomb position blocks key movement corridors
	// For now, just check if it's in a central/strategic location
	centerX, centerY := g.Width/2, g.Height/2
	distanceFromCenter := abs(bombX-centerX) + abs(bombY-centerY)

	// Closer to center = more strategic for area denial
	denial += float64(10 - distanceFromCenter)

	return denial
}

// EvaluateStrategicShooting evaluates shooting opportunities considering territory
func (g *Game) EvaluateStrategicShooting(agent *Agent, territoryScore TerritoryScore) AgentAction {
	bestTarget := g.FindStrategicShootTarget(agent, territoryScore)

	if bestTarget != nil {
		return AgentAction{
			Type:          ActionShoot,
			TargetAgentID: bestTarget.ID,
			Priority:      PriorityCombat,
			Reason:        fmt.Sprintf("Strategic shooting - Enemy %d", bestTarget.ID),
		}
	}

	return AgentAction{Type: -1}
}

// FindStrategicShootTarget finds the best shooting target considering multiple factors
func (g *Game) FindStrategicShootTarget(agent *Agent, territoryScore TerritoryScore) *Agent {
	bestTarget := (*Agent)(nil)
	bestScore := 0.0

	for _, enemy := range g.Agents {
		if enemy.Player != g.MyID && enemy.Wetness < 100 {
			distance := abs(agent.X-enemy.X) + abs(agent.Y-enemy.Y)
			if distance > agent.OptimalRange*2 {
				continue
			}

			score := 0.0

			// Damage effectiveness score
			damage := float64(agent.SoakingPower)
			if distance > agent.OptimalRange {
				damage *= 0.5
			}
			cover := g.CalculateCoverProtection(agent.X, agent.Y, enemy.X, enemy.Y)
			effectiveDamage := damage * (1.0 - cover)
			score += effectiveDamage

			// Elimination bonus
			if float64(enemy.Wetness)+effectiveDamage >= 100 {
				score += 100.0
				// Territory gain bonus for elimination
				territoryGain := g.CalculateTerritoryGainFromElimination(enemy)
				score += territoryGain * 5.0
			}

			// High wetness targets (easier to finish)
			score += float64(enemy.Wetness) * 0.5

			// Distance penalty (prefer closer targets)
			score -= float64(distance) * 2.0

			if score > bestScore {
				bestTarget = enemy
				bestScore = score
			}
		}
	}

	return bestTarget
}

// EvaluateStrategicMovement evaluates movement for territory control and positioning
func (g *Game) EvaluateStrategicMovement(agent *Agent, territoryScore TerritoryScore) AgentAction {
	bestX, bestY := agent.X, agent.Y
	bestScore := g.CalculatePositionValue(agent.X, agent.Y, agent, territoryScore)

	// Search area around agent
	searchRadius := 3
	for dy := -searchRadius; dy <= searchRadius; dy++ {
		for dx := -searchRadius; dx <= searchRadius; dx++ {
			newX, newY := agent.X+dx, agent.Y+dy
			if !g.IsValidPosition(newX, newY) || g.Grid[newY][newX].Type > 0 {
				continue
			}

			score := g.CalculatePositionValue(newX, newY, agent, territoryScore)
			if score > bestScore {
				bestX, bestY = newX, newY
				bestScore = score
			}
		}
	}

	if bestX != agent.X || bestY != agent.Y {
		return AgentAction{
			Type:     ActionMove,
			TargetX:  bestX,
			TargetY:  bestY,
			Priority: PriorityMovement,
			Reason:   fmt.Sprintf("Strategic positioning - score: %.1f", bestScore),
		}
	}

	return AgentAction{Type: -1}
}

// CalculatePositionValue calculates the strategic value of a position
func (g *Game) CalculatePositionValue(x, y int, agent *Agent, territoryScore TerritoryScore) float64 {
	score := 0.0

	// Territory control value
	territoryValue := g.CalculatePositionTerritoryValue(x, y)
	score += territoryValue * 10.0

	// Combat positioning value
	combatValue := g.CalculatePositionCombatValue(x, y, agent)
	score += combatValue * 5.0

	// Safety value
	safetyValue := g.CalculatePositionSafety(x, y, agent)
	score += safetyValue * 1.0

	// Cover bonus
	coverLevel := g.GetMaxAdjacentCover(x, y)
	score += float64(coverLevel) * 8.0

	// Coordination penalty: avoid positions too close to other friendly agents
	coordinationPenalty := g.CalculateCoordinationPenalty(x, y, agent)
	score -= coordinationPenalty

	// Agent-specific spread: add small variation based on agent ID to break ties
	agentSpread := float64(agent.ID%4) * 0.1 // Small variation: 0.0, 0.1, 0.2, 0.3
	score += agentSpread

	return score
}

// CalculateCoordinationPenalty calculates penalty for positions too close to other friendly agents
func (g *Game) CalculateCoordinationPenalty(x, y int, agent *Agent) float64 {
	penalty := 0.0

	for _, friendly := range g.MyAgents {
		if friendly.ID == agent.ID {
			continue
		}

		distance := abs(x-friendly.X) + abs(y-friendly.Y)

		// Heavy penalty for being too close (1-2 tiles away)
		if distance <= 1 {
			penalty += 50.0
		} else if distance == 2 {
			penalty += 20.0
		} else if distance == 3 {
			penalty += 5.0
		}
	}

	return penalty
}

// CalculatePositionTerritoryValue calculates how much territory this position could control
func (g *Game) CalculatePositionTerritoryValue(x, y int) float64 {
	value := 0.0
	controlRadius := 6 // Check area around position

	for dy := -controlRadius; dy <= controlRadius; dy++ {
		for dx := -controlRadius; dx <= controlRadius; dx++ {
			checkX, checkY := x+dx, y+dy
			if !g.IsValidPosition(checkX, checkY) || g.Grid[checkY][checkX].Type > 0 {
				continue
			}

			distance := abs(dx) + abs(dy)

			// Find closest enemy to this tile
			closestEnemyDistance := 999
			for _, enemy := range g.Agents {
				if enemy.Player != g.MyID && enemy.Wetness < 100 {
					enemyDistance := abs(enemy.X-checkX) + abs(enemy.Y-checkY)
					if enemy.Wetness >= 50 {
						enemyDistance *= 2
					}
					if enemyDistance < closestEnemyDistance {
						closestEnemyDistance = enemyDistance
					}
				}
			}

			// If we would control this tile, add value (weighted by distance)
			if distance < closestEnemyDistance {
				tileValue := 1.0 / (1.0 + float64(distance)*0.1)
				value += tileValue
			}
		}
	}

	return value
}

// CalculatePositionCombatValue calculates combat effectiveness from a position
func (g *Game) CalculatePositionCombatValue(x, y int, agent *Agent) float64 {
	value := 0.0

	// Count enemies in range from this position
	for _, enemy := range g.Agents {
		if enemy.Player != g.MyID && enemy.Wetness < 100 {
			distance := abs(x-enemy.X) + abs(y-enemy.Y)

			if distance <= agent.OptimalRange {
				value += 20.0 // In optimal shooting range
			} else if distance <= 4 {
				value += 10.0 // In bomb range
			} else if distance <= agent.OptimalRange*2 {
				value += 5.0 // In extended shooting range
			}
		}
	}

	return value
}

// IsUnderThreat checks if agent is under significant threat
func (g *Game) IsUnderThreat(agent *Agent) bool {
	threatLevel := 0.0

	for _, enemy := range g.Agents {
		if enemy.Player != g.MyID && enemy.Wetness < 100 {
			distance := abs(agent.X-enemy.X) + abs(agent.Y-enemy.Y)

			if distance <= enemy.OptimalRange {
				threatLevel += 30.0
			} else if distance <= 4 {
				threatLevel += 20.0
			}
		}
	}

	return threatLevel >= 40.0 // Threshold for significant threat
}

// FindBestAlternativeMove finds the best alternative position when the preferred position is occupied
func (g *Game) FindBestAlternativeMove(agent *Agent, preferredX, preferredY int, occupiedPositions map[string]bool) (int, int, bool) {
	bestX, bestY := agent.X, agent.Y
	bestScore := -999.0
	found := false

	// Search in expanding rings around the preferred position
	maxRadius := 3
	for radius := 1; radius <= maxRadius; radius++ {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				// Only check positions on the edge of current radius
				if abs(dx) != radius && abs(dy) != radius {
					continue
				}

				candidateX := preferredX + dx
				candidateY := preferredY + dy

				// Check if position is valid and available
				if !g.IsValidPosition(candidateX, candidateY) ||
					g.Grid[candidateY][candidateX].Type > 0 {
					continue
				}

				posKey := fmt.Sprintf("%d,%d", candidateX, candidateY)
				if occupiedPositions[posKey] {
					continue
				}

				// Score this alternative position
				score := g.scoreAlternativePosition(agent, candidateX, candidateY, preferredX, preferredY)

				if score > bestScore {
					bestX, bestY = candidateX, candidateY
					bestScore = score
					found = true
				}
			}
		}

		// If we found a good alternative at this radius, use it
		if found && bestScore > 0 {
			break
		}
	}

	// Fallback: try positions adjacent to current position if nothing better found
	if !found || bestScore <= -999.0 {
		directions := [][]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}

		for _, dir := range directions {
			candidateX := agent.X + dir[0]
			candidateY := agent.Y + dir[1]

			if !g.IsValidPosition(candidateX, candidateY) ||
				g.Grid[candidateY][candidateX].Type > 0 {
				continue
			}

			posKey := fmt.Sprintf("%d,%d", candidateX, candidateY)
			if occupiedPositions[posKey] {
				continue
			}

			// Even a small step is better than staying completely stuck
			score := g.scoreAlternativePosition(agent, candidateX, candidateY, preferredX, preferredY)
			if score > bestScore {
				bestX, bestY = candidateX, candidateY
				bestScore = score
				found = true
			}
		}
	}

	return bestX, bestY, found
}

// scoreAlternativePosition scores an alternative position based on how good it is
func (g *Game) scoreAlternativePosition(agent *Agent, candidateX, candidateY, preferredX, preferredY int) float64 {
	score := 0.0

	// Penalty for distance from preferred position (closer to preferred = better)
	distanceFromPreferred := abs(candidateX-preferredX) + abs(candidateY-preferredY)
	score -= float64(distanceFromPreferred) * 5.0

	// Bonus for movement progress (getting closer to preferred than current position)
	currentDistanceFromPreferred := abs(agent.X-preferredX) + abs(agent.Y-preferredY)
	if distanceFromPreferred < currentDistanceFromPreferred {
		score += 10.0 // Progress bonus
	}

	// Small bonus for cover nearby
	coverLevel := g.GetMaxAdjacentCover(candidateX, candidateY)
	score += float64(coverLevel) * 2.0

	// Bonus for territory control from this position
	territoryValue := g.CalculatePositionTerritoryValue(candidateX, candidateY)
	score += territoryValue * 0.5

	// Safety consideration
	safetyValue := g.CalculatePositionSafety(candidateX, candidateY, agent)
	score += safetyValue * 0.1

	return score
}
