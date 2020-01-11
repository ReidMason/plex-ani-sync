import json
import os


def load_json(filepath: str):
    if not os.path.exists(filepath):
        save_json({}, filepath)

    with open(filepath, 'r', encoding = 'utf-8') as f:
        return json.load(f)


def save_json(data, filepath: str):
    with open(filepath, 'w', encoding = 'utf-8') as f:
        json.dump(data, f)
