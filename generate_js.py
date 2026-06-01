import re
import json

file_path = "/home/fedorae1/.gemini/tmp/gama-fit/tool-outputs/session-f091cd24-a6f7-447f-89a2-f387df95bdd8/run_shell_command__run_shell_command_1780257979559_0.txt"

with open(file_path, 'r') as f:
    content = f.read()

groups = re.split(r'ExerciseAttributeValueEnum\.([A-Z_]+)', content)

def is_front(path_str):
    nums = re.findall(r'[-+]?\d*\.\d+|\d+', path_str)
    if not nums: return True
    xs = [float(nums[i]) for i in range(0, len(nums), 2)]
    if not xs: return True
    avg_x = sum(xs) / len(xs)
    return avg_x < 270

muscle_data = {}

for i in range(1, len(groups), 2):
    group_name = groups[i].lower()
    group_content = groups[i+1]
    paths = re.findall(r'\bd="([^"]+)"', group_content)
    
    for p in paths:
        cleaned_p = " ".join(p.split())
        if not cleaned_p.upper().startswith('M'): continue
        view = "front" if is_front(cleaned_p) else "back"
        
        if group_name not in muscle_data:
            muscle_data[group_name] = {"front": [], "back": []}
        if cleaned_p not in muscle_data[group_name][view]:
            muscle_data[group_name][view].append(cleaned_p)

# Define the JS keys and which groups go into them
mapping = {
    "abs": ["abdominals"],
    "chest": ["chest"],
    "biceps": ["biceps"],
    "triceps": ["triceps"],
    "shoulders": ["shoulders"],
    "back": ["back"],
    "legs": ["quadriceps", "hamstrings", "glutes", "calves"],
    "obliques": ["obliques"],
    "forearms": ["forearms"],
    "traps": ["traps"]
}

js_paths = {}
for js_key, group_names in mapping.items():
    js_paths[js_key] = {"front": [], "back": []}
    for g_name in group_names:
        if g_name in muscle_data:
            js_paths[js_key]["front"].extend(muscle_data[g_name]["front"])
            js_paths[js_key]["back"].extend(muscle_data[g_name]["back"])

print(json.dumps(js_paths, indent=2))
