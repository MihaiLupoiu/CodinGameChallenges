package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

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

	// Set initial strategy
	game.CurrentStrategy = &MoveAndShootStrategy{}
	fmt.Fprintln(os.Stderr, "Starting with strategy:", game.CurrentStrategy.Name())

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

		// myAgentCount: Number of alive agents controlled by you
		var myAgentCount int
		scanner.Scan()
		fmt.Sscan(scanner.Text(), &myAgentCount)

		// Coordinate all agent actions using current strategy
		actions := game.CoordinateActions()

		for _, agent := range game.MyAgents {
			action := actions[agent.ID]
			actionStr := game.FormatAction(action)

			log := fmt.Sprintf("Agent %d: %s (Reason: %s)", agent.ID, actionStr, action.Reason)
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
func (g *Game) CoordinateActions() map[int]AgentAction {
	allActions := make(map[int][]AgentAction)

	// Step 1: Each agent evaluates possible actions
	for _, agent := range g.MyAgents {
		actions := g.CurrentStrategy.EvaluateActions(agent, g)
		allActions[agent.ID] = actions
		log := fmt.Sprintf("Agent %d generated %d possible actions", agent.ID, len(actions))
		fmt.Fprintln(os.Stderr, log)
	}

	// Step 2: Select best action for each agent and resolve conflicts
	finalActions := g.resolveActionConflicts(allActions)

	return finalActions
}

// FormatAction formats an action into a string for output
func (g *Game) FormatAction(action AgentAction) string {
	switch action.Type {
	case ActionMove:
		return fmt.Sprintf("MOVE %d %d", action.TargetX, action.TargetY)
	case ActionShoot:
		return fmt.Sprintf("SHOOT %d", action.TargetAgentID)
	case ActionThrow:
		return fmt.Sprintf("THROW %d %d", action.TargetX, action.TargetY)
	case ActionHunker:
		return "HUNKER_DOWN"
	case ActionMessage:
		return fmt.Sprintf("MESSAGE %s", action.Message)
	default:
		return "HUNKER_DOWN"
	}
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
	PriorityCombat    = 80  // Shooting
	PriorityMovement  = 60  // Positioning
	PriorityDefault   = 40  // Hunker down
)

// Strategy defines the interface for coordinating agent actions
type Strategy interface {
	Name() string
	EvaluateActions(agent *Agent, game *Game) []AgentAction
}

// MoveAndShootStrategy implements a strategy that prioritizes movement to targets and shooting
type MoveAndShootStrategy struct{}

func (s *MoveAndShootStrategy) Name() string {
	return "TargetHighestWetness"
}

func (s *MoveAndShootStrategy) EvaluateActions(agent *Agent, game *Game) []AgentAction {
	actions := []AgentAction{}

	// Priority 1: Shoot the enemy with highest wetness (main objective)
	if shootAction := s.evaluateShooting(agent, game); shootAction != nil {
		actions = append(actions, *shootAction)
	}

	// Priority 2: Move to get in range of highest wetness enemy
	if moveAction := s.evaluateMovement(agent, game); moveAction != nil {
		actions = append(actions, *moveAction)
	}

	// Default fallback
	actions = append(actions, AgentAction{
		Type:     ActionHunker,
		Priority: PriorityDefault,
		Reason:   "In position - ready to engage or all enemies eliminated",
	})

	return actions
}

func (s *MoveAndShootStrategy) evaluateShooting(agent *Agent, game *Game) *AgentAction {
	if agent.Cooldown > 0 {
		return nil // Can't shoot yet
	}

	// Find the enemy with highest wetness (main objective) - uses cached target
	highestWetnessEnemy := game.GetCurrentTarget()
	if highestWetnessEnemy == nil {
		// No valid targets - all enemies eliminated or no enemies exist
		return &AgentAction{
			Type:     ActionMessage,
			Message:  "Victory!",
			Priority: PriorityDefault,
			Reason:   "No enemies left to target",
		}
	}

	// Check if this enemy is within shooting range
	distance := abs(agent.X-highestWetnessEnemy.X) + abs(agent.Y-highestWetnessEnemy.Y)
	maxRange := agent.OptimalRange * 2 // Shots fail beyond 2x optimal range

	if distance <= maxRange {
		return &AgentAction{
			Type:          ActionShoot,
			TargetAgentID: highestWetnessEnemy.ID,
			Priority:      PriorityCombat,
			Reason: fmt.Sprintf("Shooting highest wetness enemy %d (wetness: %d, range: %d)",
				highestWetnessEnemy.ID, highestWetnessEnemy.Wetness, distance),
		}
	}

	return nil // Out of range - movement will handle getting closer
}

func (s *MoveAndShootStrategy) evaluateMovement(agent *Agent, game *Game) *AgentAction {
	// Move toward enemy with highest wetness to get in shooting range - uses cached target
	target := game.GetCurrentTarget()
	if target == nil {
		// No enemies left - stay put and celebrate
		return nil
	}

	// Calculate current distance to target
	currentDistance := abs(agent.X-target.X) + abs(agent.Y-target.Y)
	maxRange := agent.OptimalRange * 2 // Max shooting range

	// Only move if we're out of shooting range
	if currentDistance > maxRange {
		nextX, nextY := game.CalculateMoveToward(agent, target.X, target.Y)
		if nextX != agent.X || nextY != agent.Y {
			newDistance := abs(nextX-target.X) + abs(nextY-target.Y)
			return &AgentAction{
				Type:     ActionMove,
				TargetX:  nextX,
				TargetY:  nextY,
				Priority: PriorityMovement,
				Reason: fmt.Sprintf("Moving toward highest wetness enemy %d (current range: %d, new range: %d)",
					target.ID, currentDistance, newDistance),
			}
		}
	}

	return nil // Already in range or can't move closer
}

// resolveActionConflicts selects the best action for each agent and handles conflicts
func (g *Game) resolveActionConflicts(allActions map[int][]AgentAction) map[int]AgentAction {
	finalActions := make(map[int]AgentAction)

	// Select highest priority action for each agent
	for agentID, actions := range allActions {
		if len(actions) > 0 {
			// Sort by priority (highest first)
			bestAction := actions[0]
			for _, action := range actions {
				if action.Priority > bestAction.Priority {
					bestAction = action
				}
			}
			finalActions[agentID] = bestAction
		}
	}

	// Handle movement collisions
	finalActions = g.resolveMovementCollisions(finalActions)

	return finalActions
}

// resolveMovementCollisions prevents agents from moving to the same tile
func (g *Game) resolveMovementCollisions(actions map[int]AgentAction) map[int]AgentAction {
	resolvedActions := make(map[int]AgentAction)

	// Copy non-movement actions
	for agentID, action := range actions {
		if action.Type != ActionMove {
			resolvedActions[agentID] = action
		}
	}

	// Sort agents by ID for consistent priority
	agentIDs := make([]int, 0, len(g.MyAgents))
	for _, agent := range g.MyAgents {
		agentIDs = append(agentIDs, agent.ID)
	}
	for i := 0; i < len(agentIDs); i++ {
		for j := i + 1; j < len(agentIDs); j++ {
			if agentIDs[i] > agentIDs[j] {
				agentIDs[i], agentIDs[j] = agentIDs[j], agentIDs[i]
			}
		}
	}

	occupiedPositions := make(map[string]bool)

	// Process movement actions in order of agent priority
	for _, agentID := range agentIDs {
		if action, exists := actions[agentID]; exists && action.Type == ActionMove {
			agent := g.Agents[agentID]
			posKey := fmt.Sprintf("%d,%d", action.TargetX, action.TargetY)

			if !occupiedPositions[posKey] && g.IsValidPosition(action.TargetX, action.TargetY) {
				// Position is free, take it
				resolvedActions[agentID] = action
				occupiedPositions[posKey] = true
				fmt.Fprintln(os.Stderr, fmt.Sprintf("Agent %d moving to (%d,%d)", agentID, action.TargetX, action.TargetY))
			} else {
				// Position occupied or invalid, try alternate
				altX, altY := g.GetAlternateMove(agent, action.TargetX, action.TargetY)
				altKey := fmt.Sprintf("%d,%d", altX, altY)

				if !occupiedPositions[altKey] && g.IsValidPosition(altX, altY) {
					resolvedActions[agentID] = AgentAction{
						Type:     ActionMove,
						TargetX:  altX,
						TargetY:  altY,
						Priority: action.Priority,
						Reason:   fmt.Sprintf("Alternate move due to collision - %s", action.Reason),
					}
					occupiedPositions[altKey] = true
					fmt.Fprintln(os.Stderr, fmt.Sprintf("Agent %d taking alternate move to (%d,%d)", agentID, altX, altY))
				} else {
					// Stay put
					resolvedActions[agentID] = AgentAction{
						Type:     ActionMove,
						TargetX:  agent.X,
						TargetY:  agent.Y,
						Priority: PriorityDefault,
						Reason:   "Staying put due to collision",
					}
					fmt.Fprintln(os.Stderr, fmt.Sprintf("Agent %d staying put at (%d,%d)", agentID, agent.X, agent.Y))
				}
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
