package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Enhanced Agent with all game mechanics
type RealAgent struct {
	ID            int
	PlayerID      int
	X, Y          int
	Wetness       int
	Cooldown      int
	SplashBombs   int
	SoakingPower  int
	OptimalRange  int
	ShootCooldown int
	LastAction    string // Track what the agent did last turn
}

// Game scenario
type RealScenario struct {
	Name   string
	Width  int
	Height int
	Map    [][]int
	Agents []RealAgent
}

// Bot process management
type RealBotProcess struct {
	PlayerID int
	Path     string
	Cmd      *exec.Cmd
	Stdin    io.WriteCloser
	Stdout   *bufio.Scanner
	Stderr   *bufio.Scanner
	Name     string
}

// Game state with real mechanics
type RealGameState struct {
	Width        int
	Height       int
	Map          [][]int
	Agents       []RealAgent
	Turn         int
	Player0Score int
	Player1Score int
}

// Action types with priorities for proper game simulation
type RealAction struct {
	AgentID  int
	Type     string // MOVE, SHOOT, THROW, HUNKER_DOWN, MESSAGE
	Args     []string
	Priority int // 1=MOVE, 2=HUNKER_DOWN, 3=SHOOT/THROW
}

// Calculate Manhattan distance
func RealManhattanDistance(x1, y1, x2, y2 int) int {
	return int(math.Abs(float64(x1-x2)) + math.Abs(float64(y1-y2)))
}

// Load scenario from simple format
func LoadRealScenario(filename string) (*RealScenario, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scenario := &RealScenario{Name: filename}
	scanner := bufio.NewScanner(file)

	// Parse file
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "MAP ") {
			parts := strings.Fields(line)
			scenario.Width, _ = strconv.Atoi(parts[1])
			scenario.Height, _ = strconv.Atoi(parts[2])
			scenario.Map = make([][]int, scenario.Height)

			// Read map data
			for i := 0; i < scenario.Height; i++ {
				scanner.Scan()
				mapLine := strings.Fields(scanner.Text())
				scenario.Map[i] = make([]int, scenario.Width)
				for j, val := range mapLine {
					scenario.Map[i][j], _ = strconv.Atoi(val)
				}
			}
		} else if line == "AGENTS" {
			// Read agents
			for scanner.Scan() {
				agentLine := strings.TrimSpace(scanner.Text())
				if agentLine == "" {
					break
				}
				parts := strings.Fields(agentLine)
				if len(parts) >= 4 {
					agent := RealAgent{
						// Default values for real gameplay
						SoakingPower:  16,
						OptimalRange:  4,
						ShootCooldown: 1,
						SplashBombs:   2,
						LastAction:    "SPAWN",
					}
					agent.ID, _ = strconv.Atoi(parts[0])
					agent.PlayerID, _ = strconv.Atoi(parts[1])
					agent.X, _ = strconv.Atoi(parts[2])
					agent.Y, _ = strconv.Atoi(parts[3])
					scenario.Agents = append(scenario.Agents, agent)
				}
			}
		}
	}
	return scenario, nil
}

