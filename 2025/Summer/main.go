package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ============================================================================
// CORE GAME STRUCTURES
// ============================================================================

// Debug configuration
const DEBUG_PRINT_AGENTS = true
const CAPTURE_INPUT_FORMAT = true

// Performance and timing constants
const TURN_TIME_LIMIT_MS = 2000 // 2 second turn limit

// ============================================================================
// TEAM STRATEGY FSM STATES
// ============================================================================

type TeamStrategyState int

const (
	TeamStateCombat           TeamStrategyState = iota // Focus on eliminating enemies
	TeamStateTerritoryControl                          // Capture and hold map territory
	TeamStateDefense                                   // Protect key areas/vulnerable agents
	TeamStateRegroupAndHeal                            // Move to safety when team is damaged
)

func (s TeamStrategyState) String() string {
	switch s {
	case TeamStateCombat:
		return "Combat"
	case TeamStateTerritoryControl:
		return "TerritoryControl"
	case TeamStateDefense:
		return "Defense"
	case TeamStateRegroupAndHeal:
		return "RegroupAndHeal"
	default:
		return "Unknown"
	}
}

// ============================================================================
// AGENT TACTICAL FSM STATES
// ============================================================================

type AgentTacticalState int

const (
	AgentStateIdle               AgentTacticalState = iota // Waiting, observing, no immediate task
	AgentStateMoveToPosition                               // Pathfinding to strategic location
	AgentStateEngageEnemy                                  // Active combat (shooting/bombing)
	AgentStateReloading                                    // On shooting cooldown
	AgentStateFleeing                                      // Low health, seeking safety
	AgentStateHunkering                                    // Using damage reduction
	AgentStateCapturingTerritory                           // Moving to control tiles
)

func (s AgentTacticalState) String() string {
	switch s {
	case AgentStateIdle:
		return "Idle"
	case AgentStateMoveToPosition:
		return "MoveTo"
	case AgentStateEngageEnemy:
		return "Engage"
	case AgentStateReloading:
		return "Reload"
	case AgentStateFleeing:
		return "Flee"
	case AgentStateHunkering:
		return "Hunker"
	case AgentStateCapturingTerritory:
		return "Capture"
	default:
		return "Unknown"
	}
}

// ============================================================================
// BEHAVIOR TREE NODES
// ============================================================================

// NodeState represents the outcome of a behavior tree node evaluation
type NodeState int

const (
	BTRunning NodeState = iota // Task is still in progress
	BTSuccess                  // Task completed successfully or condition met
	BTFailure                  // Task failed or condition not met
)

func (s NodeState) String() string {
	switch s {
	case BTRunning:
		return "Running"
	case BTSuccess:
		return "Success"
	case BTFailure:
		return "Failure"
	default:
		return "Unknown"
	}
}

// Node interface defines the contract for all behavior tree nodes
type Node interface {
	Evaluate(agent *Agent, game *Game) NodeState
	Name() string
}

// ============================================================================
// COMPOSITE BEHAVIOR TREE NODES
// ============================================================================

// Sequence node: Executes children in order, fails on first failure, succeeds if all succeed
type Sequence struct {
	Children []Node
	name     string
}

func NewSequence(name string, children ...Node) *Sequence {
	return &Sequence{Children: children, name: name}
}

func (s *Sequence) Name() string {
	return fmt.Sprintf("Sequence(%s)", s.name)
}

func (s *Sequence) Evaluate(agent *Agent, game *Game) NodeState {
	fmt.Fprintln(os.Stderr, fmt.Sprintf("🔗 Agent %d evaluating %s with %d children",
		agent.ID, s.Name(), len(s.Children)))

	for i, child := range s.Children {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("  ├─ Agent %d sequence step %d: %s",
			agent.ID, i, child.Name()))

		switch child.Evaluate(agent, game) {
		case BTFailure:
			fmt.Fprintln(os.Stderr, fmt.Sprintf("  ❌ Agent %d: sequence failed at %s",
				agent.ID, child.Name()))
			return BTFailure // One child failed, sequence fails
		case BTRunning:
			fmt.Fprintln(os.Stderr, fmt.Sprintf("  ⏳ Agent %d: sequence running at %s",
				agent.ID, child.Name()))
			return BTRunning // One child is running, sequence is running
		case BTSuccess:
			fmt.Fprintln(os.Stderr, fmt.Sprintf("  ✅ Agent %d: %s succeeded, continuing sequence",
				agent.ID, child.Name()))
			continue // Child succeeded, move to next child
		}
	}
	fmt.Fprintln(os.Stderr, fmt.Sprintf("  🎉 Agent %d: All children of %s succeeded",
		agent.ID, s.Name()))
	return BTSuccess // All children succeeded
}

// Selector node: Tries children in order, succeeds on first success, fails if all fail
type Selector struct {
	Children []Node
	name     string
}

func NewSelector(name string, children ...Node) *Selector {
	return &Selector{Children: children, name: name}
}

func (s *Selector) Name() string {
	return fmt.Sprintf("Selector(%s)", s.name)
}

func (s *Selector) Evaluate(agent *Agent, game *Game) NodeState {
	fmt.Fprintln(os.Stderr, fmt.Sprintf("🌳 Agent %d evaluating %s with %d children",
		agent.ID, s.Name(), len(s.Children)))

	for i, child := range s.Children {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("  ├─ Agent %d trying child %d: %s",
			agent.ID, i, child.Name()))

		switch child.Evaluate(agent, game) {
		case BTSuccess:
			fmt.Fprintln(os.Stderr, fmt.Sprintf("  ✅ Agent %d: %s succeeded",
				agent.ID, child.Name()))
			return BTSuccess // One child succeeded, selector succeeds
		case BTRunning:
			fmt.Fprintln(os.Stderr, fmt.Sprintf("  ⏳ Agent %d: %s running",
				agent.ID, child.Name()))
			return BTRunning // One child is running, selector is running
		case BTFailure:
			fmt.Fprintln(os.Stderr, fmt.Sprintf("  ❌ Agent %d: %s failed",
				agent.ID, child.Name()))
			continue // Child failed, try next child
		}
	}
	fmt.Fprintln(os.Stderr, fmt.Sprintf("  💔 Agent %d: All children of %s failed",
		agent.ID, s.Name()))
	return BTFailure // All children failed
}

// ============================================================================
// BASIC GAME STRUCTURES (from original)
// ============================================================================

// Tile represents a single grid tile
type Tile struct {
	X, Y int
	Type int // 0=empty, 1=low cover, 2=high cover
}

// Agent represents an agent with all its properties
type Agent struct {
	// Static properties (from initialization)
	ID             int
	Player         int
	ShootCooldown  int // Base cooldown between shots
	OptimalRange   int
	SoakingPower   int
	MaxSplashBombs int

	// Dynamic properties (updated each turn)
	X           int
	Y           int
	Cooldown    int // Current cooldown remaining
	SplashBombs int
	Wetness     int

	// AI State (new)
	CurrentTacticalState AgentTacticalState
	PersistentPath       []Point // Path being followed
	CurrentPathIndex     int     // Current step in path
	TargetX, TargetY     int     // Current movement target
	LastTargetID         int     // Last enemy targeted
	StateTimer           int     // How long in current state
}

// Point represents a coordinate
type Point struct {
	X, Y int
}

// Game holds the entire game state
type Game struct {
	MyID     int
	Grid     [][]Tile
	Width    int
	Height   int
	Agents   map[int]*Agent
	MyAgents []*Agent

	// AI Strategy (new)
	TeamStrategy    *TeamCoordinationStrategy
	AgentActions    map[int][]AgentAction // Collected actions for this turn
	TurnNumber      int
	TerritoryScores TerritoryScore // Cached territory calculation
}

// TerritoryScore holds territory control evaluation
type TerritoryScore struct {
	FriendlyTiles int
	EnemyTiles    int
	Contested     int
	Advantage     int // FriendlyTiles - EnemyTiles
}

// ============================================================================
// ACTION SYSTEM (from original, simplified)
// ============================================================================

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
	PriorityCombat    = 50  // Shooting/bombing
	PriorityMovement  = 30  // Positioning
	PriorityDefault   = 10  // Hunker down
)

// ============================================================================
// TEAM COORDINATION STRATEGY (Enhanced with FSM)
// ============================================================================

