load("cirrus", "env")

def main(ctx):
    test_env()

    return []

def test_env():
    variable_name = "SOME_VARIABLE"
    expected = "some value"
    got = env.get(variable_name)
    if got == None or got != expected:
        fail("expected %s environment variable to be equal to %s, but got %s instead" % (variable_name, expected, got))