// Print map with current agent positions and enhanced status
func PrintRealMapWithAgents(scenario *RealScenario, agents []RealAgent, turn int) {
	fmt.Printf("🗺️  Turn %d Battle Map:\n", turn)

	// Create a map to show agents with status
	agentMap := make(map[string]string)
	for _, agent := range agents {
		if agent.Wetness < 100 { // Only show alive agents
			key := fmt.Sprintf("%d,%d", agent.X, agent.Y)
			symbol := fmt.Sprintf("%d", agent.PlayerID+1)

			// Status indicators
			if agent.Wetness >= 75 {
				symbol = "💀" + symbol // Critical condition
			} else if agent.Wetness >= 50 {
				symbol = "🟡" + symbol // Wounded (reduced territory control)
			} else if agent.Cooldown > 0 {
				symbol = "🔄" + symbol // On cooldown
			} else if agent.SplashBombs == 0 {
				symbol = "🚫" + symbol // No bombs left
			}
			agentMap[key] = symbol
		}
	}

	// Print column numbers
	fmt.Printf("     ")
	for x := 0; x < scenario.Width; x++ {
		fmt.Printf("%2d", x%10)
	}
	fmt.Printf("\n")

	// Print map with agents and terrain
	for y := 0; y < scenario.Height; y++ {
		fmt.Printf(" %2d  ", y)
		for x := 0; x < scenario.Width; x++ {
			key := fmt.Sprintf("%d,%d", x, y)

			if agentSymbol, hasAgent := agentMap[key]; hasAgent {
				fmt.Printf("%2s", agentSymbol)
			} else {
				switch scenario.Map[y][x] {
				case 0:
					fmt.Printf(" .")
				case 1:
					fmt.Printf(" ▒") // Low cover (50% protection)
				case 2:
					fmt.Printf(" █") // High cover (75% protection)
				default:
					fmt.Printf(" ?")
				}
			}
		}
		fmt.Printf("\n")
	}

	// Enhanced legend
	fmt.Printf("   Legend: 1=P0 2=P1 🟡=wounded(≥50 wetness) 💀=critical(≥75) 🔄=cooldown 🚫=no bombs\n")
	fmt.Printf("   Terrain: .=empty  ▒=low cover(50%% protection)  █=high cover(75%% protection)\n\n")
}

// Check if position is valid and empty
func (gs *RealGameState) IsValidPosition(x, y int) bool {
	if x < 0 || x >= gs.Width || y < 0 || y >= gs.Height {
		return false
	}
	// Check if tile has cover
	if gs.Map[y][x] != 0 {
		return false
	}
	// Check if another agent is there
	for _, agent := range gs.Agents {
		if agent.Wetness < 100 && agent.X == x && agent.Y == y {
			return false
		}
	}
	return true
}

// Get cover protection for an agent being shot
func (gs *RealGameState) GetCoverProtection(defenderX, defenderY, attackerX, attackerY int) float64 {
	maxProtection := 0.0

	// Check all adjacent cover tiles
	directions := [][]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}

	for _, dir := range directions {
		coverX := defenderX + dir[0]
		coverY := defenderY + dir[1]

		// Check if cover exists
		if coverX >= 0 && coverX < gs.Width && coverY >= 0 && coverY < gs.Height {
			coverType := gs.Map[coverY][coverX]
			if coverType > 0 {
				// Check if attacker is on opposite side of cover
				coverToAttackerX := attackerX - coverX
				coverToAttackerY := attackerY - coverY
				defenderToCoverX := coverX - defenderX
				defenderToCoverY := coverY - defenderY

				// Same direction means opposite sides
				if (coverToAttackerX*defenderToCoverX + coverToAttackerY*defenderToCoverY) > 0 {
					// Check if both are adjacent to same cover (nullifies protection)
					attackerAdjacentToCover := RealManhattanDistance(attackerX, attackerY, coverX, coverY) == 1
					if !attackerAdjacentToCover {
						protection := 0.5 // Low cover
						if coverType == 2 {
							protection = 0.75 // High cover
						}
						if protection > maxProtection {
							maxProtection = protection
						}
					}
				}
			}
		}
	}

	return maxProtection
}