// TeamCoordinationStrategy (Enhanced with FSM)
type TeamCoordinationStrategy struct {
	CurrentTeamState TeamStrategyState
	StateTimer       int                // How long in current state
	LastStateChange  int                // Turn when state last changed
	Config           TeamStrategyConfig // Configurable thresholds

	// Cached computations for performance (recalculated each turn)
	TerritoryCache       TerritoryScore
	TerritoryCacheValid  bool
	EnemyThreatCache     map[int]float64 // Agent ID -> threat level
	ThreatCacheValid     bool
	TeamHealthCache      float64 // Average team health (100 - average wetness)
	HealthCacheValid     bool
	EnemyCountCache      int // Living enemy count
	EnemyCountCacheValid bool
}

func NewTeamCoordinationStrategy() *TeamCoordinationStrategy {
	return &TeamCoordinationStrategy{
		CurrentTeamState: TeamStateCombat, // Default starting strategy
		Config:           DefaultTeamConfig,
		EnemyThreatCache: make(map[int]float64),
	}
}

func (s *TeamCoordinationStrategy) Name() string {
	return fmt.Sprintf("TeamCoordination_%s", s.CurrentTeamState.String())
}

// ============================================================================
// CONFIGURABLE AI PARAMETERS
// ============================================================================

// TeamStrategyConfig holds configurable parameters for team strategy
type TeamStrategyConfig struct {
	// Health thresholds (wetness levels)
	FleeWetnessThreshold   int // Switch to regroup when average wetness > this
	ReturnWetnessThreshold int // Return to combat when average wetness < this

	// Territory thresholds
	TerritoryDeficitLimit   int // Switch to territory control when behind by this much
	TerritoryAdvantageLimit int // Switch to combat when ahead by this much

	// Turn-based thresholds
	LateGameTurnThreshold int // Turn after which territory becomes more important

	// Enemy count thresholds
	FewEnemiesThreshold int // Switch to combat when enemies <= this number

	// State persistence timing
	MinStateTime int // Minimum turns to stay in a state (prevents rapid switching)

	// Health thresholds for RegroupAndHeal state
	RecoverHealthThreshold int // Health threshold to switch to Combat state
}

// Default configuration with moderate aggressiveness
var DefaultTeamConfig = TeamStrategyConfig{
	FleeWetnessThreshold:    60, // Moderate: flee when average wetness > 60
	ReturnWetnessThreshold:  40, // Moderate: return when average wetness < 40
	TerritoryDeficitLimit:   15, // Switch to territory when 15+ tiles behind
	TerritoryAdvantageLimit: 20, // Switch to combat when 20+ tiles ahead
	LateGameTurnThreshold:   70, // After turn 70, prioritize territory more
	FewEnemiesThreshold:     2,  // Aggressive combat when <= 2 enemies
	MinStateTime:            3,  // Stay in state for at least 3 turns
	RecoverHealthThreshold:  70, // Health threshold to switch to Combat state
}

// ============================================================================
// MAIN FUNCTION AND GAME LOOP
// ============================================================================

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1000000), 1000000)

	// Initialize game state
	game := NewGame()

	// Read initial game data (same as original)
	initializeGame(scanner, game)

	// Print the loaded map for context
	game.PrintMap()

	fmt.Fprintln(os.Stderr, "🤖 Starting New AI System:", game.TeamStrategy.Name())

	// Main game loop
	for {
		game.TurnNumber++

		// Read turn input
		readTurnInput(scanner, game)

		// Update AI and coordinate actions
		actions := game.CoordinateActions()

		// Output actions
		outputActions(game, actions)
	}
}

// NewGame creates a new game instance
func NewGame() *Game {
	return &Game{
		Agents:       make(map[int]*Agent),
		MyAgents:     make([]*Agent, 0),
		AgentActions: make(map[int][]AgentAction),
		TeamStrategy: NewTeamCoordinationStrategy(),
	}
}

// Helper function
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ============================================================================
// PLACEHOLDER IMPLEMENTATIONS (TO BE COMPLETED)
// ============================================================================

