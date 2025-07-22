# New Approach

The CodinGame Summer 2025 challenge presents a fascinating opportunity to develop a sophisticated bot for a 2D multi-agent environment, all while adhering to the strict constraint of no external library imports. Your provided task5-final.md rules and main.go code offer invaluable context, detailing the game mechanics, agent attributes, available actions, and the overarching victory conditions. This means our AI must be capable of both aggressive combat and strategic territorial control.

Given the "no library" constraint, all AI components, including Finite State Machines (FSMs) and Behavior Trees (BTs), must be implemented manually using Go's native capabilities. This emphasizes clean, efficient, and idiomatic Go code.

## 1. Understanding the Game and Available Actions

I. Understanding the Game and Available Actions
Based on the provided task5.pdf and main.go code, here's a breakdown of the game state and actions:

Game State (Game struct):

Map: A 2D Grid of Tiles, with Width and Height. Each Tile has X, Y, and Type (0: empty, 1: low cover, 2: high cover). Cover tiles are impassable.

Agents: A map[int]*Agent of all living agents, and a *Agent for MyAgents.

MyID: Your player ID (0 or 1).

Turn Information: Implicitly managed by the game loop.

Strategy: CurrentStrategy Strategy indicates a central decision-making component.

Agent Attributes (Agent struct):

Static (from initialization): ID, Player, ShootCooldown (turns between shots), OptimalRange (for 100% damage), SoakingPower (base damage), MaxSplashBombs (total available).

Dynamic (per turn): X, Y (current position), Cooldown (turns left until next shot), SplashBombs (current remaining), Wetness (damage taken, 100 = eliminated).

Targeting: TargetX, TargetY (for movement).

Available Actions (AgentAction struct):
Each agent can perform one MOVE and one combat action (SHOOT, THROW, or HUNKER_DOWN) per turn. Actions are prioritized: MOVE > HUNKER_DOWN > SHOOT/THROW.

MOVE X Y: Move towards (X, Y) using shortest valid path. Movement is cancelled on collision with cover or another agent.

SHOOT id: Shoot enemy id. Damage depends on OptimalRange and SoakingPower, reduced by target's HUNKER_DOWN or Cover.

THROW X Y: Throw splash bomb at (X, Y). Max distance 4 tiles. Deals 30 wetness to target tile and all adjacent (ortho/diag). Ignores damage reduction. Limited SplashBombs.

HUNKER_DOWN: Gain 25% damage reduction for the turn. Stacks with cover.

MESSAGE text: Debugging message.

Victory Conditions:

Reach 600 more points than opponent.

Eliminate all opposing agents.

Have the most points after 100 turns.

Territory Control:

A tile is controlled if it's closer to your agent than an enemy agent.

If an agent's Wetness >= 50, its distance is doubled for territory comparison.

II. Possible State Machines (FSMs) for High-Level Strategy
FSMs are excellent for managing distinct, high-level strategic modes for your bot. Given the dual victory conditions (elimination vs. territory), a team-level FSM is highly recommended to dictate the overall strategy. Each agent can then have its own FSM for tactical behavior within that strategy, or directly use a Behavior Tree.

A. Team-Level FSM (Implemented by TeamCoordinationStrategy)
This FSM would determine the overarching goal for your entire team of agents. It would reside within your TeamCoordinationStrategy struct, which implements the Strategy interface.

Conceptual States:

StrategicCombat: The primary goal is to eliminate enemy agents. This mode would be active when enemy agents are vulnerable, few in number, or when a quick elimination victory seems plausible.

StrategicTerritoryControl: The primary goal is to capture and hold map territory. This mode would be active when your team is behind on points, when enemies are well-defended, or as the turn limit approaches.

StrategicDefense: Focus on protecting key areas or vulnerable friendly agents. This might be a sub-state of StrategicTerritoryControl or a temporary state triggered by high threat.

