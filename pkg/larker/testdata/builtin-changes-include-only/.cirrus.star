load("cirrus", "changes_include_only")

def main(ctx):
    if changes_include_only('**.go', 'go.mod') != True:
        fail("changes_include_only() includes all files and should yield a positive result")

    if changes_include_only('**.go') != False:
        fail("changes_include_only() does not include all files and should yield a negative result")

    return []