// Initialize game from input
func initializeGame(scanner *bufio.Scanner, game *Game) {
	// myId: Your player id (0 or 1)
	scanner.Scan()
	inputLine := scanner.Text()
	if CAPTURE_INPUT_FORMAT {
		fmt.Fprintln(os.Stderr, "CAPTURED INPUT:", inputLine)
	}
	fmt.Sscan(inputLine, &game.MyID)

	// agentCount: Total number of agents in the game
	var agentCount int
	scanner.Scan()
	inputLine = scanner.Text()
	if CAPTURE_INPUT_FORMAT {
		fmt.Fprintln(os.Stderr, "CAPTURED INPUT:", inputLine)
	}
	fmt.Sscan(inputLine, &agentCount)

	for i := 0; i < agentCount; i++ {
		var agentId, player, shootCooldown, optimalRange, soakingPower, splashBombs int
		scanner.Scan()
		inputLine = scanner.Text()
		if CAPTURE_INPUT_FORMAT {
			fmt.Fprintln(os.Stderr, "CAPTURED INPUT:", inputLine)
		}
		fmt.Sscan(inputLine, &agentId, &player, &shootCooldown, &optimalRange, &soakingPower, &splashBombs)

		// Store agent data with AI state initialization
		agent := &Agent{
			ID:             agentId,
			Player:         player,
			ShootCooldown:  shootCooldown,
			OptimalRange:   optimalRange,
			SoakingPower:   soakingPower,
			MaxSplashBombs: splashBombs,
			// Initialize AI state
			CurrentTacticalState: AgentStateIdle,
			PersistentPath:       make([]Point, 0),
			CurrentPathIndex:     0,
			TargetX:              -1,
			TargetY:              -1,
			LastTargetID:         -1,
			StateTimer:           0,
		}

		game.Agents[agentId] = agent
		if player == game.MyID {
			game.MyAgents = append(game.MyAgents, agent)
		}
	}

	// width: Width of the game map
	// height: Height of the game map
	scanner.Scan()
	inputLine = scanner.Text()
	if CAPTURE_INPUT_FORMAT {
		fmt.Fprintln(os.Stderr, "CAPTURED INPUT:", inputLine)
	}
	fmt.Sscan(inputLine, &game.Width, &game.Height)

	// Initialize and read grid
	game.Grid = make([][]Tile, game.Height)
	for i := 0; i < game.Height; i++ {
		game.Grid[i] = make([]Tile, game.Width)
	}

	for i := 0; i < game.Height; i++ {
		scanner.Scan()
		inputLine = scanner.Text()
		if CAPTURE_INPUT_FORMAT {
			fmt.Fprintln(os.Stderr, "CAPTURED INPUT:", inputLine)
		}
		inputs := strings.Split(inputLine, " ")
		for j := 0; j < game.Width; j++ {
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

	if CAPTURE_INPUT_FORMAT {
		fmt.Fprintln(os.Stderr, "CAPTURED: Game initialization complete")
	}
}

// Read turn input data
func readTurnInput(scanner *bufio.Scanner, game *Game) {
	var agentCount int
	scanner.Scan()
	inputLine := scanner.Text()
	if CAPTURE_INPUT_FORMAT && game.TurnNumber <= 3 {
		fmt.Fprintln(os.Stderr, "CAPTURED TURN", game.TurnNumber, "INPUT:", inputLine)
	}
	fmt.Sscan(inputLine, &agentCount)

	// Clear current agent list - only keep agents that exist this turn
	currentAgents := make(map[int]*Agent)
	game.MyAgents = make([]*Agent, 0)

	for i := 0; i < agentCount; i++ {
		var agentId, x, y, cooldown, splashBombs, wetness int
		scanner.Scan()
		inputLine = scanner.Text()
		if CAPTURE_INPUT_FORMAT && game.TurnNumber <= 3 {
			fmt.Fprintln(os.Stderr, "CAPTURED TURN", game.TurnNumber, "AGENT INPUT:", inputLine)
		}
		fmt.Sscan(inputLine, &agentId, &x, &y, &cooldown, &splashBombs, &wetness)

		// Get agent from previous turn (to keep static properties and AI state)
		if existingAgent, exists := game.Agents[agentId]; exists {
			// Update dynamic properties
			existingAgent.X = x
			existingAgent.Y = y
			existingAgent.Cooldown = cooldown
			existingAgent.SplashBombs = splashBombs
			existingAgent.Wetness = wetness

			// Update AI state timers
			existingAgent.StateTimer++

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

	// Clear state for eliminated agents (proper state persistence)
	eliminatedAgents := []int{}
	for agentID := range game.TeamStrategy.EnemyThreatCache {
		if _, stillAlive := currentAgents[agentID]; !stillAlive {
			eliminatedAgents = append(eliminatedAgents, agentID)
		}
	}
	for _, agentID := range eliminatedAgents {
		delete(game.TeamStrategy.EnemyThreatCache, agentID)
		fmt.Fprintln(os.Stderr, fmt.Sprintf("🗑️  Cleared state for eliminated agent %d", agentID))
	}

	// Clear cached computations for new turn (recalculate fresh data)
	game.TeamStrategy.TerritoryCacheValid = false
	game.TeamStrategy.ThreatCacheValid = false
	game.TeamStrategy.HealthCacheValid = false
	game.TeamStrategy.EnemyCountCacheValid = false

	fmt.Fprintln(os.Stderr, fmt.Sprintf("Turn %d: %d total agents, %d mine",
		game.TurnNumber, len(game.Agents), len(game.MyAgents)))

	// Read myAgentCount (required by protocol but we calculate it ourselves)
	var myAgentCount int
	scanner.Scan()
	inputLine = scanner.Text()
	if CAPTURE_INPUT_FORMAT && game.TurnNumber <= 3 {
		fmt.Fprintln(os.Stderr, "CAPTURED TURN", game.TurnNumber, "MY_AGENT_COUNT:", inputLine)
	}
	fmt.Sscan(inputLine, &myAgentCount)
}

// Main action coordination using Team FSM + Behavior Trees
func (g *Game) CoordinateActions() map[int][]AgentAction {
	// Clear previous turn's actions
	g.AgentActions = make(map[int][]AgentAction)

	// Step 1: Update team strategy state
	g.TeamStrategy.UpdateTeamState(g)

	// Step 2: For each agent, evaluate their behavior tree based on team strategy
	for _, agent := range g.MyAgents {
		// Get the appropriate behavior tree for current team strategy
		var behaviorTree Node
		switch g.TeamStrategy.CurrentTeamState {
		case TeamStateCombat:
			behaviorTree = g.BuildCombatBT()
		case TeamStateTerritoryControl:
			behaviorTree = g.BuildTerritoryBT()
		case TeamStateRegroupAndHeal:
			behaviorTree = g.BuildRegroupBT()
		case TeamStateDefense:
			behaviorTree = g.BuildDefenseBT()
		default:
			behaviorTree = g.BuildDefaultBT()
		}

		// Evaluate the behavior tree for this agent
		g.AgentActions[agent.ID] = make([]AgentAction, 0)
		result := behaviorTree.Evaluate(agent, g)

		// If no actions were generated, add default hunker
		if len(g.AgentActions[agent.ID]) == 0 {
			g.AgentActions[agent.ID] = append(g.AgentActions[agent.ID], AgentAction{
				Type:     ActionHunker,
				Priority: PriorityDefault,
				Reason:   "Default action - no BT actions generated",
			})
		}

		fmt.Fprintln(os.Stderr, fmt.Sprintf("Agent %d [%s/%s]: BT=%s, %d actions",
			agent.ID, g.TeamStrategy.CurrentTeamState.String(), agent.CurrentTacticalState.String(),
			result.String(), len(g.AgentActions[agent.ID])))
	}

	// Step 3: Resolve action conflicts (movement collisions, etc.)
	return g.resolveActionConflicts(g.AgentActions)
}

// Output actions in correct format
func outputActions(game *Game, actions map[int][]AgentAction) {
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

		// Output to game engine
		fmt.Printf("%d; %s\n", agent.ID, actionStr)
	}
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

// Print map layout
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

// resolveActionConflicts prevents movement collisions and handles priority conflicts
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

			fmt.Fprintln(os.Stderr, fmt.Sprintf("✅ Agent %d moving to (%d,%d) [priority %d]", agentID, action.TargetX, action.TargetY, action.Priority))
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

				fmt.Fprintln(os.Stderr, fmt.Sprintf("🔄 Agent %d taking alternative move to (%d,%d) [wanted (%d,%d)]",
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

				fmt.Fprintln(os.Stderr, fmt.Sprintf("⚠️  Agent %d staying put at (%d,%d) - no alternatives", agentID, agent.X, agent.Y))
			}
		}
	}

	return resolvedActions
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
	safetyValue := g.CalculatePositionSafety(candidateX, candidateY)
	score += safetyValue * 0.1

	return score
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

// ============================================================================
// TEAM STRATEGY METHODS (TO BE IMPLEMENTED)
// ============================================================================

// UpdateTeamState updates the team's strategic state based on game conditions
func (s *TeamCoordinationStrategy) UpdateTeamState(game *Game) {
	s.StateTimer++

	// Calculate key metrics (cached for performance)
	teamHealth := s.GetTeamHealth(game)
	territoryScore := s.GetTerritoryScore(game)
	enemyCount := s.GetEnemyCount(game)
	isLateGame := game.TurnNumber > s.Config.LateGameTurnThreshold

	fmt.Fprintln(os.Stderr, fmt.Sprintf("🧠 Team Analysis: Health=%.1f, Territory=%+d, Enemies=%d, Turn=%d, State=%s(%d)",
		teamHealth, territoryScore.Advantage, enemyCount, game.TurnNumber, s.CurrentTeamState.String(), s.StateTimer))

	// Prevent rapid state switching - require minimum time in state
	if s.StateTimer < s.Config.MinStateTime {
		return
	}

	previousState := s.CurrentTeamState

	// FSM Transition Logic based on priorities
	switch s.CurrentTeamState {

	case TeamStateCombat:
		// Combat → RegroupAndHeal: Team health is low
		if teamHealth < float64(s.Config.FleeWetnessThreshold) {
			s.transitionToState(TeamStateRegroupAndHeal, game.TurnNumber,
				fmt.Sprintf("Low team health: %.1f < %d", teamHealth, s.Config.FleeWetnessThreshold))

			// Combat → TerritoryControl: Losing territory in late game OR severely behind (less aggressive)
		} else if (isLateGame && territoryScore.Advantage < -s.Config.TerritoryDeficitLimit) ||
			territoryScore.Advantage < -s.Config.TerritoryDeficitLimit*4 { // Much higher threshold
			s.transitionToState(TeamStateTerritoryControl, game.TurnNumber,
				fmt.Sprintf("Territory deficit: %d (late=%v)", territoryScore.Advantage, isLateGame))
		}

	case TeamStateTerritoryControl:
		// TerritoryControl → RegroupAndHeal: Team health is low
		if teamHealth < float64(s.Config.FleeWetnessThreshold) {
			s.transitionToState(TeamStateRegroupAndHeal, game.TurnNumber,
				fmt.Sprintf("Low team health: %.1f < %d", teamHealth, s.Config.FleeWetnessThreshold))

			// TerritoryControl → Combat: Few enemies left OR big territory advantage
		} else if enemyCount <= s.Config.FewEnemiesThreshold ||
			territoryScore.Advantage > s.Config.TerritoryAdvantageLimit {
			s.transitionToState(TeamStateCombat, game.TurnNumber,
				fmt.Sprintf("Combat opportunity: enemies=%d, territory=%+d", enemyCount, territoryScore.Advantage))
		}

	case TeamStateRegroupAndHeal:
		// RegroupAndHeal → Combat: Team recovered OR last stand situation
		if teamHealth > float64(s.Config.RecoverHealthThreshold) &&
			len(game.MyAgents) >= enemyCount-1 { // Not critically outnumbered
			s.transitionToState(TeamStateCombat, game.TurnNumber,
				fmt.Sprintf("Team recovered: health=%.1f, enemies=%d, territory=%+d", teamHealth, enemyCount, territoryScore.Advantage))

			// LAST STAND: If only 1 agent left OR critically outnumbered (2:1 ratio) or very late game
		} else if len(game.MyAgents) == 1 || len(game.MyAgents)*2 <= enemyCount || game.TurnNumber >= 80 {
			s.transitionToState(TeamStateCombat, game.TurnNumber,
				fmt.Sprintf("Last stand: %d vs %d enemies, turn %d", len(game.MyAgents), enemyCount, game.TurnNumber))

			// RegroupAndHeal → TerritoryControl: If safe but losing territory
		} else if teamHealth > 50.0 && territoryScore.Advantage < -20 {
			s.transitionToState(TeamStateTerritoryControl, game.TurnNumber,
				fmt.Sprintf("Safe territory push: health=%.1f, territory=%+d", teamHealth, territoryScore.Advantage))
		}

	case TeamStateDefense:
		// Defense → RegroupAndHeal: Team health is critically low
		if teamHealth < float64(s.Config.FleeWetnessThreshold-10) { // Even lower threshold for defense
			s.transitionToState(TeamStateRegroupAndHeal, game.TurnNumber,
				fmt.Sprintf("Critical health from defense: %.1f", teamHealth))

			// Defense → Combat: Threat passed, can attack
		} else if teamHealth > float64(s.Config.ReturnWetnessThreshold+10) &&
			enemyCount <= s.Config.FewEnemiesThreshold {
			s.transitionToState(TeamStateCombat, game.TurnNumber,
				fmt.Sprintf("Threat passed, attacking: health=%.1f, enemies=%d", teamHealth, enemyCount))
		}
	}

	// Log state changes
	if previousState != s.CurrentTeamState {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("🔄 TEAM STATE CHANGE: %s → %s",
			previousState.String(), s.CurrentTeamState.String()))
	}
}

// transitionToState handles state transitions with proper bookkeeping
func (s *TeamCoordinationStrategy) transitionToState(newState TeamStrategyState, turnNumber int, reason string) {
	s.CurrentTeamState = newState
	s.StateTimer = 0
	s.LastStateChange = turnNumber
	fmt.Fprintln(os.Stderr, fmt.Sprintf("🎯 State Transition: %s (Reason: %s)", newState.String(), reason))
}

// GetTeamHealth calculates average team health (cached)
func (s *TeamCoordinationStrategy) GetTeamHealth(game *Game) float64 {
	if s.HealthCacheValid {
		return s.TeamHealthCache
	}

	if len(game.MyAgents) == 0 {
		s.TeamHealthCache = 0
	} else {
		totalWetness := 0
		for _, agent := range game.MyAgents {
			totalWetness += agent.Wetness
		}
		avgWetness := float64(totalWetness) / float64(len(game.MyAgents))
		s.TeamHealthCache = 100.0 - avgWetness // Convert wetness to health
	}

	s.HealthCacheValid = true
	return s.TeamHealthCache
}

// GetTerritoryScore calculates territory control (cached)
func (s *TeamCoordinationStrategy) GetTerritoryScore(game *Game) TerritoryScore {
	if s.TerritoryCacheValid {
		return s.TerritoryCache
	}

	friendlyTiles := 0
	enemyTiles := 0
	contested := 0

	for y := 0; y < game.Height; y++ {
		for x := 0; x < game.Width; x++ {
			// Skip impassable tiles
			if game.Grid[y][x].Type > 0 {
				continue
			}

			closestFriendly := 999
			closestEnemy := 999

			// Find closest friendly agent
			for _, agent := range game.MyAgents {
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
			for _, agent := range game.Agents {
				if agent.Player != game.MyID && agent.Wetness < 100 {
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

	s.TerritoryCache = TerritoryScore{
		FriendlyTiles: friendlyTiles,
		EnemyTiles:    enemyTiles,
		Contested:     contested,
		Advantage:     friendlyTiles - enemyTiles,
	}

	s.TerritoryCacheValid = true
	return s.TerritoryCache
}

// GetEnemyCount calculates living enemy count (cached)
func (s *TeamCoordinationStrategy) GetEnemyCount(game *Game) int {
	if s.EnemyCountCacheValid {
		return s.EnemyCountCache
	}

	count := 0
	for _, agent := range game.Agents {
		if agent.Player != game.MyID && agent.Wetness < 100 {
			count++
		}
	}

	s.EnemyCountCache = count
	s.EnemyCountCacheValid = true
	return s.EnemyCountCache
}

// ============================================================================
// BEHAVIOR TREE BUILDERS (TO BE IMPLEMENTED)
// ============================================================================

// BuildCombatBT creates a combat-focused behavior tree
func (g *Game) BuildCombatBT() Node {
	// Combat behavior tree: Survival -> Shooting -> Bombing -> Advance -> Cover (reduced hunkering)
	return &Selector{
		name: "Combat",
		Children: []Node{
			// Priority 1: Survival (high wetness)
			&Sequence{
				name: "Survival",
				Children: []Node{
					&CheckWetnessHigh{Threshold: 70},
					&TaskMoveToSafety{},
				},
			},
			// Priority 2: Shooting (HIGHER PRIORITY than bombing)
			&Sequence{
				name: "Shooting",
				Children: []Node{
					&CheckCanShoot{},
					&TaskShootBestTarget{},
				},
			},
			// Priority 3: Bombing (conservative)
			&Sequence{
				name: "Bombing",
				Children: []Node{
					&CheckHasBombs{},
					&TaskThrowOptimalBomb{},
				},
			},
			// Priority 4: Advance toward enemies (when out of shooting range)
			&TaskMoveTowardsEnemies{},
			// Priority 5: Cover (last resort)
			&Selector{
				name: "Positioning",
				Children: []Node{
					&TaskMoveToCover{},
				},
			},
		},
	}
}

// BuildTerritoryBT creates a territory-control behavior tree
func (g *Game) BuildTerritoryBT() Node {
	return NewSelector("Territory",
		// Priority 1: Survival (flee if low health)
		NewSequence("Survival",
			NewCheckWetnessHigh(70),
			&TaskMoveToSafety{},
		),
		// Priority 2: Opportunistic shooting
		NewSequence("OpportunisticShooting",
			&CheckCanShoot{},
			NewCheckEnemiesInRange(4), // Only shoot very close enemies
			&TaskShootBestTarget{},
		),
		// Priority 3: Territory capture
		&TaskMoveToTerritory{},
		// Priority 4: Default
		&TaskHunkerDown{},
	)
}

// BuildRegroupBT creates a regroup-and-heal behavior tree
func (g *Game) BuildRegroupBT() Node {
	return NewSelector("Regroup",
		// Priority 1: Move to safety
		&TaskMoveToSafety{},
		// Priority 2: Find cover
		&TaskMoveToCover{},
		// Priority 3: Default defensive
		&TaskHunkerDown{},
	)
}

// BuildDefenseBT creates a defensive behavior tree
func (g *Game) BuildDefenseBT() Node {
	return NewSelector("Defense",
		// Priority 1: Survival (flee if critical health)
		NewSequence("CriticalSurvival",
			NewCheckWetnessHigh(80),
			&TaskMoveToSafety{},
		),
		// Priority 2: Defensive shooting
		NewSequence("DefensiveShooting",
			&CheckCanShoot{},
			NewCheckEnemiesInRange(6), // Shoot nearby threats
			&TaskShootBestTarget{},
		),
		// Priority 3: Hold position with cover
		NewSelector("HoldPosition",
			&TaskMoveToCover{},
			&TaskHunkerDown{},
		),
	)
}

// BuildDefaultBT creates a default fallback behavior tree
func (g *Game) BuildDefaultBT() Node {
	return NewSelector("Default",
		// Priority 1: Basic survival
		NewSequence("BasicSurvival",
			NewCheckWetnessHigh(50),
			&TaskMoveToSafety{},
		),
		// Priority 2: Basic shooting
		NewSequence("BasicShooting",
			&CheckCanShoot{},
			&TaskShootBestTarget{},
		),
		// Priority 3: Default action
		&TaskHunkerDown{},
	)
}

// ============================================================================
// BASIC BEHAVIOR TREE TASK NODES (PLACEHOLDERS)
// ============================================================================

// ============================================================================
// CONDITION TASKS (Check game state)
// ============================================================================

// CheckWetnessHigh - Check if agent's wetness is above danger threshold
type CheckWetnessHigh struct {
	Threshold int
}

func NewCheckWetnessHigh(threshold int) *CheckWetnessHigh {
	return &CheckWetnessHigh{Threshold: threshold}
}

func (c *CheckWetnessHigh) Name() string {
	return fmt.Sprintf("CheckWetnessHigh(%d)", c.Threshold)
}

func (c *CheckWetnessHigh) Evaluate(agent *Agent, game *Game) NodeState {
	if agent.Wetness >= c.Threshold {
		return BTSuccess
	}
	return BTFailure
}

// CheckCanShoot - Check if agent can shoot (cooldown is 0)
type CheckCanShoot struct{}

func (c *CheckCanShoot) Name() string {
	return "CheckCanShoot"
}

func (c *CheckCanShoot) Evaluate(agent *Agent, game *Game) NodeState {
	if agent.Cooldown == 0 {
		return BTSuccess
	}
	return BTFailure
}

// CheckHasBombs - Check if agent has splash bombs available
type CheckHasBombs struct{}

func (c *CheckHasBombs) Name() string {
	return "CheckHasBombs"
}

func (c *CheckHasBombs) Evaluate(agent *Agent, game *Game) NodeState {
	if agent.SplashBombs > 0 {
		return BTSuccess
	}
	return BTFailure
}

// CheckEnemiesInRange - Check if there are enemies within specified range
type CheckEnemiesInRange struct {
	Range int
}

func NewCheckEnemiesInRange(maxRange int) *CheckEnemiesInRange {
	return &CheckEnemiesInRange{Range: maxRange}
}

func (c *CheckEnemiesInRange) Name() string {
	return fmt.Sprintf("CheckEnemiesInRange(%d)", c.Range)
}

func (c *CheckEnemiesInRange) Evaluate(agent *Agent, game *Game) NodeState {
	for _, enemy := range game.Agents {
		if enemy.Player != game.MyID && enemy.Wetness < 100 {
			distance := abs(agent.X-enemy.X) + abs(agent.Y-enemy.Y)
			if distance <= c.Range {
				return BTSuccess
			}
		}
	}
	return BTFailure
}

// CheckHasCover - Check if agent is adjacent to cover
type CheckHasCover struct{}

func (c *CheckHasCover) Name() string {
	return "CheckHasCover"
}

func (c *CheckHasCover) Evaluate(agent *Agent, game *Game) NodeState {
	if game.GetMaxAdjacentCover(agent.X, agent.Y) > 0 {
		return BTSuccess
	}
	return BTFailure
}

// ============================================================================
// ACTION TASKS (Perform game actions)
// ============================================================================

// TaskShootBestTarget - Shoot the best available target (with better logging)
type TaskShootBestTarget struct{}

func (t *TaskShootBestTarget) Name() string {
	return "TaskShootBestTarget"
}

func (t *TaskShootBestTarget) Evaluate(agent *Agent, game *Game) NodeState {
	target := game.FindBestShootTarget(agent)
	if target != nil {
		distance := abs(agent.X-target.X) + abs(agent.Y-target.Y)
		action := AgentAction{
			Type:          ActionShoot,
			TargetAgentID: target.ID,
			Priority:      PriorityCombat,
			Reason:        fmt.Sprintf("Shooting enemy %d (dist %d, wetness %d)", target.ID, distance, target.Wetness),
		}
		game.AgentActions[agent.ID] = append(game.AgentActions[agent.ID], action)
		fmt.Fprintln(os.Stderr, fmt.Sprintf("🎯 Agent %d: shooting target %d", agent.ID, target.ID))
		return BTSuccess
	}

	fmt.Fprintln(os.Stderr, fmt.Sprintf("🔫 Agent %d: no valid shooting targets", agent.ID))
	return BTFailure
}

// TaskThrowOptimalBomb - Throw bomb at optimal position (with better logging)
type TaskThrowOptimalBomb struct{}

func (t *TaskThrowOptimalBomb) Name() string {
	return "TaskThrowOptimalBomb"
}

func (t *TaskThrowOptimalBomb) Evaluate(agent *Agent, game *Game) NodeState {
	if agent.SplashBombs <= 0 {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("💣 Agent %d: no bombs left", agent.ID))
		return BTFailure
	}

	bombX, bombY, score := game.FindOptimalBombTarget(agent)
	if score > 25.0 { // Higher threshold with escape prediction
		action := AgentAction{
			Type:     ActionThrow,
			TargetX:  bombX,
			TargetY:  bombY,
			Priority: PriorityCombat,
			Reason:   fmt.Sprintf("Bombing (%d,%d) score %.0f", bombX, bombY, score),
		}
		game.AgentActions[agent.ID] = append(game.AgentActions[agent.ID], action)
		fmt.Fprintln(os.Stderr, fmt.Sprintf("💥 Agent %d: throwing bomb at (%d,%d) score %.0f",
			agent.ID, bombX, bombY, score))
		return BTSuccess
	}

	fmt.Fprintln(os.Stderr, fmt.Sprintf("💣 Agent %d: bomb score %.0f too low (need 25+), saving bomb", agent.ID, score))
	return BTFailure
}

// TaskMoveToSafety - Move agent to safest nearby position
type TaskMoveToSafety struct{}

func (t *TaskMoveToSafety) Name() string {
	return "TaskMoveToSafety"
}

func (t *TaskMoveToSafety) Evaluate(agent *Agent, game *Game) NodeState {
	safeX, safeY := game.FindSafetyPosition(agent)
	if safeX != agent.X || safeY != agent.Y {
		action := AgentAction{
			Type:     ActionMove,
			TargetX:  safeX,
			TargetY:  safeY,
			Priority: PriorityEmergency,
			Reason:   fmt.Sprintf("Moving to safety (%d,%d)", safeX, safeY),
		}
		game.AgentActions[agent.ID] = append(game.AgentActions[agent.ID], action)
		fmt.Fprintln(os.Stderr, fmt.Sprintf("🏃 Agent %d: moving to safety (%d,%d)", agent.ID, safeX, safeY))
		return BTSuccess
	}
	fmt.Fprintln(os.Stderr, fmt.Sprintf("🏃 Agent %d: already at safest position", agent.ID))
	return BTFailure
}

// TaskMoveToCover - Move to best cover position
type TaskMoveToCover struct{}

func (t *TaskMoveToCover) Name() string {
	return "TaskMoveToCover"
}

func (t *TaskMoveToCover) Evaluate(agent *Agent, game *Game) NodeState {
	// Check if there are enemies nearby - if so, prioritize combat over cover
	nearestEnemy := game.FindNearestEnemy(agent)
	if nearestEnemy != nil {
		enemyDistance := abs(agent.X-nearestEnemy.X) + abs(agent.Y-nearestEnemy.Y)
		// If enemies are close (within range 8), don't waste time seeking cover
		if enemyDistance <= 8 {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("🛡️  Agent %d: enemy too close (dist %d), skipping cover", agent.ID, enemyDistance))
			return BTFailure
		}
	}

	targetX, targetY := game.FindNearestCover(agent)

	// Don't move if we're already at good cover
	currentCover := game.GetMaxAdjacentCover(agent.X, agent.Y)
	targetCover := game.GetMaxAdjacentCover(targetX, targetY)

	if currentCover >= targetCover || (targetX == agent.X && targetY == agent.Y) {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("🛡️  Agent %d: already at good cover level %d, hunkering", agent.ID, currentCover))
		action := AgentAction{
			Type:     ActionHunker,
			Priority: PriorityDefault,
			Reason:   fmt.Sprintf("At cover level %d", currentCover),
		}
		game.AgentActions[agent.ID] = append(game.AgentActions[agent.ID], action)
		return BTSuccess
	}

	// Only move to cover if it's a significant improvement
	if targetCover <= currentCover+1 {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("🛡️  Agent %d: cover improvement too small (%d->%d), skipping", agent.ID, currentCover, targetCover))
		return BTFailure
	}

	action := AgentAction{
		Type:     ActionMove,
		TargetX:  targetX,
		TargetY:  targetY,
		Priority: PriorityMovement,
		Reason:   fmt.Sprintf("Moving to cover level %d", targetCover),
	}
	game.AgentActions[agent.ID] = append(game.AgentActions[agent.ID], action)
	fmt.Fprintln(os.Stderr, fmt.Sprintf("🛡️  Agent %d: moving to cover (%d,%d) level %d",
		agent.ID, targetX, targetY, targetCover))
	return BTSuccess
}

// TaskMoveToTerritory - Move agent to capture territory
type TaskMoveToTerritory struct{}

func (t *TaskMoveToTerritory) Name() string {
	return "TaskMoveToTerritory"
}

func (t *TaskMoveToTerritory) Evaluate(agent *Agent, game *Game) NodeState {
	targetX, targetY := game.FindTerritoryTarget(agent)

	// Accept any territory improvement move
	if targetX != agent.X || targetY != agent.Y {
		action := AgentAction{
			Type:     ActionMove,
			TargetX:  targetX,
			TargetY:  targetY,
			Priority: PriorityMovement,
			Reason:   fmt.Sprintf("Territory control to (%d,%d)", targetX, targetY),
		}
		game.AgentActions[agent.ID] = append(game.AgentActions[agent.ID], action)
		return BTSuccess
	}

	// If no territory move needed, try to support by positioning
	return BTFailure
}

// TaskMoveTowardsEnemies - Move agent towards nearest enemies (more aggressive)
type TaskMoveTowardsEnemies struct{}

func (t *TaskMoveTowardsEnemies) Name() string {
	return "TaskMoveTowardsEnemies"
}

func (t *TaskMoveTowardsEnemies) Evaluate(agent *Agent, game *Game) NodeState {
	nearestEnemy := game.FindNearestEnemy(agent)
	if nearestEnemy == nil {
		return BTFailure
	}

	distance := abs(agent.X-nearestEnemy.X) + abs(agent.Y-nearestEnemy.Y)

	// Only hunker if we're in optimal shooting range AND cooldown is short (1-2 turns)
	if distance <= agent.OptimalRange && agent.Cooldown > 0 && agent.Cooldown <= 2 {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("🎯 Agent %d: in optimal range %d, short cooldown %d, hunkering",
			agent.ID, agent.OptimalRange, agent.Cooldown))
		action := AgentAction{
			Type:     ActionHunker,
			Priority: PriorityDefault,
			Reason:   fmt.Sprintf("In optimal range %d, short cooldown %d", distance, agent.Cooldown),
		}
		game.AgentActions[agent.ID] = append(game.AgentActions[agent.ID], action)
		return BTSuccess
	}

	// If cooldown is long (3+ turns), always reposition for safety
	if agent.Cooldown >= 3 {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("🎯 Agent %d: long cooldown %d, repositioning for safety", agent.ID, agent.Cooldown))
		// Continue to movement logic below - will move to safety or better position
	} else if distance <= agent.OptimalRange {
		// In optimal range with short/no cooldown - this should have been handled by shooting logic
		fmt.Fprintln(os.Stderr, fmt.Sprintf("🎯 Agent %d: in optimal range %d but shooting failed, advancing", agent.ID))
		// Continue to movement logic below
	}

	// Move towards the nearest enemy or to safety if cooldown is high
	var targetX, targetY int
	if agent.Cooldown >= 3 {
		// Long cooldown - prioritize safety
		targetX, targetY = game.FindSafetyPosition(agent)
		if targetX == agent.X && targetY == agent.Y {
			// No safety position found, try tactical position
			targetX, targetY = game.FindTacticalPosition(agent, nearestEnemy.X, nearestEnemy.Y)
		}
	} else {
		// Short/no cooldown - advance tactically
		targetX, targetY = game.FindTacticalPosition(agent, nearestEnemy.X, nearestEnemy.Y)
	}

	if targetX == agent.X && targetY == agent.Y {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("🎯 Agent %d: no good position found, hunkering", agent.ID))
		action := AgentAction{
			Type:     ActionHunker,
			Priority: PriorityDefault,
			Reason:   "No good position found",
		}
		game.AgentActions[agent.ID] = append(game.AgentActions[agent.ID], action)
		return BTSuccess
	}

	reason := "Advancing toward enemy"
	if agent.Cooldown >= 3 {
		reason = "Repositioning for safety (long cooldown)"
	}

	action := AgentAction{
		Type:     ActionMove,
		TargetX:  targetX,
		TargetY:  targetY,
		Priority: PriorityMovement,
		Reason:   reason,
	}
	game.AgentActions[agent.ID] = append(game.AgentActions[agent.ID], action)
	fmt.Fprintln(os.Stderr, fmt.Sprintf("🎯 Agent %d: %s from (%d,%d) to (%d,%d)",
		agent.ID, reason, agent.X, agent.Y, targetX, targetY))
	return BTSuccess
}

