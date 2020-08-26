load("core.star", "task", "container", "script", "always", "artifacts")

def main(ctx):
    return [
        task(
            name="Lint",
            instance=container("golangci/golangci-lint:latest", cpu=1.0, memory=512),
            env={
                "STARLARK": True
            },
            instructions=[
                script("lint", "echo $STARLARK", "golangci-lint run -v --out-format json > golangci.json"),
                always(
                    artifacts("report", "golangci.json", type="text/json", format="golangci")
                )
            ]
        )
    ]