// Execute SHOOT action with real damage calculations
func (gs *RealGameState) ExecuteShoot(shooterID, targetID int) {
	shooter := gs.GetAgent(shooterID)
	target := gs.GetAgent(targetID)

	if shooter == nil || target == nil || shooter.Wetness >= 100 || target.Wetness >= 100 {
		return
	}

	if shooter.Cooldown > 0 {
		fmt.Printf("   🚫 Agent %d cannot shoot (cooldown: %d)\n", shooterID, shooter.Cooldown)
		return
	}

	distance := RealManhattanDistance(shooter.X, shooter.Y, target.X, target.Y)
	maxRange := shooter.OptimalRange * 2

	if distance > maxRange {
		fmt.Printf("   ❌ Agent %d shot at Agent %d failed (too far: %d > %d)\n",
			shooterID, targetID, distance, maxRange)
		return
	}

	// Calculate base damage
	damage := float64(shooter.SoakingPower)

	// Apply range penalty
	if distance > shooter.OptimalRange {
		damage *= 0.5
		fmt.Printf("   📐 Range penalty applied: %d > %d (50%% damage)\n", distance, shooter.OptimalRange)
	}

	// Apply cover protection
	coverProtection := gs.GetCoverProtection(target.X, target.Y, shooter.X, shooter.Y)

	// Apply hunker down protection (if target hunkered this turn)
	totalProtection := coverProtection
	if target.LastAction == "HUNKER_DOWN" {
		totalProtection = math.Min(1.0, totalProtection+0.25) // Stack with cover, max 100%
	}

	damage *= (1.0 - totalProtection)
	finalDamage := int(math.Round(damage))

	// Apply damage
	gs.UpdateAgent(targetID, func(a *RealAgent) {
		a.Wetness += finalDamage
		if a.Wetness > 100 {
			a.Wetness = 100
		}
	})

	// Set cooldown
	gs.UpdateAgent(shooterID, func(a *RealAgent) {
		a.Cooldown = a.ShootCooldown
	})

	protectionInfo := ""
	if totalProtection > 0 {
		protectionInfo = fmt.Sprintf(" (%.0f%% protection)", totalProtection*100)
	}

	eliminatedMsg := ""
	if target.Wetness >= 100 {
		eliminatedMsg = " [ELIMINATED!]"
	}

	fmt.Printf("   💥 Agent %d shot Agent %d: %d damage%s (wetness: %d/100)%s\n",
		shooterID, targetID, finalDamage, protectionInfo, target.Wetness, eliminatedMsg)
}

// Execute THROW action with splash damage
func (gs *RealGameState) ExecuteThrow(agentID int, targetX, targetY int) {
	agent := gs.GetAgent(agentID)
	if agent == nil || agent.Wetness >= 100 || agent.SplashBombs <= 0 {
		return
	}

	distance := RealManhattanDistance(agent.X, agent.Y, targetX, targetY)
	if distance > 4 {
		fmt.Printf("   ❌ Agent %d bomb throw failed (too far: %d > 4)\n", agentID, distance)
		return
	}

	// Splash bomb hits target + all adjacent tiles (3x3 area)
	splashPositions := [][]int{{targetX, targetY}}
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue // Already added center
			}
			splashPositions = append(splashPositions, []int{targetX + dx, targetY + dy})
		}
	}

	hitAgents := 0
	eliminatedAgents := 0
	for _, pos := range splashPositions {
		x, y := pos[0], pos[1]
		if x >= 0 && x < gs.Width && y >= 0 && y < gs.Height {
			// Find agents at this position
			for i := range gs.Agents {
				if gs.Agents[i].Wetness < 100 && gs.Agents[i].X == x && gs.Agents[i].Y == y {
					oldWetness := gs.Agents[i].Wetness
					gs.Agents[i].Wetness += 30 // Splash damage ignores protection
					if gs.Agents[i].Wetness > 100 {
						gs.Agents[i].Wetness = 100
					}
					hitAgents++
					if oldWetness < 100 && gs.Agents[i].Wetness >= 100 {
						eliminatedAgents++
					}
					fmt.Printf("   💣 Agent %d hit by splash: +30 wetness (total: %d/100)\n",
						gs.Agents[i].ID, gs.Agents[i].Wetness)
				}
			}
		}
	}

	// Use bomb
	gs.UpdateAgent(agentID, func(a *RealAgent) {
		a.SplashBombs--
	})

	fmt.Printf("   🧨 Agent %d threw bomb at (%d,%d): %d agents hit",
		agentID, targetX, targetY, hitAgents)
	if eliminatedAgents > 0 {
		fmt.Printf(", %d eliminated!", eliminatedAgents)
	}
	fmt.Printf("\n")
}