// TaskHunkerDown - Simple task that makes agent hunker down
type TaskHunkerDown struct{}

func (t *TaskHunkerDown) Name() string {
	return "TaskHunkerDown"
}

func (t *TaskHunkerDown) Evaluate(agent *Agent, game *Game) NodeState {
	// Add hunker action to agent's action list
	action := AgentAction{
		Type:     ActionHunker,
		Priority: PriorityDefault,
		Reason:   "Default hunker action",
	}
	game.AgentActions[agent.ID] = append(game.AgentActions[agent.ID], action)
	return BTSuccess
}

// ============================================================================
// GAME HELPER METHODS (Supporting BT Tasks)
// ============================================================================

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

// IsValidPosition checks if a position is within grid bounds
func (g *Game) IsValidPosition(x, y int) bool {
	return x >= 0 && x < g.Width && y >= 0 && y < g.Height
}

// FindBestShootTarget finds the best enemy to shoot (CLOSEST PRIORITY)
func (g *Game) FindBestShootTarget(agent *Agent) *Agent {
	var bestTarget *Agent
	bestDistance := 999999
	bestScore := 0.0

	for _, enemy := range g.Agents {
		if enemy.Player == g.MyID || enemy.Wetness >= 100 {
			continue // Skip friendly and eliminated agents
		}

		distance := abs(agent.X-enemy.X) + abs(agent.Y-enemy.Y)

		// Check if in range (shooting range is up to OptimalRange*2)
		if distance > agent.OptimalRange*2 {
			continue
		}

		// PRIORITIZE CLOSEST ENEMY - distance is most important factor
		score := 1000.0 - float64(distance)*4.0 // Even heavier distance penalty (was *10)

		// PREFER WET ENEMIES (closer to elimination) - FIXED!
		score += float64(enemy.Wetness) // Higher wetness = higher score = prefer wet enemies

		// Big bonus for enemies very close to elimination
		if enemy.Wetness >= 80 {
			score += 100.0 // Massive bonus for finishing kills
		}

		fmt.Fprintln(os.Stderr, fmt.Sprintf("🎯 Agent %d evaluating enemy %d: dist=%d, wetness=%d, score=%.1f",
			agent.ID, enemy.ID, distance, enemy.Wetness, score))

		// ALWAYS prefer closer enemies
		if distance < bestDistance || (distance == bestDistance && score > bestScore) {
			bestTarget = enemy
			bestDistance = distance
			bestScore = score
		}
	}

	if bestTarget != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("🎯 Agent %d: closest target is %d at distance %d (score %.1f)",
			agent.ID, bestTarget.ID, bestDistance, bestScore))
	} else {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("🎯 Agent %d: no targets in range %d", agent.ID, agent.OptimalRange*2))
	}

	return bestTarget
}

