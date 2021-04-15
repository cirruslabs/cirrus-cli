load("github.com/cirrus-templates/helpers@a5e5d1649c05c40bab6c82f084b69a8d82977d96", "task", "container", "script", "always", "artifacts")

# Ensure that we can load by hash and a branch name
load("github.com/cirrus-templates/graphql@1f72f65b1d0aaa6052f2401cad2701e2bd3bd12d", "rerun_task_from_hash")
load("github.com/cirrus-templates/graphql@initial-queries", "rerun_task_from_branch")

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