// Execute MOVE action with pathfinding
func (gs *RealGameState) ExecuteMove(agentID int, targetX, targetY int) {
	agent := gs.GetAgent(agentID)
	if agent == nil || agent.Wetness >= 100 {
		return
	}

	// Simple movement: try to move one step toward target
	dx := 0
	dy := 0

	if targetX > agent.X {
		dx = 1
	} else if targetX < agent.X {
		dx = -1
	}

	if targetY > agent.Y {
		dy = 1
	} else if targetY < agent.Y {
		dy = -1
	}

	// Try primary direction first
	newX := agent.X + dx
	newY := agent.Y + dy

	if gs.IsValidPosition(newX, newY) {
		gs.UpdateAgent(agentID, func(a *RealAgent) {
			a.X = newX
			a.Y = newY
		})
		fmt.Printf("   🚶 Agent %d moved to (%d,%d)\n", agentID, newX, newY)
	} else {
		// Try alternative directions
		if dx != 0 && gs.IsValidPosition(agent.X+dx, agent.Y) {
			gs.UpdateAgent(agentID, func(a *RealAgent) {
				a.X = agent.X + dx
			})
			fmt.Printf("   🚶 Agent %d moved to (%d,%d) [alt-x]\n", agentID, agent.X+dx, agent.Y)
		} else if dy != 0 && gs.IsValidPosition(agent.X, agent.Y+dy) {
			gs.UpdateAgent(agentID, func(a *RealAgent) {
				a.Y = agent.Y + dy
			})
			fmt.Printf("   🚶 Agent %d moved to (%d,%d) [alt-y]\n", agentID, agent.X, agent.Y+dy)
		} else {
			fmt.Printf("   🚫 Agent %d movement blocked\n", agentID)
		}
	}
}

// Calculate territory control and update scores (key game mechanic!)
func (gs *RealGameState) UpdateTerritoryControl() {
	player0Tiles := 0
	player1Tiles := 0

	for y := 0; y < gs.Height; y++ {
		for x := 0; x < gs.Width; x++ {
			minDist0 := math.MaxInt32
			minDist1 := math.MaxInt32

			// Find closest agent for each player
			for _, agent := range gs.Agents {
				if agent.Wetness >= 100 {
					continue // Dead agents don't control territory
				}

				distance := RealManhattanDistance(x, y, agent.X, agent.Y)

				// Double distance if agent has wetness >= 50 (key rule!)
				if agent.Wetness >= 50 {
					distance *= 2
				}

				if agent.PlayerID == 0 && distance < minDist0 {
					minDist0 = distance
				} else if agent.PlayerID == 1 && distance < minDist1 {
					minDist1 = distance
				}
			}

			// Assign tile control
			if minDist0 < minDist1 {
				player0Tiles++
			} else if minDist1 < minDist0 {
				player1Tiles++
			}
		}
	}

	// Award points for controlling more territory
	if player0Tiles > player1Tiles {
		pointsGained := player0Tiles - player1Tiles
		gs.Player0Score += pointsGained
		fmt.Printf("   🏆 Player 0 gains %d points this turn!\n", pointsGained)
	} else if player1Tiles > player0Tiles {
		pointsGained := player1Tiles - player0Tiles
		gs.Player1Score += pointsGained
		fmt.Printf("   🏆 Player 1 gains %d points this turn!\n", pointsGained)
	}

	fmt.Printf("   📊 Territory: P0=%d tiles, P1=%d tiles | Scores: P0=%d, P1=%d\n",
		player0Tiles, player1Tiles, gs.Player0Score, gs.Player1Score)
}

