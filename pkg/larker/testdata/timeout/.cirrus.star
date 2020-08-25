def main(ctx):
    ten_millions = 10000000
    annoying_monologue = ""

    for i in range(ten_millions):
        # Separate phrases with space
        if i != 0:
            annoying_monologue += " "

        annoying_monologue += "Are we there yet?"

    print(annoying_monologue)

    # Return no tasks
    return []
