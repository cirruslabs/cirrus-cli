load("cirrus", "env")

def main(ctx):
    variable_name = "SOME_VARIABLE"
    expected = "some value"
    actual = env[variable_name]

    if actual != expected:
        fail("expected %s variable to be '%s', but got '%s' instead" % (variable_name, expected, actual))

    return []
