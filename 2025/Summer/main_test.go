package main

import (
	"fmt"
	"testing"
)

// Test helper to create a basic game state
func createTestGame() *Game {
	game := NewGame()
	game.Width = 10
	game.Height = 6
	game.MyID = 0

	// Initialize grid with empty tiles
	game.Grid = make([][]Tile, game.Height)
	for i := 0; i < game.Height; i++ {
		game.Grid[i] = make([]Tile, game.Width)
		for j := 0; j < game.Width; j++ {
			game.Grid[i][j] = Tile{X: j, Y: i, Type: 0}
		}
	}

	// Add some cover tiles
	game.Grid[2][4].Type = 2 // High cover
	game.Grid[3][6].Type = 1 // Low cover

	return game
}

// Test agent role assignment
func TestRoleAssignment(t *testing.T) {
	game := createTestGame()
	strategy := NewTeamCoordinationStrategy()

	// Add test agents
	agent1 := &Agent{ID: 1, Player: 0, X: 2, Y: 2, SplashBombs: 2, Cooldown: 0}
	agent2 := &Agent{ID: 2, Player: 0, X: 3, Y: 2, SplashBombs: 0, Cooldown: 0}
	agent3 := &Agent{ID: 3, Player: 0, X: 4, Y: 2, SplashBombs: 1, Cooldown: 1}
	agent4 := &Agent{ID: 4, Player: 0, X: 5, Y: 2, SplashBombs: 0, Cooldown: 0}

	game.MyAgents = []*Agent{agent1, agent2, agent3, agent4}
	game.Agents = map[int]*Agent{1: agent1, 2: agent2, 3: agent3, 4: agent4}

	strategy.assignOptimalRoles(game)

	// Check that roles were assigned
	if len(strategy.agentRoles) != 4 {
		t.Errorf("Expected 4 role assignments, got %d", len(strategy.agentRoles))
	}

	// Check that bomber role is assigned to agent with bombs
	bomberCount := 0
	for _, role := range strategy.agentRoles {
		if role == RoleBomber {
			bomberCount++
		}
	}

	if bomberCount == 0 {
		t.Error("Expected at least one bomber role assignment")
	}
}

// Test collision avoidance
func TestCollisionAvoidance(t *testing.T) {
	game := createTestGame()

	agent1 := &Agent{ID: 1, Player: 0, X: 2, Y: 2}
	agent2 := &Agent{ID: 2, Player: 0, X: 3, Y: 2}
	game.MyAgents = []*Agent{agent1, agent2}
	game.Agents = map[int]*Agent{1: agent1, 2: agent2}

	// Create conflicting move actions (both trying to move to same spot)
	actions := map[int]AgentAction{
		1: {Type: ActionMove, TargetX: 5, TargetY: 3, Priority: PriorityMovement},
		2: {Type: ActionMove, TargetX: 5, TargetY: 3, Priority: PriorityMovement},
	}

	resolved := game.resolveMovementCollisions(actions)

	// Check that both agents didn't get the same target
	pos1 := ""
	pos2 := ""

	if action1, exists := resolved[1]; exists {
		pos1 = fmt.Sprintf("%d,%d", action1.TargetX, action1.TargetY)
	}
	if action2, exists := resolved[2]; exists {
		pos2 = fmt.Sprintf("%d,%d", action2.TargetX, action2.TargetY)
	}

	if pos1 == pos2 && pos1 != "" {
		t.Errorf("Collision not resolved: both agents moving to %s", pos1)
	}
}

// Test territory calculation
func TestTerritoryControl(t *testing.T) {
	game := createTestGame()

	// Add friendly agents
	agent1 := &Agent{ID: 1, Player: 0, X: 2, Y: 2, Wetness: 0}
	agent2 := &Agent{ID: 2, Player: 0, X: 8, Y: 2, Wetness: 0}
	game.MyAgents = []*Agent{agent1, agent2}

	// Add enemy agents
	enemy1 := &Agent{ID: 3, Player: 1, X: 2, Y: 4, Wetness: 0}
	enemy2 := &Agent{ID: 4, Player: 1, X: 8, Y: 4, Wetness: 0}

	game.Agents = map[int]*Agent{1: agent1, 2: agent2, 3: enemy1, 4: enemy2}

	territoryScore := game.EvaluateTerritoryControl()

	// Should have some territory control calculated
	totalTiles := territoryScore.FriendlyTiles + territoryScore.EnemyTiles + territoryScore.Contested
	if totalTiles == 0 {
		t.Error("No territory calculated")
	}

	// Should have roughly balanced control with symmetric setup
	if abs(territoryScore.Advantage) > 20 {
		t.Errorf("Territory imbalance too high: %d", territoryScore.Advantage)
	}
}