// FindOptimalBombTarget finds the best position to throw a bomb (IMPROVED MULTI-TARGET)
func (g *Game) FindOptimalBombTarget(agent *Agent) (int, int, float64) {
	bestX, bestY := agent.X, agent.Y
	bestScore := 0.0

	// Search all positions within bomb range for optimal multi-enemy hits
	for dy := -4; dy <= 4; dy++ {
		for dx := -4; dx <= 4; dx++ {
			bombX := agent.X + dx
			bombY := agent.Y + dy

			// Check bomb distance from agent (Manhattan distance)
			bombDistance := abs(dx) + abs(dy)
			if bombDistance > 4 || bombDistance == 0 { // Bombs have range 4, can't bomb self
				continue
			}

			// Check if bomb position is valid
			if !g.IsValidPosition(bombX, bombY) {
				continue
			}

			// Calculate enemies hit in bomb splash area (3x3)
			score := 0.0
			enemiesHit := 0
			enemyDetails := []string{}

			for _, target := range g.Agents {
				if target.Player == g.MyID || target.Wetness >= 100 {
					continue // Skip friendly and eliminated
				}

				// Check if target is in bomb splash area (3x3 around bomb)
				targetDistance := abs(target.X-bombX) + abs(target.Y-bombY)
				if targetDistance <= 1 { // Manhattan distance 1 for 3x3 square
					// IMPROVED: Check if enemy can easily escape the bomb
					canEscape := g.CanEnemyEscapeBomb(target, bombX, bombY)
					if canEscape {
						// Apply penalty for easily escapable bombs
						damage := (100 - target.Wetness) / 2 // Half damage for escapable bombs
						score += float64(damage)
						enemiesHit++
						enemyDetails = append(enemyDetails, fmt.Sprintf("Agent%d(dist%d,wet%d,ESCAPABLE)",
							target.ID, targetDistance, target.Wetness))
					} else {
						// Full damage for trapped enemies
						damage := 100 - target.Wetness
						score += float64(damage)
						enemiesHit++
						enemyDetails = append(enemyDetails, fmt.Sprintf("Agent%d(dist%d,wet%d,TRAPPED)",
							target.ID, targetDistance, target.Wetness))
					}
				}
			}

			// Check for friendly fire
			friendlyDamage := 0
			for _, friendly := range g.MyAgents {
				if friendly.ID == agent.ID {
					continue
				}
				friendlyDistance := abs(friendly.X-bombX) + abs(friendly.Y-bombY)
				if friendlyDistance <= 1 {
					friendlyDamage += 50 // Heavy penalty for friendly fire
				}
			}

			// Apply friendly fire penalty
			score -= float64(friendlyDamage)

			// Multi-enemy bonus
			if enemiesHit >= 2 {
				score += float64(enemiesHit) * 25.0 // Bonus for hitting multiple enemies
			}

			// Log potential targets for debugging
			if enemiesHit > 0 {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("💣 Agent %d: bomb at (%d,%d) dist=%d hits %d enemies %v, score=%.0f (friendly_penalty=%d)",
					agent.ID, bombX, bombY, bombDistance, enemiesHit, enemyDetails, score, friendlyDamage))
			}

			// IMPROVED: Accept bombs that hit multiple enemies OR have high score
			if enemiesHit >= 2 || score >= 25.0 { // Higher threshold with escape prediction
				if score > bestScore {
					bestScore = score
					bestX, bestY = bombX, bombY
					fmt.Fprintln(os.Stderr, fmt.Sprintf("💣 Agent %d: NEW BEST bomb target (%d,%d) hits %d enemies, score %.0f",
						agent.ID, bestX, bestY, enemiesHit, score))
				}
			} else if enemiesHit > 0 {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("💣 Agent %d: REJECTED bomb at (%d,%d) - score %.0f too low or only %d enemies",
					agent.ID, bombX, bombY, score, enemiesHit))
			}
		}
	}

	if bestScore == 0.0 {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("💣 Agent %d: no valid bomb targets found that would hit enemies", agent.ID))
	} else {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("💣 Agent %d: FINAL best bomb target (%d,%d) score %.0f", agent.ID, bestX, bestY, bestScore))
	}

	return bestX, bestY, bestScore
}

