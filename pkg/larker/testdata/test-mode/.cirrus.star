load("cirrus", "is_test")

def main(ctx):
    if is_test:
        print("testing mode enabled")
    else:
        print("testing mode disabled")

    return []