// Test bomb target selection
func TestBombTargeting(t *testing.T) {
	game := createTestGame()

	bomber := &Agent{ID: 1, Player: 0, X: 2, Y: 2, SplashBombs: 2}
	game.MyAgents = []*Agent{bomber}

	// Add clustered enemies
	enemy1 := &Agent{ID: 3, Player: 1, X: 5, Y: 3, Wetness: 50}
	enemy2 := &Agent{ID: 4, Player: 1, X: 6, Y: 3, Wetness: 60}
	enemy3 := &Agent{ID: 5, Player: 1, X: 5, Y: 4, Wetness: 40}

	game.Agents = map[int]*Agent{1: bomber, 3: enemy1, 4: enemy2, 5: enemy3}

	bombX, bombY, enemiesHit, shouldBomb := game.FindStrategicBombTarget(bomber)

	// Should decide to bomb multiple enemies
	if !shouldBomb {
		t.Errorf("Should decide to bomb clustered enemies")
	}

	// Should hit multiple enemies
	if enemiesHit < 2 {
		t.Errorf("Should hit at least 2 enemies, got %d", enemiesHit)
	}

	// Should target somewhere near the enemy cluster
	if abs(bombX-5) > 2 || abs(bombY-3) > 2 {
		t.Errorf("Bomb target (%d,%d) too far from enemy cluster", bombX, bombY)
	}
}

// Test cover detection
func TestCoverDetection(t *testing.T) {
	game := createTestGame()

	// Test position next to high cover
	coverLevel := game.GetMaxAdjacentCover(3, 2) // Next to high cover at (4,2)
	if coverLevel != 2 {
		t.Errorf("Expected cover level 2, got %d", coverLevel)
	}

	// Test position with no adjacent cover
	coverLevel = game.GetMaxAdjacentCover(0, 0)
	if coverLevel != 0 {
		t.Errorf("Expected cover level 0, got %d", coverLevel)
	}
}

// Test clustering penalty
func TestClusteringPenalty(t *testing.T) {
	game := createTestGame()

	agent1 := &Agent{ID: 1, Player: 0, X: 3, Y: 3}
	agent2 := &Agent{ID: 2, Player: 0, X: 4, Y: 3}
	game.MyAgents = []*Agent{agent1, agent2}

	// Test penalty for being adjacent to another agent
	penalty := game.calculateAgentClusteringPenalty(4, 4, agent1) // Adjacent to agent2
	if penalty <= 0 {
		t.Error("Expected clustering penalty for adjacent position")
	}

	// Test no penalty for distant position
	penalty = game.calculateAgentClusteringPenalty(0, 0, agent1)
	if penalty > 0 {
		t.Error("Expected no clustering penalty for distant position")
	}
}

// Benchmark territory calculation (expensive operation)
func BenchmarkTerritoryControl(b *testing.B) {
	game := createTestGame()

	// Add agents
	agent1 := &Agent{ID: 1, Player: 0, X: 2, Y: 2, Wetness: 0}
	agent2 := &Agent{ID: 2, Player: 0, X: 8, Y: 2, Wetness: 0}
	enemy1 := &Agent{ID: 3, Player: 1, X: 2, Y: 4, Wetness: 0}
	enemy2 := &Agent{ID: 4, Player: 1, X: 8, Y: 4, Wetness: 0}

	game.MyAgents = []*Agent{agent1, agent2}
	game.Agents = map[int]*Agent{1: agent1, 2: agent2, 3: enemy1, 4: enemy2}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		game.EvaluateTerritoryControl()
	}
}
