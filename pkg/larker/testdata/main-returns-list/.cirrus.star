def main():
    return [
        {
            "name": "task 1",
            "container": {"image": "debian:latest"},
            "script": "printenv",
        },
        {
            "name": "task 2",
            "container": {"image": "debian:latest"},
            "script": "echo 'hello'",
        }
    ]
