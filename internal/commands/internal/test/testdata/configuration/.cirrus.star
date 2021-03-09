load("cirrus", "env", "changes_include")

def main(ctx):
    # Test env
    variable_name = "SOME_VARIABLE"
    expected = "some value"
    actual = env[variable_name]

    if actual != expected:
        fail("expected %s variable to be '%s', but got '%s' instead" % (variable_name, expected, actual))

    # Test changes_include
    if changes_include("**.sh") != True:
        fail("changes_include() builtin is not detecting any Shell-scripts, "+
            "however they are present in the test configuration file")

    return []