// Parse actions from bot output
func ParseRealActions(agentActions []string) []RealAction {
	var actions []RealAction

	for _, actionLine := range agentActions {
		parts := strings.Split(actionLine, ";")
		if len(parts) < 2 {
			continue
		}

		agentIDStr := strings.TrimSpace(parts[0])
		agentID, err := strconv.Atoi(agentIDStr)
		if err != nil {
			continue
		}

		// Parse each action component
		for i := 1; i < len(parts); i++ {
			actionStr := strings.TrimSpace(parts[i])
			fields := strings.Fields(actionStr)
			if len(fields) == 0 {
				continue
			}

			action := RealAction{AgentID: agentID}

			switch fields[0] {
			case "MOVE":
				action.Type = "MOVE"
				action.Priority = 1
				if len(fields) >= 3 {
					action.Args = []string{fields[1], fields[2]}
				}
			case "HUNKER_DOWN":
				action.Type = "HUNKER_DOWN"
				action.Priority = 2
			case "SHOOT":
				action.Type = "SHOOT"
				action.Priority = 3
				if len(fields) >= 2 {
					action.Args = []string{fields[1]}
				}
			case "THROW":
				action.Type = "THROW"
				action.Priority = 3
				if len(fields) >= 3 {
					action.Args = []string{fields[1], fields[2]}
				}
			case "MESSAGE":
				action.Type = "MESSAGE"
				action.Priority = 4
				if len(fields) >= 2 {
					action.Args = fields[1:]
				}
			}

			if action.Type != "" {
				actions = append(actions, action)
			}
		}
	}

	return actions
}

// Helper functions for game state
func (gs *RealGameState) GetAgent(id int) *RealAgent {
	for i := range gs.Agents {
		if gs.Agents[i].ID == id {
			return &gs.Agents[i]
		}
	}
	return nil
}

func (gs *RealGameState) UpdateAgent(id int, updateFunc func(*RealAgent)) {
	for i := range gs.Agents {
		if gs.Agents[i].ID == id {
			updateFunc(&gs.Agents[i])
			break
		}
	}
}

// Execute one complete game turn with proper action prioritization
func (gs *RealGameState) ExecuteTurn(player0Actions, player1Actions []string) {
	fmt.Printf("⚔️  Turn %d\n", gs.Turn)
	fmt.Printf("========\n")

	// Parse all actions
	allActions := append(ParseRealActions(player0Actions), ParseRealActions(player1Actions)...)

	// Sort actions by priority (MOVE=1, HUNKER_DOWN=2, SHOOT/THROW=3)
	actionsByPriority := make(map[int][]RealAction)
	for _, action := range allActions {
		actionsByPriority[action.Priority] = append(actionsByPriority[action.Priority], action)
	}

	// Execute actions in priority order (critical for fair gameplay!)
	for priority := 1; priority <= 4; priority++ {
		actions := actionsByPriority[priority]

		for _, action := range actions {
			// Mark agent's last action for hunker down tracking
			gs.UpdateAgent(action.AgentID, func(a *RealAgent) {
				a.LastAction = action.Type
			})

			switch action.Type {
			case "MOVE":
				if len(action.Args) >= 2 {
					x, _ := strconv.Atoi(action.Args[0])
					y, _ := strconv.Atoi(action.Args[1])
					gs.ExecuteMove(action.AgentID, x, y)
				}
			case "HUNKER_DOWN":
				fmt.Printf("   🛡️  Agent %d hunkers down (+25%% protection)\n", action.AgentID)
			case "SHOOT":
				if len(action.Args) >= 1 {
					targetID, _ := strconv.Atoi(action.Args[0])
					gs.ExecuteShoot(action.AgentID, targetID)
				}
			case "THROW":
				if len(action.Args) >= 2 {
					x, _ := strconv.Atoi(action.Args[0])
					y, _ := strconv.Atoi(action.Args[1])
					gs.ExecuteThrow(action.AgentID, x, y)
				}
			case "MESSAGE":
				fmt.Printf("   💬 Agent %d: %s\n", action.AgentID, strings.Join(action.Args, " "))
			}
		}
	}

	// Decrease cooldowns
	for i := range gs.Agents {
		if gs.Agents[i].Cooldown > 0 {
			gs.Agents[i].Cooldown--
		}
	}

	// Update territory control and scores (key scoring mechanism!)
	gs.UpdateTerritoryControl()

	// Increment turn counter
	gs.Turn++
}

