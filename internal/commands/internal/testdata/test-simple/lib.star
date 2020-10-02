def get_tasks():
    return [
        {
            "container": {
                "image": "debian:latest",
            },
            "script": "sleep 5",
        }
    ]