// CanEnemyEscapeBomb checks if an enemy can easily move out of bomb blast area
func (g *Game) CanEnemyEscapeBomb(enemy *Agent, bombX, bombY int) bool {
	// Count escape routes (positions outside bomb blast area that enemy can reach)
	escapeRoutes := 0

	// Check all adjacent positions to the enemy
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue // Skip current position
			}

			escapeX := enemy.X + dx
			escapeY := enemy.Y + dy

			// Check if escape position is valid and passable
			if !g.IsValidPosition(escapeX, escapeY) || g.Grid[escapeY][escapeX].Type > 0 {
				continue
			}

			// Check if escape position is outside bomb blast area (3x3 around bomb)
			distanceFromBomb := abs(escapeX-bombX) + abs(escapeY-bombY)
			if distanceFromBomb > 1 { // Outside bomb splash area
				escapeRoutes++
			}
		}
	}

	// Enemy can escape if they have 2+ escape routes (good mobility)
	return escapeRoutes >= 2
}

// WouldImproveTerritory checks if moving to a position would improve territory control (FIXED)
func (g *Game) WouldImproveTerritory(x, y int, agent *Agent) bool {
	if !g.IsValidPosition(x, y) || g.Grid[y][x].Type > 0 {
		return false
	}

	// Simple territory improvement check
	// Count how many empty tiles this position could help control
	controlledTiles := 0

	// Check nearby tiles (within 3 distance)
	for dy := -3; dy <= 3; dy++ {
		for dx := -3; dx <= 3; dx++ {
			checkX, checkY := x+dx, y+dy
			if !g.IsValidPosition(checkX, checkY) || g.Grid[checkY][checkX].Type > 0 {
				continue
			}

			ourDistance := abs(dx) + abs(dy)
			if ourDistance > 6 { // Max control distance
				continue
			}

			// Find closest enemy to this tile
			closestEnemyDistance := 999
			for _, enemy := range g.Agents {
				if enemy.Player != g.MyID && enemy.Wetness < 100 {
					enemyDistance := abs(enemy.X-checkX) + abs(enemy.Y-checkY)
					if enemyDistance < closestEnemyDistance {
						closestEnemyDistance = enemyDistance
					}
				}
			}

			// If we'd be closer than enemies, we could control this tile
			if ourDistance < closestEnemyDistance {
				controlledTiles++
			}
		}
	}

	// Consider it an improvement if we can control at least 5 tiles from this position
	improvement := controlledTiles >= 5

	if improvement {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("🗺️  Agent %d position (%d,%d): controls %d tiles - IMPROVEMENT!",
			agent.ID, x, y, controlledTiles))
	}

	return improvement
}

