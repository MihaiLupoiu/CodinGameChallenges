package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

/**
 * Win the water fight by controlling the most territory, or out-soak your opponent!
 **/

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1000000), 1000000)
	var inputs []string

	// myId: Your player id (0 or 1)
	var myId int
	scanner.Scan()
	fmt.Sscan(scanner.Text(), &myId)

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
	}
	// width: Width of the game map
	// height: Height of the game map
	var width, height int
	scanner.Scan()
	fmt.Sscan(scanner.Text(), &width, &height)

	for i := 0; i < height; i++ {
		scanner.Scan()
		inputs = strings.Split(scanner.Text(), " ")
		for j := 0; j < width; j++ {
			// x: X coordinate, 0 is left edge
			// y: Y coordinate, 0 is top edge
			x, _ := strconv.ParseInt(inputs[3*j], 10, 32)
			_ = x
			y, _ := strconv.ParseInt(inputs[3*j+1], 10, 32)
			_ = y
			tileType, _ := strconv.ParseInt(inputs[3*j+2], 10, 32)
			_ = tileType
		}
	}
	for {
		var agentCount int
		scanner.Scan()
		fmt.Sscan(scanner.Text(), &agentCount)
		for i := 0; i < agentCount; i++ {
			// cooldown: Number of turns before this agent can shoot
			// wetness: Damage (0-100) this agent has taken
			var agentId, x, y, cooldown, splashBombs, wetness int
			scanner.Scan()
			fmt.Sscan(scanner.Text(), &agentId, &x, &y, &cooldown, &splashBombs, &wetness)
		}
		// myAgentCount: Number of alive agents controlled by you
		var myAgentCount int
		scanner.Scan()
		fmt.Sscan(scanner.Text(), &myAgentCount)

		for i := 0; i < myAgentCount; i++ {

			// fmt.Fprintln(os.Stderr, "Debug messages...")

			// One line per agent: <agentId>;<action1;action2;...> actions are "MOVE x y | SHOOT id | THROW x y | HUNKER_DOWN | MESSAGE text"
			fmt.Println("HUNKER_DOWN")
		}
	}
}
