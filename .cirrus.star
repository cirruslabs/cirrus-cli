load("github.com/cirrus-templates/golang", "lint_task")

def main(ctx):
    return [lint_task()]
