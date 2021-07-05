def main(ctx):
    print("actual log line")
    return [
        {
            "container": {
                "image": "debian:latest",
            }
        }
    ]