// Check win conditions according to task5-final.md
func (gs *RealGameState) CheckWinCondition() (bool, string) {
	// Count alive agents
	alive0, alive1 := 0, 0
	for _, agent := range gs.Agents {
		if agent.Wetness < 100 {
			if agent.PlayerID == 0 {
				alive0++
			} else {
				alive1++
			}
		}
	}

	// Elimination victory
	if alive0 == 0 && alive1 > 0 {
		return true, "🏆 Player 1 WINS! (All enemy agents eliminated)"
	}
	if alive1 == 0 && alive0 > 0 {
		return true, "🏆 Player 0 WINS! (All enemy agents eliminated)"
	}
	if alive0 == 0 && alive1 == 0 {
		return true, "🤝 TIE! (All agents eliminated)"
	}

	// Score victory (600 point lead)
	scoreDiff := gs.Player0Score - gs.Player1Score
	if scoreDiff >= 600 {
		return true, fmt.Sprintf("🏆 Player 0 WINS! (Score lead: %d ≥ 600)", scoreDiff)
	}
	if scoreDiff <= -600 {
		return true, fmt.Sprintf("🏆 Player 1 WINS! (Score lead: %d ≥ 600)", -scoreDiff)
	}

	// Turn limit (100 turns)
	if gs.Turn >= 100 {
		if gs.Player0Score > gs.Player1Score {
			return true, fmt.Sprintf("🏆 Player 0 WINS! (Final scores: %d vs %d)", gs.Player0Score, gs.Player1Score)
		} else if gs.Player1Score > gs.Player0Score {
			return true, fmt.Sprintf("🏆 Player 1 WINS! (Final scores: %d vs %d)", gs.Player1Score, gs.Player0Score)
		} else {
			return true, fmt.Sprintf("🤝 TIE! (Final scores: %d vs %d)", gs.Player0Score, gs.Player1Score)
		}
	}

	return false, ""
}

