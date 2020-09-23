load("cirrus", "env")

def main(ctx):
    # Check that at least one base variable is present and has expected value
    variable_name = "CIRRUS_TAG"
    actual_value = env.get("CIRRUS_TAG")
    expected_value = "v0.1.0"
    if actual_value != expected_value:
        fail("expected %s variable to be %s, but got %s instead" % (variable_name, expected_value, actual_value))

    # Check that user variable is present and has expected value
    variable_name = "USER_VARIABLE"
    actual_value = env.get("USER_VARIABLE")
    expected_value = "user variable value"
    if actual_value != expected_value:
        fail("expected %s variable to be %s, but got %s instead" % (variable_name, expected_value, actual_value))

    # Return dummy configuration
    return [
        {
            "container": {
                "image": "debian:latest",
            },
        },
    ]
