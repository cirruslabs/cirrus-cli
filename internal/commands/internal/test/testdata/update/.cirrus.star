def main(ctx):
    print("some")
    print("log")
    print("contents")

    return [
        {
            "container": {
                "image": "debian:latest",
            },
            "script": "sleep 5",
        }
    ]
