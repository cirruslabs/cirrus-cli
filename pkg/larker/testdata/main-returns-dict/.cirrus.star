def main():
    return {
        "container": {
          "image": "debian:latest",
        },
        "env": {
          "VARIABLE_NAME": "VARIABLE_VALUE",
        },
        "task": {
            "script": [
                "printenv",
            ],
        },
    }
