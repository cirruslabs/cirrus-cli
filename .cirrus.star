load("github.com/cirrus-templates/golang@main", "lint_task")

def main(ctx):
    return [lint_task()]
