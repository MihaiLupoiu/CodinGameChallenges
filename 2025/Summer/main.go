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

		// Assign targets for this challenge
		if agentId == 1 {
			agent.TargetX, agent.TargetY = 6, 1
		} else if agentId == 2 {
			agent.TargetX, agent.TargetY = 6, 3
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

	for {
		scanner.Scan()
		fmt.Sscan(scanner.Text(), &agentCount)

		for i := 0; i < agentCount; i++ {
			// cooldown: Number of turns before this agent can shoot
			// wetness: Damage (0-100) this agent has taken
			var agentId, x, y, cooldown, splashBombs, wetness int
			scanner.Scan()
			fmt.Sscan(scanner.Text(), &agentId, &x, &y, &cooldown, &splashBombs, &wetness)

			// Update agent positions and state
			if agent, exists := game.Agents[agentId]; exists {
				agent.X = x
				agent.Y = y
				agent.Cooldown = cooldown
				agent.SplashBombs = splashBombs
				agent.Wetness = wetness
			}
		}

		// myAgentCount: Number of alive agents controlled by you
		var myAgentCount int
		scanner.Scan()
		fmt.Sscan(scanner.Text(), &myAgentCount)

		// Calculate moves with collision avoidance
		moves := game.ProcessMoves()

		for _, agent := range game.MyAgents {
			move := moves[agent.ID]
			nextX, nextY := move[0], move[1]

			// One line per agent: <agentId>;<action1;action2;...> actions are "MOVE x y | SHOOT id | THROW x y | HUNKER_DOWN | MESSAGE text"
			fmt.Printf("%d; MOVE %d %d\n", agent.ID, nextX, nextY)
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
}

// NewGame creates a new game instance
func NewGame() *Game {
	return &Game{
		Agents:   make(map[int]*Agent),
		MyAgents: make([]*Agent, 0),
	}
}

// CalculateNextMove determines the optimal next position for an agent
func (g *Game) CalculateNextMove(agent *Agent) (int, int) {
	dx := agent.TargetX - agent.X
	dy := agent.TargetY - agent.Y

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

// ProcessMoves calculates moves for all agents with collision avoidance
func (g *Game) ProcessMoves() map[int][2]int {
	moves := make(map[int][2]int)

	// Calculate intended moves for all agents
	for _, agent := range g.MyAgents {
		nextX, nextY := g.CalculateNextMove(agent)
		moves[agent.ID] = [2]int{nextX, nextY}
	}

	// Check for collisions between our agents and resolve them
	agentList := make([]*Agent, 0, len(g.MyAgents))
	agentList = append(agentList, g.MyAgents...)

	// Sort by agent ID for consistent priority
	for i := 0; i < len(agentList); i++ {
		for j := i + 1; j < len(agentList); j++ {
			if agentList[i].ID > agentList[j].ID {
				agentList[i], agentList[j] = agentList[j], agentList[i]
			}
		}
	}

	// Resolve collisions: lower ID gets priority
	for i := 0; i < len(agentList); i++ {
		for j := i + 1; j < len(agentList); j++ {
			agent1 := agentList[i]
			agent2 := agentList[j]

			move1 := moves[agent1.ID]
			move2 := moves[agent2.ID]

			if g.CheckCollision(move1[0], move1[1], move2[0], move2[1]) {
				// Agent1 (lower ID) keeps their move, agent2 finds alternate
				altX, altY := g.GetAlternateMove(agent2, move2[0], move2[1])

				// Check if alternate move is valid and doesn't create new collision
				if g.IsValidPosition(altX, altY) {
					// Check if alternate doesn't collide with agent1's move
					if !g.CheckCollision(altX, altY, move1[0], move1[1]) {
						moves[agent2.ID] = [2]int{altX, altY}
					} else {
						// Stay put if alternate also collides
						moves[agent2.ID] = [2]int{agent2.X, agent2.Y}
					}
				} else {
					// Stay put if alternate is out of bounds
					moves[agent2.ID] = [2]int{agent2.X, agent2.Y}
				}
			}
		}
	}

	return moves
}
