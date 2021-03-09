load("cirrus", "changes_include")

def main(ctx):
    if changes_include('**.sh') != True:
        fail("no Shell-scripts detected in changed files")

    if changes_include('*.txt', '*.md') != True:
        fail("no text and Markdown files detected in changed files")

    if changes_include('**.yml', '**.yaml') == True:
        fail("unexpected change detected for YAML files pattern")

    return []
