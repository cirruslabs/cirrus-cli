load("github.com/cirrus-templates/golang@a4e91ca453a4ade8f41013fca0888536d680f51d", "detect_tasks")

def main(ctx):
    return [detect_tasks()]
