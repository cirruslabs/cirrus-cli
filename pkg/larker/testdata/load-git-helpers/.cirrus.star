load("github.com/cirrus-modules/helpers@a5e5d1649c05c40bab6c82f084b69a8d82977d96", "task", "container", "script", "always", "artifacts")

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