// Start a bot process
func StartRealBot(botPath string, playerID int) (*RealBotProcess, error) {
	cmd := exec.Command(botPath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	bot := &RealBotProcess{
		PlayerID: playerID,
		Path:     botPath,
		Cmd:      cmd,
		Stdin:    stdin,
		Stdout:   bufio.NewScanner(stdout),
		Stderr:   bufio.NewScanner(stderr),
		Name:     fmt.Sprintf("Bot%d", playerID+1),
	}

	return bot, nil
}

// Send initialization data to bot
func SendRealInitData(bot *RealBotProcess, scenario *RealScenario) error {
	// Send player ID
	fmt.Fprintf(bot.Stdin, "%d\n", bot.PlayerID)

	// Send agent count
	fmt.Fprintf(bot.Stdin, "%d\n", len(scenario.Agents))

	// Send agent data
	for _, agent := range scenario.Agents {
		fmt.Fprintf(bot.Stdin, "%d %d %d %d %d %d\n",
			agent.ID, agent.PlayerID, agent.ShootCooldown,
			agent.OptimalRange, agent.SoakingPower, agent.SplashBombs)
	}

	// Send map dimensions
	fmt.Fprintf(bot.Stdin, "%d %d\n", scenario.Width, scenario.Height)

	// Send map data
	for y := 0; y < scenario.Height; y++ {
		var mapLine []string
		for x := 0; x < scenario.Width; x++ {
			mapLine = append(mapLine, fmt.Sprintf("%d %d %d", x, y, scenario.Map[y][x]))
		}
		fmt.Fprintf(bot.Stdin, "%s\n", strings.Join(mapLine, " "))
	}

	return nil
}

// Send turn data to bot
func SendRealTurnData(bot *RealBotProcess, agents []RealAgent) error {
	// Count alive agents
	aliveAgents := 0
	for _, agent := range agents {
		if agent.Wetness < 100 {
			aliveAgents++
		}
	}

	// Send alive agent count
	fmt.Fprintf(bot.Stdin, "%d\n", aliveAgents)

	// Send agent states
	for _, agent := range agents {
		if agent.Wetness < 100 {
			fmt.Fprintf(bot.Stdin, "%d %d %d %d %d %d\n",
				agent.ID, agent.X, agent.Y, agent.Cooldown, agent.SplashBombs, agent.Wetness)
		}
	}

	// Count player's agents
	myAgents := 0
	for _, agent := range agents {
		if agent.Wetness < 100 && agent.PlayerID == bot.PlayerID {
			myAgents++
		}
	}
	fmt.Fprintf(bot.Stdin, "%d\n", myAgents)

	return nil
}

// Read bot response with timeout
func ReadRealBotResponse(bot *RealBotProcess) ([]string, []string, error) {
	var actions []string
	var stderr []string

	// Read stderr (debug output) non-blocking
	go func() {
		for bot.Stderr.Scan() {
			line := bot.Stderr.Text()
			stderr = append(stderr, line)
			if len(stderr) > 5 { // Keep only last 5 lines
				stderr = stderr[1:]
			}
		}
	}()

	// Read actions from stdout
	for {
		if bot.Stdout.Scan() {
			line := strings.TrimSpace(bot.Stdout.Text())
			if line != "" {
				actions = append(actions, line)
				// Stop after reasonable number of actions per turn
				if len(actions) >= 10 {
					break
				}
			}
		} else {
			break
		}

		// Timeout after 1 second
		time.Sleep(10 * time.Millisecond)
		if len(actions) > 0 {
			break
		}
	}

	return actions, stderr, nil
}

// Test if bot executable exists and is runnable
func TestRealBot(botPath string) bool {
	if _, err := os.Stat(botPath); os.IsNotExist(err) {
		return false
	}

	cmd := exec.Command(botPath)
	err := cmd.Start()
	if err != nil {
		return false
	}
	cmd.Process.Kill()
	return true
}

// MAIN REAL GAME TESTER - implements actual water fight mechanics!
func RunRealWaterFightBattle(bot1Path, bot2Path, scenarioPath string) {
	fmt.Printf("💧 REAL WATER FIGHT SIMULATION 💧\n")
	fmt.Printf("==================================\n")
	fmt.Printf("🤖 Bot 1: %s\n", bot1Path)
	fmt.Printf("🤖 Bot 2: %s\n", bot2Path)
	fmt.Printf("🗺️  Scenario: %s\n", scenarioPath)
	fmt.Printf("📖 Using REAL game mechanics from task5-final.md\n")
	fmt.Printf("\n")

	// Load scenario
	scenario, err := LoadRealScenario(scenarioPath)
	if err != nil {
		fmt.Printf("❌ Failed to load scenario: %v\n", err)
		return
	}

	fmt.Printf("📋 Battle Info:\n")
	fmt.Printf("   🗺️  Map: %dx%d\n", scenario.Width, scenario.Height)
	fmt.Printf("   👥 Agents: %d total\n", len(scenario.Agents))

	// Count agents per player
	player0Count := 0
	player1Count := 0
	for _, agent := range scenario.Agents {
		if agent.PlayerID == 0 {
			player0Count++
		} else {
			player1Count++
		}
	}
	fmt.Printf("   🔵 Player 0: %d agents\n", player0Count)
	fmt.Printf("   🔴 Player 1: %d agents\n", player1Count)

	// Initialize game state with real mechanics
	gameState := &RealGameState{
		Width:        scenario.Width,
		Height:       scenario.Height,
		Map:          scenario.Map,
		Agents:       scenario.Agents,
		Turn:         1,
		Player0Score: 0,
		Player1Score: 0,
	}

	// Print initial map
	PrintRealMapWithAgents(scenario, gameState.Agents, 0)

	fmt.Printf("🚀 Starting REAL water fight battle...\n")
	fmt.Printf("💥 Real shooting • 🧨 Splash bombs • 🛡️ Cover system • 🏆 Territory control\n\n")

	// Start both bots
	bot1, err := StartRealBot(bot1Path, 0)
	if err != nil {
		fmt.Printf("❌ Failed to start Bot1: %v\n", err)
		return
	}
	defer bot1.Cmd.Process.Kill()

	bot2, err := StartRealBot(bot2Path, 1)
	if err != nil {
		fmt.Printf("❌ Failed to start Bot2: %v\n", err)
		return
	}
	defer bot2.Cmd.Process.Kill()

	// Send initialization data
	err = SendRealInitData(bot1, scenario)
	if err != nil {
		fmt.Printf("❌ Failed to send init data to Bot1: %v\n", err)
		return
	}

	err = SendRealInitData(bot2, scenario)
	if err != nil {
		fmt.Printf("❌ Failed to send init data to Bot2: %v\n", err)
		return
	}

	// REAL GAME SIMULATION LOOP
	for gameState.Turn <= 100 {
		// Send turn data to both bots
		SendRealTurnData(bot1, gameState.Agents)
		SendRealTurnData(bot2, gameState.Agents)

		// Read bot responses
		fmt.Printf("🤖 Bot1 thinking...")
		actions1, stderr1, err := ReadRealBotResponse(bot1)
		if err != nil {
			fmt.Printf(" ❌ Error\n")
			actions1 = []string{} // Continue with empty actions
		} else {
			fmt.Printf(" ✅ Done\n")
		}

		fmt.Printf("🤖 Bot2 thinking...")
		actions2, stderr2, err := ReadRealBotResponse(bot2)
		if err != nil {
			fmt.Printf(" ❌ Error\n")
			actions2 = []string{} // Continue with empty actions
		} else {
			fmt.Printf(" ✅ Done\n")
		}

		// Show what bots are planning
		if len(actions1) > 0 {
			fmt.Printf("   🔵 Bot1 Commands: %s\n", strings.Join(actions1, " | "))
		}
		if len(actions2) > 0 {
			fmt.Printf("   🔴 Bot2 Commands: %s\n", strings.Join(actions2, " | "))
		}

		// Show debug output
		if len(stderr1) > 0 {
			fmt.Printf("   🔍 Bot1 Debug: %s\n", stderr1[len(stderr1)-1])
		}
		if len(stderr2) > 0 {
			fmt.Printf("   🔍 Bot2 Debug: %s\n", stderr2[len(stderr2)-1])
		}

		// EXECUTE REAL GAME TURN with proper mechanics
		gameState.ExecuteTurn(actions1, actions2)

		// Print updated map with real agent positions
		PrintRealMapWithAgents(scenario, gameState.Agents, gameState.Turn-1)

		// Check for real win conditions
		gameOver, winMessage := gameState.CheckWinCondition()
		if gameOver {
			fmt.Printf("\n🏁 %s\n", winMessage)
			break
		}

		fmt.Printf("\n")
		time.Sleep(1200 * time.Millisecond) // Pause for readability
	}
}

func main() {
	if len(os.Args) < 4 {
		fmt.Printf("💧 REAL WATER FIGHT SIMULATOR 💧\n")
		fmt.Printf("==================================\n\n")
		fmt.Printf("USAGE:\n")
		fmt.Printf("  %s <bot1> <bot2> <scenario>\n\n", os.Args[0])
		fmt.Printf("EXAMPLE:\n")
		fmt.Printf("  %s ./current_bot ./new_bot ./sample1_real.txt\n\n", os.Args[0])
		fmt.Printf("FEATURES:\n")
		fmt.Printf("  💥 Real shooting mechanics (soaking power, range, cover)\n")
		fmt.Printf("  🧨 Splash bombs (30 damage, 4 tile range, 3x3 splash)\n")
		fmt.Printf("  🛡️  Cover system (50%% low, 75%% high protection)\n")
		fmt.Printf("  🏆 Territory control scoring (wounded agents = 2x distance)\n")
		fmt.Printf("  🎯 Real win conditions (600 point lead, elimination, 100 turns)\n")
		fmt.Printf("  📊 Live agent status tracking and map visualization\n")
		return
	}

	bot1Path := os.Args[1]
	bot2Path := os.Args[2]
	scenarioPath := os.Args[3]

	// Quick validation
	if !TestRealBot(bot1Path) {
		fmt.Printf("❌ Bot 1 not found or not executable: %s\n", bot1Path)
		return
	}
	if !TestRealBot(bot2Path) {
		fmt.Printf("❌ Bot 2 not found or not executable: %s\n", bot2Path)
		return
	}

	// Run the real water fight simulation
	RunRealWaterFightBattle(bot1Path, bot2Path, scenarioPath)
}
