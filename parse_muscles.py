import re
import json

file_path = "/home/fedorae1/.gemini/tmp/gama-fit/tool-outputs/session-f091cd24-a6f7-447f-89a2-f387df95bdd8/run_shell_command__run_shell_command_1780257979559_0.txt"

with open(file_path, 'r') as f:
    content = f.read()

# Match groups
groups = re.split(r'ExerciseAttributeValueEnum\.([A-Z_]+)', content)

final_data = {}

def is_front(path_str):
    nums = re.findall(r'[-+]?\d*\.\d+|\d+', path_str)
    if not nums: return True
    xs = [float(nums[i]) for i in range(0, len(nums), 2)]
    if not xs: return True
    avg_x = sum(xs) / len(xs)
    return avg_x < 270

for i in range(1, len(groups), 2):
    group_name = groups[i].lower()
    group_content = groups[i+1]
    
    # Match ONLY d="..."
    paths = re.findall(r'\bd="([^"]+)"', group_content)
    
    for p in paths:
        cleaned_p = " ".join(p.split())
        if not cleaned_p.upper().startswith('M'):
            continue
            
        view = "front" if is_front(cleaned_p) else "back"
        
        if group_name not in final_data:
            final_data[group_name] = {"front": [], "back": []}
        
        if cleaned_p not in final_data[group_name][view]:
            final_data[group_name][view].append(cleaned_p)

# Mapping to expected JS keys
key_map = {
    "abdominals": "abs",
    "chest": "chest",
    "biceps": "biceps",
    "quadriceps": "quads",
    "hamstrings": "hamstrings",
    "glutes": "glutes",
    "calves": "calves",
    "shoulders": "shoulders",
    "obliques": "obliques",
    "forearms": "forearms",
    "back": "back",
    "traps": "traps",
    "triceps": "triceps"
}

js_muscle_paths = {}
for g_name, views in final_data.items():
    js_key = key_map.get(g_name, g_name)
    js_muscle_paths[js_key] = views

print(json.dumps(js_muscle_paths, indent=2))
