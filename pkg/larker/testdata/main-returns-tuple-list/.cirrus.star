def main():
    return [
        ("container", {"image": "debian:latest"}),
        ("env", {"VARIABLE_NAME": "VARIABLE_VALUE"}),
        ("task", {"name": "task 1", "script": ["printenv"]}),
        ("task", {"name": "task 2", "script": ['echo "task"']})
    ]
