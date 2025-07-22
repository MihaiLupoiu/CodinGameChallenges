#!/usr/bin/env python3

import re
import sys

def parse_captured_scenario(filename):
    """Parse a captured scenario file and extract map + initial positions"""
    
    with open(filename, 'r') as f:
        content = f.read()
    
    # Extract map dimensions - need to find the specific line with width/height
    lines = content.split('\n')
    width, height = None, None
    
    # Look for the width/height line (comes after agent data)
    for i, line in enumerate(lines):
        if 'CAPTURED INPUT:' in line:
            parts = line.replace('CAPTURED INPUT: ', '').strip().split()
            # Width/height line has exactly 2 numbers and comes after agent setup
            if len(parts) == 2 and parts[0].isdigit() and parts[1].isdigit():
                # Check if this looks like reasonable map dimensions
                w, h = int(parts[0]), int(parts[1])
                if w >= 10 and h >= 5:  # Reasonable map size
                    width, height = w, h
                    print(f"Map: {width}x{height}")
                    break
    
    if not width or not height:
        print("Could not find map dimensions")
        return
    
    # Extract map data lines - they come right after the width/height line
    map_lines = []
    found_dimensions = False
    
    for line in lines:
        if 'CAPTURED INPUT:' in line and 'Game initialization complete' not in line:
            parts = line.replace('CAPTURED INPUT: ', '').strip().split()
            
            # Check if this is the width/height line
            if len(parts) == 2 and parts[0].isdigit() and parts[1].isdigit():
                w, h = int(parts[0]), int(parts[1])
                if w >= 10 and h >= 5:  # Found our dimensions line
                    found_dimensions = True
                    continue
            
            # If we found dimensions, next lines are map data
            if found_dimensions and len(parts) > 10:  # Map lines have many coordinates
                map_lines.append(line.replace('CAPTURED INPUT: ', '').strip())
                
                if len(map_lines) >= height:
                    break
    
    # Parse map into 2D array
    game_map = [[0 for _ in range(width)] for _ in range(height)]
    
    for y, line in enumerate(map_lines[:height]):
        coords = line.split()
        for i in range(0, len(coords), 3):
            if i + 2 < len(coords):
                x, y_coord, tile_type = int(coords[i]), int(coords[i+1]), int(coords[i+2])
                if 0 <= x < width and 0 <= y_coord < height:
                    game_map[y_coord][x] = tile_type
    
    # Extract initial agent positions from TURN 1
    agent_lines = []
    turn1_started = False
    for line in lines:
        if 'CAPTURED TURN 1 AGENT INPUT:' in line:
            turn1_started = True
            agent_data = line.replace('CAPTURED TURN 1 AGENT INPUT: ', '').strip()
            agent_lines.append(agent_data)
        elif turn1_started and 'CAPTURED TURN 1 MY_AGENT_COUNT:' in line:
            break
    
    # Parse agents
    agents = []
    for agent_line in agent_lines:
        parts = agent_line.split()
        if len(parts) >= 6:
            agent_id, x, y, cooldown, bombs, wetness = map(int, parts)
            # Determine player from position (left side = player 0, right side = player 1)
            player_id = 0 if x < width // 2 else 1
            agents.append({
                'id': agent_id,
                'player': player_id,
                'x': x,
                'y': y,
                'bombs': bombs
            })
    
    return {
        'width': width,
        'height': height,
        'map': game_map,
        'agents': agents
    }

def create_scenario_file(scenario_data, output_filename):
    """Create a scenario file from parsed data"""
    
    with open(output_filename, 'w') as f:
        f.write(f"# Real captured scenario: {output_filename}\n")
        f.write(f"MAP {scenario_data['width']} {scenario_data['height']}\n")
        
        # Write map
        for row in scenario_data['map']:
            f.write(' '.join(map(str, row)) + '\n')
        
        f.write('\nAGENTS\n')
        
        # Write agents
        for agent in scenario_data['agents']:
            f.write(f"{agent['id']} {agent['player']} {agent['x']} {agent['y']}\n")

if __name__ == "__main__":
    sample_files = ['samples/sample1.txt','samples/sample2.txt','samples/sample3.txt','samples/sample4.txt','samples/sample5.txt', 'samples/sample_scenario.txt', 
                   'samples/sample_scenario2.txt', 'samples/sample_scenario3.txt']
    
    for sample_file in sample_files:
        try:
            print(f"Processing {sample_file}...")
            scenario_data = parse_captured_scenario(sample_file)
            
            if scenario_data:
                output_name = sample_file.replace('samples/', '').replace('.txt', '_real.txt')
                create_scenario_file(scenario_data, output_name)
                print(f"Created {output_name}")
                
                # Print summary
                print(f"  Map: {scenario_data['width']}x{scenario_data['height']}")
                print(f"  Agents: {len(scenario_data['agents'])}")
                player0 = sum(1 for a in scenario_data['agents'] if a['player'] == 0)
                player1 = sum(1 for a in scenario_data['agents'] if a['player'] == 1)
                print(f"  Teams: Player0={player0}, Player1={player1}")
                print()
        except Exception as e:
            print(f"Error processing {sample_file}: {e}") 