// FindSafePosition finds the safest nearby position for an agent
func (g *Game) FindSafePosition(agent *Agent) (int, int) {
	bestX, bestY := agent.X, agent.Y
	bestSafety := g.CalculatePositionSafety(agent.X, agent.Y)

	// Search nearby positions
	for dy := -3; dy <= 3; dy++ {
		for dx := -3; dx <= 3; dx++ {
			newX, newY := agent.X+dx, agent.Y+dy
			if !g.IsValidPosition(newX, newY) || g.Grid[newY][newX].Type > 0 {
				continue
			}

			safety := g.CalculatePositionSafety(newX, newY)
			if safety > bestSafety {
				bestX, bestY = newX, newY
				bestSafety = safety
			}
		}
	}
	return bestX, bestY
}

// CalculatePositionSafety calculates how safe a position is
func (g *Game) CalculatePositionSafety(x, y int) float64 {
	safety := 100.0

	// Reduce safety based on enemy proximity
	for _, enemy := range g.Agents {
		if enemy.Player != g.MyID && enemy.Wetness < 100 {
			distance := abs(x-enemy.X) + abs(y-enemy.Y)
			threat := 0.0

			if distance <= enemy.OptimalRange {
				threat += 30.0
			}
			if distance <= 4 { // Bomb range
				threat += 20.0
			}

			safety -= threat / float64(distance+1)
		}
	}

	// Bonus for cover
	coverLevel := g.GetMaxAdjacentCover(x, y)
	safety += float64(coverLevel) * 15.0

	return safety
}

// FindNearestCover finds the nearest position adjacent to cover (with agent coordination)
func (g *Game) FindNearestCover(agent *Agent) (int, int) {
	bestX, bestY := agent.X, agent.Y
	bestDistance := 999
	bestCover := 0

	// Find all cover positions and score them
	type coverPosition struct {
		x, y, cover, distance int
		score                 float64
	}

	coverPositions := []coverPosition{}

	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			// Skip impassable tiles
			if g.Grid[y][x].Type > 0 {
				continue
			}

			coverLevel := g.GetMaxAdjacentCover(x, y)
			if coverLevel > 0 {
				distance := abs(agent.X-x) + abs(agent.Y-y)

				// STRONG agent spacing penalty
				crowdingPenalty := 0.0
				for _, otherAgent := range g.MyAgents {
					if otherAgent.ID == agent.ID || otherAgent.Wetness >= 100 {
						continue
					}
					otherDistance := abs(otherAgent.X-x) + abs(otherAgent.Y-y)
					if otherDistance == 0 {
						crowdingPenalty += 1000.0 // Massive penalty for same position
					} else if otherDistance == 1 {
						crowdingPenalty += 200.0 // Heavy penalty for adjacent positions
					} else if otherDistance == 2 {
						crowdingPenalty += 20.0 // Moderate penalty for close positions
					}
				}

				// Score: prioritize high cover, low distance, avoid crowding
				score := float64(coverLevel)*20.0 - float64(distance)*2.0 - crowdingPenalty

				coverPositions = append(coverPositions, coverPosition{
					x: x, y: y, cover: coverLevel, distance: distance, score: score,
				})
			}
		}
	}

	// Sort by score and pick the best available
	for i := 0; i < len(coverPositions); i++ {
		for j := i + 1; j < len(coverPositions); j++ {
			if coverPositions[i].score < coverPositions[j].score {
				coverPositions[i], coverPositions[j] = coverPositions[j], coverPositions[i]
			}
		}
	}

	// Pick the best position that's not the same as other agents' targets
	for _, pos := range coverPositions {
		// Check if any other agent is already targeting this position
		alreadyTargeted := false
		for _, otherAgent := range g.MyAgents {
			if otherAgent.ID != agent.ID && otherAgent.TargetX == pos.x && otherAgent.TargetY == pos.y {
				alreadyTargeted = true
				break
			}
		}

		if !alreadyTargeted {
			bestX, bestY = pos.x, pos.y
			bestCover = pos.cover
			bestDistance = pos.distance
			break
		}
	}

	// Update agent's target for coordination
	agent.TargetX, agent.TargetY = bestX, bestY

	fmt.Fprintln(os.Stderr, fmt.Sprintf("🛡️  Agent %d: cover target (%d,%d) level=%d dist=%d",
		agent.ID, bestX, bestY, bestCover, bestDistance))

	return bestX, bestY
}