StrategicRegroupAndHeal: If a significant portion of your team is low on Wetness, this state prioritizes moving agents to safe zones or cover to HUNKER_DOWN and recover (though wetness doesn't decrease, hunkering reduces further damage).

Conceptual Transitions (within TeamCoordinationStrategy.EvaluateActions):

Current Team State	Condition(s)	Next Team State	High-Level Rationale
StrategicCombat	AllEnemyAgentsEliminated	Victory	Objective achieved.
StrategicCombat	MyAgentsWetnessHigh (e.g., average > 70) AND EnemyAgentsAlive > 0	StrategicRegroupAndHeal	Team is too vulnerable to continue direct combat.
StrategicCombat	TurnNumber > 70 AND MyTerritoryScore < EnemyTerritoryScore	StrategicTerritoryControl	Time is running out, need to secure points.
StrategicTerritoryControl	EnemyAgentsVulnerable (e.g., low health, few in number)	StrategicCombat	Opportunity for quick elimination.
StrategicTerritoryControl	MyAgentsWetnessHigh (e.g., average > 70)	StrategicRegroupAndHeal	Team is too vulnerable to hold territory effectively.
StrategicRegroupAndHeal	MyAgentsWetnessLow (e.g., average < 30) AND EnemyAgentsThreatLow	StrategicCombat or StrategicTerritoryControl	Team is recovered and ready to re-engage.
Any Strategic State	MyAgentsCount == 0	Defeat	All friendly agents eliminated.

Export to Sheets
Go Implementation Sketch for Team FSM:

Go

package main

// TeamStrategyState defines the high-level states for the entire team
type TeamStrategyState int
const (
    TeamStateCombat TeamStrategyState = iota
    TeamStateTerritoryControl
    TeamStateDefense
    TeamStateRegroupAndHeal
)

// TeamCoordinationStrategy will manage the team's overall state
type TeamCoordinationStrategy struct {
    CurrentTeamState TeamStrategyState
    // Add any team-level metrics or timers here
}

func NewTeamCoordinationStrategy() *TeamCoordinationStrategy {
    return &TeamCoordinationStrategy{
        CurrentTeamState: TeamStateCombat, // Default starting strategy
    }
}

func (s *TeamCoordinationStrategy) Name() string {
    switch s.CurrentTeamState {
    case TeamStateCombat: return "Combat"
    case TeamStateTerritoryControl: return "Territory Control"
    case TeamStateDefense: return "Defense"
    case TeamStateRegroupAndHeal: return "Regroup & Heal"
    default: return "Unknown"
    }
}

// EvaluateActions is the main entry point for the team's AI
func (s *TeamCoordinationStrategy) EvaluateActions(agent *Agent, game *Game)AgentAction {
    // This method will be called for EACH agent, but the team strategy update
    // should ideally happen once per turn. A simple way to ensure this is
    // to update the team state only when evaluating the first agent (e.g., agent.ID == game.MyAgents.ID)
    // or by having a separate global update function called once per turn.

    // For simplicity, let's assume this is called once per turn for the "team leader" agent.
    // In a real implementation, you'd need a mechanism to ensure team state update is not redundant.
    if agent.ID == game.MyAgents.ID { // Only update team state once per turn
        s.updateTeamState(game)
    }

    // Now, based on the current team state, each agent decides its individual actions
    return s.decideAgentActions(agent, game)
}

func (s *TeamCoordinationStrategy) updateTeamState(game *Game) {
    // Example simplified transition logic for the team FSM
    myAliveAgents := len(game.MyAgents)
    enemyAliveAgents := 0
    for _, a := range game.Agents {
        if a.Player!= game.MyID && a.Wetness < 100 {
            enemyAliveAgents++
        }
    }

    myTerritoryScore, enemyTerritoryScore := game.CalculateTerritoryScores() // Need to implement this helper
    
    avgMyWetness := 0
    if myAliveAgents > 0 {
        totalWetness := 0
        for _, a := range game.MyAgents {
            totalWetness += a.Wetness
        }
        avgMyWetness = totalWetness / myAliveAgents
    }

    switch s.CurrentTeamState {
    case TeamStateCombat:
        if avgMyWetness >= 70 && myAliveAgents > 0 && enemyAliveAgents > 0 {
            s.CurrentTeamState = TeamStateRegroupAndHeal
            fmt.Fprintln(os.Stderr, "Team: High wetness, switching to Regroup & Heal.")
        } else if game.TurnNumber > 70 && myTerritoryScore < enemyTerritoryScore {
            s.CurrentTeamState = TeamStateTerritoryControl
            fmt.Fprintln(os.Stderr, "Team: Losing on territory late game, switching to Territory Control.")
        } else if enemyAliveAgents == 0 {
            fmt.Fprintln(os.Stderr, "Team: All enemies eliminated! Victory condition met.")
            // No state change needed, game ends
        }
    case TeamStateTerritoryControl:
        if enemyAliveAgents <= 1 && myAliveAgents > 0 { // Only one enemy left, go for the kill
            s.CurrentTeamState = TeamStateCombat
            fmt.Fprintln(os.Stderr, "Team: Few enemies left, switching to Combat.")
        } else if avgMyWetness >= 70 && myAliveAgents > 0 {
            s.CurrentTeamState = TeamStateRegroupAndHeal
            fmt.Fprintln(os.Stderr, "Team: High wetness, switching to Regroup & Heal.")
        }
    case TeamStateRegroupAndHeal:
        if avgMyWetness < 30 && myAliveAgents > 0 { // Recovered enough
            if enemyAliveAgents > 0 {
                s.CurrentTeamState = TeamStateCombat
                fmt.Fprintln(os.Stderr, "Team: Recovered, switching to Combat.")
            } else {
                s.CurrentTeamState = TeamStateTerritoryControl // No enemies, focus on territory
                fmt.Fprintln(os.Stderr, "Team: Recovered, no enemies, switching to Territory Control.")
            }
        }
    }
    fmt.Fprintln(os.Stderr, "Current Team Strategy:", s.Name())
}

// decideAgentActions will be implemented using Behavior Trees or Agent-level FSMs
func (s *TeamCoordinationStrategy) decideAgentActions(agent *Agent, game *Game)AgentAction {
    // This is where each agent's individual AI logic (BT or FSM) would be called
    // based on the s.CurrentTeamState.
    // For now, a placeholder.
    returnAgentAction{
        {Type: ActionHunker, Priority: PriorityDefault, Reason: "Default Hunker"},
    }
}

// Placeholder for Game methods (needs to be added to Game struct)
func (g *Game) CalculateTerritoryScores() (int, int) {
    myScore := 0
    enemyScore := 0
    // Iterate through all tiles
    for y := 0; y < g.Height; y++ {
        for x := 0; x < g.Width; x++ {
            tileControlledByMe := g.IsTileControlledByMe(x, y) // Implement this helper
            if tileControlledByMe {
                myScore++
            } else {
                enemyScore++
            }
        }
    }
    return myScore, enemyScore
}

func (g *Game) IsTileControlledByMe(x, y int) bool {
    minDistMe := 9999
    closestMyAgentWetness := 0
    for _, myAgent := range g.MyAgents {
        dist := abs(myAgent.X-x) + abs(myAgent.Y-y)
        if dist < minDistMe {
            minDistMe = dist
            closestMyAgentWetness = myAgent.Wetness
        }
    }

    minDistEnemy := 9999
    closestEnemyAgentWetness := 0
    for _, enemyAgent := range g.Agents {
        if enemyAgent.Player!= g.MyID && enemyAgent.Wetness < 100 { // Only consider living enemies
            dist := abs(enemyAgent.X-x) + abs(enemyAgent.Y-y)
            if dist < minDistEnemy {
                minDistEnemy = dist
                closestEnemyAgentWetness = enemyAgent.Wetness
            }
        }
    }

    // Apply wetness penalty for territory control
    effectiveMinDistMe := minDistMe
    if closestMyAgentWetness >= 50 {
        effectiveMinDistMe *= 2
    }

    effectiveMinDistEnemy := minDistEnemy
    if closestEnemyAgentWetness >= 50 {
        effectiveMinDistEnemy *= 2
    }

    return effectiveMinDistMe <= effectiveMinDistEnemy
}
B. Agent-Level FSMs (within a Strategy, or as part of a Hybrid BT)
Each agent can have its own FSM to manage its immediate tactical behavior. This can be a simpler FSM or a higher-level FSM that then calls a Behavior Tree.

Conceptual States:

Idle: Agent is waiting, observing, or has no immediate task.

MoveToPosition: Agent is pathfinding and moving to a specific strategic tile (e.g., cover, objective, flanking position).

EngageEnemy: Agent is actively trying to attack an enemy (shooting or throwing bombs).

Reloading: Agent is on ShootCooldown.

Fleeing: Agent is low on Wetness and trying to move to safety/cover.

Hunkering: Agent is actively using HUNKER_DOWN for damage reduction.

CapturingTerritory: Agent is moving to and holding an uncontrolled tile.

Conceptual Transitions:

Current Agent State	Condition(s)	Next Agent State	Rationale
Idle	EnemyDetected	EngageEnemy	Threat detected.
Idle	TeamNeedsTerritory AND UncontrolledTileNearby	CapturingTerritory	Fulfilling team objective.
Idle	NeedsBetterPosition	MoveToPosition	Seeking strategic advantage.
MoveToPosition	TargetPositionReached	Idle	Movement complete.
MoveToPosition	EnemyDetected	EngageEnemy	Interrupted by combat.
EngageEnemy	Agent.Wetness >= 70	Fleeing	Critical health, prioritize survival.
EngageEnemy	Agent.Cooldown > 0	Reloading	Cannot shoot, wait for cooldown.
EngageEnemy	TargetEliminated OR NoViableTarget	Idle	Target gone, return to default.
Reloading	Agent.Cooldown == 0	EngageEnemy	Can shoot again.
Fleeing	Agent.Wetness < 30 AND IsSafe	Idle	Recovered and safe.
CapturingTerritory	TerritoryControlled	Idle	Objective achieved.
CapturingTerritory	EnemyDetected	EngageEnemy	Threat to objective.
Hunkering	NoImmediateThreat	Idle	Threat passed.

Export to Sheets
Go Implementation Sketch for Agent FSM (using switch statement for simplicity):

Go

package main

// AgentTacticalState defines the tactical states for an individual agent
type AgentTacticalState int
const (
    AgentStateIdle AgentTacticalState = iota
    AgentStateMoveToPosition
    AgentStateEngageEnemy
    AgentStateReloading
    AgentStateFleeing
    AgentStateHunkering
    AgentStateCapturingTerritory
)

// Agent struct needs a field to store its current tactical state
// type Agent struct {... CurrentTacticalState AgentTacticalState... }

// DecideAgentTacticalAction determines the agent's immediate action based on its state
func (a *Agent) DecideAgentTacticalAction(game *Game, teamStrategy TeamStrategyState)AgentAction {
    actions :=AgentAction{}

    // This is a simplified FSM logic. In a real bot, each state would have more complex logic
    // and potentially call into Behavior Trees.
    switch a.CurrentTacticalState {
    case AgentStateIdle:
        if a.Wetness >= 70 {
            a.CurrentTacticalState = AgentStateFleeing
            actions = append(actions, AgentAction{Type: ActionMessage, Message: "Fleeing (low health)", Priority: PriorityEmergency, Reason: "Low Health"})
        } else if game.FindNearestEnemy(a)!= nil { // Implement FindNearestEnemy helper
            a.CurrentTacticalState = AgentStateEngageEnemy
            actions = append(actions, AgentAction{Type: ActionMessage, Message: "Engaging Enemy", Priority: PriorityCombat, Reason: "Enemy Detected"})
        } else if teamStrategy == TeamStateTerritoryControl && game.FindNearestUncontrolledTile(a)!= nil { // Implement FindNearestUncontrolledTile
            a.CurrentTacticalState = AgentStateCapturingTerritory
            actions = append(actions, AgentAction{Type: ActionMessage, Message: "Capturing Territory", Priority: PriorityMovement, Reason: "Team Territory Goal"})
        } else {
            actions = append(actions, AgentAction{Type: ActionHunker, Priority: PriorityDefault, Reason: "Idle Hunker"})
        }
    case AgentStateEngageEnemy:
        if a.Wetness >= 70 {
            a.CurrentTacticalState = AgentStateFleeing
            actions = append(actions, AgentAction{Type: ActionMessage, Message: "Fleeing (combat low health)", Priority: PriorityEmergency, Reason: "Low Health in Combat"})
        } else if a.Cooldown > 0 {
            a.CurrentTacticalState = AgentStateReloading
            actions = append(actions, AgentAction{Type: ActionMessage, Message: "Reloading", Priority: PriorityDefault, Reason: "Shoot Cooldown"})
        } else {
            target := game.FindOptimalShootTarget(a) // Implement FindOptimalShootTarget
            if target!= nil {
                actions = append(actions, AgentAction{Type: ActionShoot, TargetAgentID: target.ID, Priority: PriorityCombat, Reason: "Shooting Enemy"})
            } else {
                // No target, maybe move closer or switch to territory
                a.CurrentTacticalState = AgentStateIdle
            }
        }
    case AgentStateReloading:
        if a.Cooldown == 0 {
            a.CurrentTacticalState = AgentStateEngageEnemy
            actions = append(actions, AgentAction{Type: ActionMessage, Message: "Cooldown over, re-engaging", Priority: PriorityCombat, Reason: "Cooldown Finished"})
        } else {
            actions = append(actions, AgentAction{Type: ActionHunker, Priority: PriorityDefault, Reason: "Waiting for Cooldown"})
        }
    case AgentStateFleeing:
        // Logic to move away from danger, find cover
        safeSpotX, safeSpotY := game.FindSafeSpot(a) // Implement FindSafeSpot
        actions = append(actions, AgentAction{Type: ActionMove, TargetX: safeSpotX, TargetY: safeSpotY, Priority: PriorityEmergency, Reason: "Fleeing to Safety"})
        if a.Wetness < 30 &&!game.IsUnderImmediateThreat(a) { // Implement IsUnderImmediateThreat
            a.CurrentTacticalState = AgentStateIdle
            actions = append(actions, AgentAction{Type: ActionMessage, Message: "Safe and recovered", Priority: PriorityDefault, Reason: "Recovered"})
        }
    case AgentStateCapturingTerritory:
        targetTile := game.FindNearestUncontrolledTile(a)
        if targetTile!= nil {
            actions = append(actions, AgentAction{Type: ActionMove, TargetX: targetTile.X, TargetY: targetTile.Y, Priority: PriorityMovement, Reason: "Moving to Capture"})
            if game.IsTileControlledByMe(targetTile.X, targetTile.Y) { // Check if tile is now controlled
                a.CurrentTacticalState = AgentStateIdle
                actions = append(actions, AgentAction{Type: ActionMessage, Message: "Territory captured", Priority: PriorityDefault, Reason: "Territory Captured"})
            }
        } else {
            a.CurrentTacticalState = AgentStateIdle
            actions = append(actions, AgentAction{Type: ActionMessage, Message: "No territory to capture", Priority: PriorityDefault, Reason: "No Territory"})
        }
    }
    return actions
}

// Placeholder Game helper methods (needs to be added to Game struct)
func (g *Game) FindNearestEnemy(agent *Agent) *Agent { return nil }
func (g *Game) FindOptimalShootTarget(agent *Agent) *Agent { return nil }
func (g *Game) FindNearestUncontrolledTile(agent *Agent) *Tile { return nil }
func (g *Game) FindSafeSpot(agent *Agent) (int, int) { return agent.X, agent.Y }
func (g *Game) IsUnderImmediateThreat(agent *Agent) bool { return false }
III. Possible Behavior Trees (BTs) for Tactical Execution
Behavior Trees are ideal for handling the complex, reactive, and interruptible tactical decisions within each FSM state or as the primary decision-making structure for agents.

Manual Go Implementation of BT Nodes:

Go

package main

// NodeState represents the outcome of a node's evaluation
type NodeState int
const (
    Running NodeState = iota // Task is still in progress (e.g., moving, waiting for cooldown)
    Success                  // Task completed successfully or condition met
    Failure                  // Task failed or condition not met
)

// Node interface defines the contract for all behavior tree nodes
type Node interface {
    Evaluate(agent *Agent, game *Game) NodeState
}

// --- Composite Nodes ---

// Composite is a base struct for nodes that have children
type Composite struct {
    ChildrenNode
}

// Sequence node: Executes children in order, fails on first failure/running, succeeds if all succeed
type Sequence struct {
    Composite
}
func (s *Sequence) Evaluate(agent *Agent, game *Game) NodeState {
    for _, child := range s.Children {
        switch child.Evaluate(agent, game) {
        case Failure:
            return Failure // One child failed, sequence fails
        case Running:
            return Running // One child is running, sequence is running
        case Success:
            continue // Child succeeded, move to next child
        }
    }
    return Success // All children succeeded
}

// Selector node: Tries children in order, succeeds on first success/running, fails if all fail
type Selector struct {
    Composite
}
func (s *Selector) Evaluate(agent *Agent, game *Game) NodeState {
    for _, child := range s.Children {
        switch child.Evaluate(agent, game) {
        case Success:
            return Success // One child succeeded, selector succeeds
        case Running:
            return Running // One child is running, selector is running
        case Failure:
            continue // Child failed, try next child
        }
        // If a child returns Running, we need to store which child is running
        // so we can resume from it next tick. This is a key complexity for manual BTs.
        // For simplicity in this high-level example, we assume tasks handle their own
        // multi-tick state and return Running until complete.
    }
    return Failure // All children failed
}

// --- Leaf Nodes (Tasks/Conditions) ---

// Task: Check if agent's wetness is high (e.g., >= 70)
type CheckWetnessHigh struct{}
func (c *CheckWetnessHigh) Evaluate(agent *Agent, game *Game) NodeState {
    if agent.Wetness >= 70 {
        return Success
    }
    return Failure
}

// Task: Move agent to a safe spot (e.g., nearest cover)
type TaskFleeToSafety struct{}
func (t *TaskFleeToSafety) Evaluate(agent *Agent, game *Game) NodeState {
    safeX, safeY := game.FindSafeSpot(agent) // Needs pathfinding
    if agent.X == safeX && agent.Y == safeY {
        return Success // Reached safe spot
    }
    agent.TargetX, agent.TargetY = safeX, safeY
    // This task would generate a MOVE action.
    // In a real implementation, this would involve pathfinding and returning Running until path is complete.
    // For now, we just generate the action.
    game.AddAgentAction(agent.ID, AgentAction{Type: ActionMove, TargetX: safeX, TargetY: safeY, Priority: PriorityEmergency, Reason: "Fleeing"}) // Add to a global action list
    return Running // Assume it takes time to move
}

// Task: Check if agent can shoot (cooldown is 0)
type CheckCanShoot struct{}
func (c *CheckCanShoot) Evaluate(agent *Agent, game *Game) NodeState {
    if agent.Cooldown == 0 {
        return Success
    }
    return Failure
}

// Task: Shoot the optimal target
type TaskShootOptimalTarget struct{}
func (t *TaskShootOptimalTarget) Evaluate(agent *Agent, game *Game) NodeState {
    target := game.FindOptimalShootTarget(agent) // Needs implementation
    if target!= nil {
        game.AddAgentAction(agent.ID, AgentAction{Type: ActionShoot, TargetAgentID: target.ID, Priority: PriorityCombat, Reason: "Shooting"})
        return Success
    }
    return Failure
}

// Task: Check if agent has splash bombs and an optimal target
type CheckCanThrowBomb struct{}
func (c *CheckCanThrowBomb) Evaluate(agent *Agent, game *Game) NodeState {
    if agent.SplashBombs > 0 {
        _, _, damageScore := game.FindOptimalSplashBombTarget(agent) // Provided in main.go
        if damageScore > 0 { // Only throw if it hits something
            return Success
        }
    }
    return Failure
}

// Task: Throw splash bomb at optimal target
type TaskThrowBombOptimal struct{}
func (t *TaskThrowBombOptimal) Evaluate(agent *Agent, game *Game) NodeState {
    targetX, targetY, _ := game.FindOptimalSplashBombTarget(agent)
    game.AddAgentAction(agent.ID, AgentAction{Type: ActionThrow, TargetX: targetX, TargetY: targetY, Priority: PriorityCombat, Reason: "Throwing Bomb"})
    return Success
}

// Task: Hunker Down
type TaskHunkerDown struct{}
func (t *TaskHunkerDown) Evaluate(agent *Agent, game *Game) NodeState {
    game.AddAgentAction(agent.ID, AgentAction{Type: ActionHunker, Priority: PriorityDefault, Reason: "Hunkering Down"})
    return Success
}

// Task: Check if there's an uncontrolled tile nearby
type CheckUncontrolledTileNearby struct{}
func (c *CheckUncontrolledTileNearby) Evaluate(agent *Agent, game *Game) NodeState {
    if game.FindNearestUncontrolledTile(agent)!= nil {
        return Success
    }
    return Failure
}

// Task: Move to nearest uncontrolled tile
type TaskMoveToUncontrolledTile struct{}
func (t *TaskMoveToUncontrolledTile) Evaluate(agent *Agent, game *Game) NodeState {
    targetTile := game.FindNearestUncontrolledTile(agent)
    if targetTile!= nil {
        agent.TargetX, agent.TargetY = targetTile.X, targetTile.Y
        game.AddAgentAction(agent.ID, AgentAction{Type: ActionMove, TargetX: targetTile.X, TargetY: targetTile.Y, Priority: PriorityMovement, Reason: "Moving to Capture Territory"})
        return Running // Assume movement takes time
    }
    return Failure
}

// Placeholder for Game method to collect actions
func (g *Game) AddAgentAction(agentID int, action AgentAction) {
    if _, ok := g.AgentActions;!ok {
        g.AgentActions =AgentAction{}
    }
    g.AgentActions = append(g.AgentActions, action)
}
// Game struct needs: AgentActions map[int]AgentAction

Example Behavior Trees:
1. Combat-Focused Behavior Tree (for StrategicCombat Team State):
This BT prioritizes survival, then high-impact attacks, then direct shooting, and finally positioning.

Go

// BuildCombatBT creates a Behavior Tree for combat-focused agents
func BuildCombatBT() Node {
    return &Selector{ // Try these in order of priority (left to right)
        Composite: Composite{
            Children:Node{
                // Priority 1: Survival (Flee if low health)
                &Sequence{
                    Composite: Composite{
                        Children:Node{
                            &CheckWetnessHigh{}, // Condition: Is agent's wetness high?
                            &TaskFleeToSafety{}, // Action: Flee to a safe spot
                        },
                    },
                },
                // Priority 2: High-impact AoE (Throw Splash Bomb)
                &Sequence{
                    Composite: Composite{
                        Children:Node{
                            &CheckCanThrowBomb{}, // Condition: Can throw bomb and optimal target exists?
                            &TaskThrowBombOptimal{}, // Action: Throw bomb
                        },
                    },
                },
                // Priority 3: Direct Combat (Shoot)
                &Sequence{
                    Composite: Composite{
                        Children:Node{
                            &CheckCanShoot{}, // Condition: Can agent shoot?
                            &TaskShootOptimalTarget{}, // Action: Shoot optimal target
                        },
                    },
                },
                // Priority 4: Default Defensive (Hunker Down if nothing else)
                &TaskHunkerDown{}, // Action: Hunker Down
            },
        },
    }
}
2. Territory-Focused Behavior Tree (for StrategicTerritoryControl Team State):
This BT prioritizes survival, then capturing territory, and finally hunkering down.

Go

// BuildTerritoryBT creates a Behavior Tree for territory-focused agents
func BuildTerritoryBT() Node {
    return &Selector{ // Try these in order of priority (left to right)
        Composite: Composite{
            Children:Node{
                // Priority 1: Survival (Flee if low health)
                &Sequence{
                    Composite: Composite{
                        Children:Node{
                            &CheckWetnessHigh{},
                            &TaskFleeToSafety{},
                        },
                    },
                },
                // Priority 2: Capture Territory
                &Sequence{
                    Composite: Composite{
                        Children:Node{
                            &CheckUncontrolledTileNearby{}, // Condition: Is there an uncontrolled tile nearby?
                            &TaskMoveToUncontrolledTile{},  // Action: Move to capture it
                        },
                    },
                },
                // Priority 3: Default Defensive (Hunker Down if no territory to capture)
                &TaskHunkerDown{},
            },
        },
    }
}
IV. High-Level Coding Flow and Go Examples
The main.go code you provided already sets up a good foundation with the Game struct, Agent struct, and the Strategy interface. The CoordinateActions method is the central orchestrator.

Overall Flow:

Initialization (main function):

Read initial game data (player ID, agent static data, map dimensions, grid tiles).

Initialize Game struct.

Initialize CurrentStrategy (e.g., NewTeamCoordinationStrategy()).

Agent AI Initialization: For each Agent in game.MyAgents, assign its initial CurrentTacticalState (if using agent-level FSMs) or its BehaviorTreeRoot (if using BTs). This could be done in a helper function like agent.SetupAI().

Game Loop (inside for {} in main):

Read Turn Input: Update dynamic agent properties (X, Y, Cooldown, SplashBombs, Wetness).

Team-Level Strategy Update: Inside game.CoordinateActions(), the CurrentStrategy (your TeamCoordinationStrategy instance) will first update its CurrentTeamState based on global game conditions (e.g., s.updateTeamState(game)). This ensures the entire team is aligned on a high-level objective.

Individual Agent Action Evaluation: For each agent in game.MyAgents:

The CurrentStrategy.EvaluateActions(agent, game) method will be called.

Inside EvaluateActions, based on the CurrentTeamState, the agent's specific Behavior Tree (or agent-level FSM) will be evaluated.

The BT's Evaluate method will traverse the tree, executing tasks and returning Success, Failure, or Running.

Tasks (leaf nodes) will generate AgentAction structs (e.g., MOVE, SHOOT, THROW, HUNKER_DOWN) and add them to a temporary list associated with the agent (e.g., game.AgentActions).

Action Conflict Resolution (game.resolveActionConflicts): The provided main.go already has this. It sorts actions by priority and handles movement conflicts. This is crucial as agents can only perform one move and one combat action.

Output Commands: The final, resolved actions for each agent are formatted and printed to os.Stdout.

Key Go Structures and Helper Functions (Manual Implementation):

Pathfinding: A crucial component for MOVE actions. Since no libraries are allowed, you'll need to implement a basic pathfinding algorithm like Breadth-First Search (BFS) or A* search.

func (g *Game) FindPath(startX, startY, targetX, targetY int)Point: Returns a slice of Point (X, Y) representing the path.

Consider Tile.Type (1, 2 are impassable) and other agents' positions as obstacles.

Example Point struct: type Point struct { X, Y int }

Targeting Logic:

func (g *Game) FindNearestEnemy(agent *Agent) *Agent: Finds the closest living enemy agent using Manhattan distance.

func (g *Game) FindOptimalShootTarget(agent *Agent) *Agent: Finds the enemy agent that would take the most damage from SHOOT (considering OptimalRange, SoakingPower, and enemy Wetness / Cover / HUNKER_DOWN).

func (g *Game) FindOptimalSplashBombTarget(agent *Agent) (int, int, float64): This is already provided in your main.go! You can use it directly.

Territory Logic:

func (g *Game) CalculateTerritoryScores() (int, int): Iterates all tiles, determines ownership based on proximity to friendly/enemy agents (doubling distance for wetness >= 50), and sums scores.

func (g *Game) FindNearestUncontrolledTile(agent *Agent) *Tile: Finds the closest tile not controlled by your team.

Cover Logic:

func (g *Game) GetMaxAdjacentCover(x, y int) int: Returns the highest cover type (1 or 2) adjacent to a tile.

func (g *Game) IsTileSafe(x, y int) bool: Checks if a tile provides cover from nearby enemies.

func (g *Game) FindSafeSpot(agent *Agent) (int, int): Finds the nearest tile that offers good cover and is not threatened.

Agent-Specific State (for BT Running nodes):

Tasks that take multiple turns (like TaskMoveTo or TaskPatrol) need to store their internal progress on the Agent struct.

Example: Agent.CurrentPathPoint, Agent.PathStep int, Agent.TargetAgentID int.

High-Level TeamCoordinationStrategy.EvaluateActions (Revised):

Go

package main

//... (TeamStrategyState, TeamCoordinationStrategy struct, NewTeamCoordinationStrategy, Name methods as above)...

// EvaluateActions is the main entry point for the team's AI
func (s *TeamCoordinationStrategy) EvaluateActions(agent *Agent, game *Game)AgentAction {
    // Ensure team state is updated once per turn (e.g., by the first agent processed)
    // This requires a global flag or a mechanism in the main loop to call updateTeamState once.
    // For this example, we'll assume it's handled externally or by a "leader" agent.

    // Based on the current team state, each agent decides its individual actions using a BT
    var agentBehaviorTree Node

    switch s.CurrentTeamState {
    case TeamStateCombat:
        agentBehaviorTree = BuildCombatBT() // Build a combat-focused BT
    case TeamStateTerritoryControl:
        agentBehaviorTree = BuildTerritoryBT() // Build a territory-focused BT
    // case TeamStateDefense:
    //     agentBehaviorTree = BuildDefensiveBT() // Build a defensive BT
    case TeamStateRegroupAndHeal:
        agentBehaviorTree = BuildRegroupAndHealBT() // Build a regroup/heal BT
    default:
        agentBehaviorTree = BuildDefaultBT() // Fallback
    }

    // Evaluate the agent's specific behavior tree
    agentBehaviorTree.Evaluate(agent, game)

    // The actions are collected by the tasks within the BT directly into game.AgentActions
    // The main loop will then retrieve and format them.
    return game.AgentActions // Return actions collected by the BT
}

// BuildRegroupAndHealBT and BuildDefaultBT would be similar to BuildCombatBT/BuildTerritoryBT
func BuildRegroupAndHealBT() Node {
    return &Selector{
        Composite: Composite{
            Children:Node{
                &Sequence{
                    Composite: Composite{
                        Children:Node{
                            &CheckWetnessHigh{},
                            &TaskFleeToSafety{},
                        },
                    },
                },
                &TaskHunkerDown{},
            },
        },
    }
}

func BuildDefaultBT() Node {
    return &TaskHunkerDown{} // Simple default
}

//... (Game helper methods like CalculateTerritoryScores, IsTileControlledByMe, FindNearestEnemy, etc.)...
This comprehensive approach, combining a high-level FSM for team strategy with detailed Behavior Trees for individual agent tactics, provides a robust and flexible AI system that can adapt to the dynamic objectives of the CodinGame Summer 2025 challenge, all within the constraints of manual Go implementation.