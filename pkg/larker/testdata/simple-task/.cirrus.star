def main():
    return [
        {
            "container": {
                "image": "debian:latest",
            },
            "script": [
                "printenv",
            ],
        },
    ]