// FindTerritoryTarget finds a good position for territory control (with agent coordination)
func (g *Game) FindTerritoryTarget(agent *Agent) (int, int) {
	bestX, bestY := agent.X, agent.Y
	bestScore := -999.0

	// Divide map into zones for different agents to avoid clustering
	agentZone := agent.ID % 4 // 0, 1, 2, 3 for different map quadrants

	zoneOffsetX := (agentZone % 2) * (g.Width / 2)
	zoneOffsetY := (agentZone / 2) * (g.Height / 2)
	zoneWidth := g.Width / 2
	zoneHeight := g.Height / 2

	fmt.Fprintln(os.Stderr, fmt.Sprintf("🗺️  Agent %d searching zone %d: x=%d-%d, y=%d-%d",
		agent.ID, agentZone, zoneOffsetX, zoneOffsetX+zoneWidth, zoneOffsetY, zoneOffsetY+zoneHeight))

	// Search the agent's assigned zone first, then expand if needed
	for expansion := 0; expansion <= 1; expansion++ {
		startX, endX := zoneOffsetX, zoneOffsetX+zoneWidth
		startY, endY := zoneOffsetY, zoneOffsetY+zoneHeight

		if expansion == 1 {
			// Second pass: search entire map if no good position in zone
			startX, endX = 0, g.Width
			startY, endY = 0, g.Height
		}

		for y := startY; y < endY; y++ {
			for x := startX; x < endX; x++ {
				if g.Grid[y][x].Type > 0 {
					continue // Skip walls
				}

				score := 0.0
				distance := abs(agent.X-x) + abs(agent.Y-y)

				// Penalty for distance (prefer closer positions)
				score -= float64(distance) * 1.0

				// Check how many tiles this position could control
				controlValue := g.CalculatePositionTerritoryValue(x, y)
				score += controlValue * 10.0

				// Bonus for cover nearby
				coverLevel := g.GetMaxAdjacentCover(x, y)
				score += float64(coverLevel) * 5.0

				// Coordination: avoid positions other agents are targeting
				crowdingPenalty := 0.0
				for _, otherAgent := range g.MyAgents {
					if otherAgent.ID != agent.ID {
						otherDist := abs(otherAgent.TargetX-x) + abs(otherAgent.TargetY-y)
						if otherDist <= 3 {
							crowdingPenalty += 20.0 / float64(otherDist+1)
						}
					}
				}
				score -= crowdingPenalty

				// Bonus for being in preferred zone
				if expansion == 0 {
					score += 30.0 // Zone preference bonus
				}

				// Safety consideration - avoid enemy-heavy areas
				enemyThreat := 0.0
				for _, enemy := range g.Agents {
					if enemy.Player != g.MyID && enemy.Wetness < 100 {
						enemyDist := abs(enemy.X-x) + abs(enemy.Y-y)
						if enemyDist <= 4 {
							enemyThreat += 10.0 / float64(enemyDist+1)
						}
					}
				}
				score -= enemyThreat

				// Prefer positions that contest enemy territory
				contestValue := 0.0
				for _, enemy := range g.Agents {
					if enemy.Player != g.MyID && enemy.Wetness < 100 {
						enemyDist := abs(enemy.X-x) + abs(enemy.Y-y)
						if enemyDist >= 3 && enemyDist <= 6 {
							contestValue += 15.0 // Good contesting distance
						}
					}
				}
				score += contestValue

				if score > bestScore {
					bestX, bestY = x, y
					bestScore = score
					fmt.Fprintln(os.Stderr, fmt.Sprintf("🎯 Agent %d NEW BEST (%d,%d): score=%.1f",
						agent.ID, x, y, score))
				}
			}
		}

		// If we found a position in our zone, use it (even if score is negative)
		// Only require a positive score if we're in the first expansion
		if bestScore > -500 && expansion == 0 {
			break
		}
		// On full map search, take the best we can find regardless of score
		if expansion == 1 {
			break
		}
	}

	// Update agent's target for coordination
	agent.TargetX, agent.TargetY = bestX, bestY

	fmt.Fprintln(os.Stderr, fmt.Sprintf("🎯 Agent %d: territory target (%d,%d) score=%.1f (zone %d)",
		agent.ID, bestX, bestY, bestScore, agentZone))

	return bestX, bestY
}

// FindNearestEnemy finds the closest enemy agent
func (g *Game) FindNearestEnemy(agent *Agent) *Agent {
	var nearestEnemy *Agent
	minDistance := 999

	for _, enemy := range g.Agents {
		if enemy.Player != g.MyID && enemy.Wetness < 100 {
			distance := abs(agent.X-enemy.X) + abs(agent.Y-enemy.Y)
			if distance < minDistance {
				nearestEnemy = enemy
				minDistance = distance
			}
		}
	}
	return nearestEnemy
}

// FindTacticalPosition finds a good tactical position relative to a target
func (g *Game) FindTacticalPosition(agent *Agent, targetX, targetY int) (int, int) {
	bestX, bestY := agent.X, agent.Y
	bestScore := -999.0

	// Search nearby positions
	searchRadius := 3
	for dy := -searchRadius; dy <= searchRadius; dy++ {
		for dx := -searchRadius; dx <= searchRadius; dx++ {
			newX, newY := agent.X+dx, agent.Y+dy

			// Skip invalid positions and stay within bounds
			if !g.IsValidPosition(newX, newY) || g.Grid[newY][newX].Type > 0 {
				continue
			}

			score := 0.0

			// Main goal: get closer to target
			currentDistanceToTarget := abs(agent.X-targetX) + abs(agent.Y-targetY)
			newDistanceToTarget := abs(newX-targetX) + abs(newY-targetY)

			if newDistanceToTarget < currentDistanceToTarget {
				score += float64(currentDistanceToTarget-newDistanceToTarget) * 20.0 // Big bonus for getting closer
			} else {
				score -= 10.0 // Penalty for moving away
			}

			// Bonus for cover
			coverLevel := g.GetMaxAdjacentCover(newX, newY)
			score += float64(coverLevel) * 15.0

			// STRONG agent spacing - enforce minimum distance of 1
			for _, friendly := range g.MyAgents {
				if friendly.ID != agent.ID && friendly.Wetness < 100 {
					friendlyDist := abs(friendly.X-newX) + abs(friendly.Y-newY)
					if friendlyDist == 0 {
						score -= 1000.0 // Massive penalty for same position
					} else if friendlyDist == 1 {
						score -= 200.0 // Heavy penalty for adjacent positions
					} else if friendlyDist == 2 {
						score -= 50.0 // Moderate penalty for close positions
					}
				}
			}

			// Safety consideration
			safetyPenalty := 0.0
			for _, enemy := range g.Agents {
				if enemy.Player != g.MyID && enemy.Wetness < 100 {
					enemyDist := abs(enemy.X-newX) + abs(enemy.Y-newY)
					if enemyDist <= 3 {
						safetyPenalty += 5.0 / float64(enemyDist+1)
					}
				}
			}
			score -= safetyPenalty

			if score > bestScore {
				bestX, bestY = newX, newY
				bestScore = score
			}
		}
	}

	return bestX, bestY
}

// FindSafetyPosition finds a safe position away from enemies with good cover
func (g *Game) FindSafetyPosition(agent *Agent) (int, int) {
	bestX, bestY := agent.X, agent.Y
	bestScore := -999.0

	// Search for positions within movement range
	searchRadius := 3
	for dy := -searchRadius; dy <= searchRadius; dy++ {
		for dx := -searchRadius; dx <= searchRadius; dx++ {
			newX, newY := agent.X+dx, agent.Y+dy

			// Skip invalid positions
			if !g.IsValidPosition(newX, newY) || g.Grid[newY][newX].Type > 0 {
				continue
			}

			score := 0.0

			// Prefer positions with cover
			coverLevel := g.GetMaxAdjacentCover(newX, newY)
			score += float64(coverLevel) * 20.0

			// Prefer positions farther from enemies
			minEnemyDistance := 999
			for _, enemy := range g.Agents {
				if enemy.Player != g.MyID && enemy.Wetness < 100 {
					distance := abs(newX-enemy.X) + abs(newY-enemy.Y)
					if distance < minEnemyDistance {
						minEnemyDistance = distance
					}
				}
			}
			score += float64(minEnemyDistance) * 5.0

			// STRONG agent spacing - maintain minimum distance while staying coordinated
			for _, friendly := range g.MyAgents {
				if friendly.ID != agent.ID && friendly.Wetness < 100 {
					distance := abs(newX-friendly.X) + abs(newY-friendly.Y)
					if distance == 0 {
						score -= 1000.0 // Massive penalty for same position
					} else if distance == 1 {
						score -= 200.0 // Heavy penalty for adjacent positions
					} else if distance == 2 {
						score += 5.0 // Small bonus for good spacing
					} else if distance == 3 {
						score += 10.0 // Bonus for staying coordinated but spaced
					}
				}
			}

			// Small penalty for distance from current position (don't move too far)
			movementDistance := abs(newX-agent.X) + abs(newY-agent.Y)
			score -= float64(movementDistance) * 1.0

			if score > bestScore {
				bestScore = score
				bestX, bestY = newX, newY
			}
		}
	}

	fmt.Fprintln(os.Stderr, fmt.Sprintf("🛡️  Agent %d: safety position (%d,%d) score %.1f",
		agent.ID, bestX, bestY, bestScore))
	return bestX, bestY
}
